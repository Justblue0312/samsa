package story_report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/infras/cache"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Report CRUD
	Create(ctx context.Context, report *sqlc.StoryReport) (*sqlc.StoryReport, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryReport, error)
	GetByStoryAndReporter(ctx context.Context, storyID, reporterID uuid.UUID) (*sqlc.StoryReport, error)
	Update(ctx context.Context, report *sqlc.StoryReport) (*sqlc.StoryReport, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// Status updates
	UpdateStatus(ctx context.Context, id uuid.UUID, status sqlc.ReportStatus, resolvedBy *uuid.UUID) (*sqlc.StoryReport, error)

	// List operations
	ListByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryReport, int64, error)
	ListByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int32) ([]sqlc.StoryReport, int64, error)
	ListPending(ctx context.Context, limit, offset int32) ([]sqlc.StoryReport, int64, error)
	ListAll(ctx context.Context, filter *ReportFilter) ([]sqlc.StoryReport, int64, error)

	// Statistics
	CountByStory(ctx context.Context, storyID uuid.UUID) (int64, error)
	CountPending(ctx context.Context) (int64, error)
}

type repository struct {
	q     *sqlc.Queries
	db    sqlc.DBTX
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(db sqlc.DBTX, cfg *config.Config, cache *cache.Client) Repository {
	return &repository{
		q:     sqlc.New(db),
		db:    db,
		cfg:   cfg,
		cache: cache,
	}
}

func buildReportKey(id uuid.UUID) string {
	return fmt.Sprintf("story_report:%s", id.String())
}

func (r *repository) cacheReport(ctx context.Context, report *sqlc.StoryReport) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildReportKey(report.ID),
		Value: report,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

// Create creates a new report
func (r *repository) Create(ctx context.Context, report *sqlc.StoryReport) (*sqlc.StoryReport, error) {
	result, err := r.q.CreateStoryReport(ctx, sqlc.CreateStoryReportParams{
		StoryID:     report.StoryID,
		ChapterID:   report.ChapterID,
		ReporterID:  report.ReporterID,
		Title:       report.Title,
		Description: report.Description,
		Status:      report.Status,
	})
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	r.cacheReport(ctx, &result)
	return &result, nil
}

// GetByID retrieves a report by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryReport, error) {
	if r.cfg.Cache.EnableCache {
		key := buildReportKey(id)
		var report sqlc.StoryReport
		if err := r.cache.Get(ctx, key, &report); err == nil {
			return &report, nil
		}
	}

	report, err := r.q.GetStoryReportByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheReport(ctx, &report)
	return &report, nil
}

// GetByStoryAndReporter retrieves a report by story and reporter
func (r *repository) GetByStoryAndReporter(ctx context.Context, storyID, reporterID uuid.UUID) (*sqlc.StoryReport, error) {
	report, err := r.q.GetStoryReportByStoryAndReporter(ctx, sqlc.GetStoryReportByStoryAndReporterParams{
		StoryID:    storyID,
		ReporterID: reporterID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByStoryAndReporter: %w", err)
	}

	return &report, nil
}

// Update updates an existing report
func (r *repository) Update(ctx context.Context, report *sqlc.StoryReport) (*sqlc.StoryReport, error) {
	if r.cfg.Cache.EnableCache {
		_ = r.cache.Delete(ctx, buildReportKey(report.ID))
	}

	result, err := r.q.UpdateStoryReport(ctx, sqlc.UpdateStoryReportParams{
		ID:          report.ID,
		Title:       report.Title,
		Description: report.Description,
		Status:      report.Status,
		IsResolved:  report.IsResolved,
		ResolvedAt:  report.ResolvedAt,
		ResolvedBy:  report.ResolvedBy,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.Update: %w", err)
	}

	r.cacheReport(ctx, &result)
	return &result, nil
}

// Delete deletes a report by ID
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.cfg.Cache.EnableCache {
		_ = r.cache.Delete(ctx, buildReportKey(id))
	}

	err := r.q.DeleteStoryReport(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a report
func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status sqlc.ReportStatus, resolvedBy *uuid.UUID) (*sqlc.StoryReport, error) {
	now := time.Now()
	isResolved := status == sqlc.ReportStatusResolved

	params := sqlc.UpdateStoryReportStatusParams{
		ID:         id,
		Status:     sqlc.NullReportStatus{ReportStatus: status, Valid: true},
		IsResolved: &isResolved,
	}

	if isResolved {
		params.ResolvedAt = &now
		if resolvedBy != nil {
			params.ResolvedBy = resolvedBy
		}
	}

	result, err := r.q.UpdateStoryReportStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.UpdateStatus: %w", err)
	}

	r.cacheReport(ctx, &result)
	return &result, nil
}

// ListByStory lists reports for a story
func (r *repository) ListByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryReport, int64, error) {
	reports, err := r.q.ListStoryReportsByStory(ctx, sqlc.ListStoryReportsByStoryParams{
		StoryID: storyID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStory: %w", err)
	}

	count, err := r.q.CountStoryReportsByStory(ctx, storyID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStory count: %w", err)
	}

	return reports, count, nil
}

// ListByReporter lists reports by a reporter
func (r *repository) ListByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int32) ([]sqlc.StoryReport, int64, error) {
	reports, err := r.q.ListStoryReportsByReporter(ctx, sqlc.ListStoryReportsByReporterParams{
		ReporterID: reporterID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByReporter: %w", err)
	}

	count, err := r.q.CountStoryReportsByReporter(ctx, reporterID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByReporter count: %w", err)
	}

	return reports, count, nil
}

// ListPending lists all pending reports
func (r *repository) ListPending(ctx context.Context, limit, offset int32) ([]sqlc.StoryReport, int64, error) {
	reports, err := r.q.ListPendingStoryReports(ctx, sqlc.ListPendingStoryReportsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListPending: %w", err)
	}

	count, err := r.q.CountPendingStoryReports(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListPending count: %w", err)
	}

	return reports, count, nil
}

// ListAll lists all reports with filters
func (r *repository) ListAll(ctx context.Context, filter *ReportFilter) ([]sqlc.StoryReport, int64, error) {
	// Build dynamic query based on filter
	args := sqlc.ListStoryReportsWithFiltersParams{
		Limit:  filter.Limit,
		Offset: filter.GetOffset(),
	}

	if filter.StoryID != nil {
		args.StoryID = filter.StoryID
	}
	if filter.ReporterID != nil {
		args.ReporterID = filter.ReporterID
	}
	if filter.Status != nil {
		args.Status = sqlc.NullReportStatus{ReportStatus: *filter.Status, Valid: true}
	}
	if filter.IsResolved != nil {
		args.IsResolved = filter.IsResolved
	}

	reports, err := r.q.ListStoryReportsWithFilters(ctx, args)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListAll: %w", err)
	}

	// Get total count
	countArgs := sqlc.CountStoryReportsWithFiltersParams{}
	if filter.StoryID != nil {
		countArgs.StoryID = filter.StoryID
	}
	if filter.ReporterID != nil {
		countArgs.ReporterID = filter.ReporterID
	}
	if filter.Status != nil {
		countArgs.Status = sqlc.NullReportStatus{ReportStatus: *filter.Status, Valid: true}
	}
	if filter.IsResolved != nil {
		countArgs.IsResolved = filter.IsResolved
	}

	count, err := r.q.CountStoryReportsWithFilters(ctx, countArgs)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListAll count: %w", err)
	}

	return reports, count, nil
}

// CountByStory counts reports for a story
func (r *repository) CountByStory(ctx context.Context, storyID uuid.UUID) (int64, error) {
	count, err := r.q.CountStoryReportsByStory(ctx, storyID)
	if err != nil {
		return 0, fmt.Errorf("repository.CountByStory: %w", err)
	}
	return count, nil
}

// CountPending counts all pending reports
func (r *repository) CountPending(ctx context.Context) (int64, error) {
	count, err := r.q.CountPendingStoryReports(ctx)
	if err != nil {
		return 0, fmt.Errorf("repository.CountPending: %w", err)
	}
	return count, nil
}

package story_report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Report CRUD
	CreateReport(ctx context.Context, reporterID uuid.UUID, req CreateReportRequest) (*ReportResponse, error)
	GetReport(ctx context.Context, id uuid.UUID) (*ReportResponse, error)
	UpdateReport(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool, req UpdateReportRequest) (*ReportResponse, error)
	DeleteReport(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool) error

	// Status management
	ResolveReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID, notes *string) (*ReportResponse, error)
	RejectReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID, reason *string) (*ReportResponse, error)
	ArchiveReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID) (*ReportResponse, error)

	// List operations
	ListReports(ctx context.Context, filter *ReportFilter, isModerator bool, userID uuid.UUID) ([]ReportResponse, int64, error)
	ListReportsByStory(ctx context.Context, storyID uuid.UUID, filter *ReportFilter) ([]ReportResponse, int64, error)
	ListPendingReports(ctx context.Context, filter *ReportFilter) ([]ReportResponse, int64, error)

	// Statistics
	GetReportCount(ctx context.Context, storyID uuid.UUID) (int64, error)
	GetPendingReportCount(ctx context.Context) (int64, error)
}

type usecase struct {
	repo     Repository
	notifier Notifier
}

func NewUseCase(repo Repository, notifier Notifier) UseCase {
	return &usecase{
		repo:     repo,
		notifier: notifier,
	}
}

// CreateReport creates a new report
func (uc *usecase) CreateReport(ctx context.Context, reporterID uuid.UUID, req CreateReportRequest) (*ReportResponse, error) {
	// Check if user already reported this story
	existing, err := uc.repo.GetByStoryAndReporter(ctx, req.StoryID, reporterID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("usecase.CreateReport: %w", err)
	}

	if existing != nil && (existing.IsResolved == nil || !*existing.IsResolved) {
		// User already has a pending report for this story
		return nil, errors.New("usecase.CreateReport: user already has a pending report for this story")
	}

	report := &sqlc.StoryReport{
		ID:          uuid.New(),
		StoryID:     req.StoryID,
		ChapterID:   req.ChapterID,
		ReporterID:  reporterID,
		Title:       req.Title,
		Description: req.Description,
		Status:      sqlc.NullReportStatus{ReportStatus: sqlc.ReportStatusPending, Valid: true},
		IsResolved:  boolPtr(false),
		ResolvedAt:  nil,
		ResolvedBy:  nil,
	}

	created, err := uc.repo.Create(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("usecase.CreateReport: %w", err)
	}

	// Notify moderators about new report
	if uc.notifier != nil {
		_ = uc.notifier.NotifyNewReport(ctx, created.ID, created.StoryID, reporterID)
	}

	return ToReportResponse(created), nil
}

// GetReport retrieves a report by ID
func (uc *usecase) GetReport(ctx context.Context, id uuid.UUID) (*ReportResponse, error) {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetReport: %w", err)
	}
	return ToReportResponse(report), nil
}

// UpdateReport updates a report (moderator only or reporter for description)
func (uc *usecase) UpdateReport(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool, req UpdateReportRequest) (*ReportResponse, error) {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.UpdateReport: %w", err)
	}

	// Check permissions
	if !isModerator && report.ReporterID != userID {
		return nil, ErrNotReporter
	}

	if req.Status != nil && !isModerator {
		return nil, ErrNotModerator
	}

	now := time.Now()
	if req.Status != nil {
		report.Status = sqlc.NullReportStatus{ReportStatus: *req.Status, Valid: true}
		report.UpdatedAt = &now
	}
	if req.Description != nil {
		report.Description = req.Description
		report.UpdatedAt = &now
	}

	updated, err := uc.repo.Update(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("usecase.UpdateReport: %w", err)
	}

	return ToReportResponse(updated), nil
}

// DeleteReport deletes a report
func (uc *usecase) DeleteReport(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool) error {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("usecase.DeleteReport: %w", err)
	}

	// Only moderator or reporter can delete
	if !isModerator && report.ReporterID != userID {
		return ErrNotReporter
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usecase.DeleteReport: %w", err)
	}

	return nil
}

// ResolveReport resolves a report (moderator only)
func (uc *usecase) ResolveReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID, notes *string) (*ReportResponse, error) {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.ResolveReport: %w", err)
	}

	if report.IsResolved != nil && *report.IsResolved {
		return nil, errors.New("usecase.ResolveReport: report is already resolved")
	}

	updated, err := uc.repo.UpdateStatus(ctx, id, sqlc.ReportStatusResolved, &moderatorID)
	if err != nil {
		return nil, fmt.Errorf("usecase.ResolveReport: %w", err)
	}

	// Notify reporter about resolution
	if uc.notifier != nil {
		_ = uc.notifier.NotifyReportResolved(ctx, updated.ID, updated.ReporterID, notes)
	}

	return ToReportResponse(updated), nil
}

// RejectReport rejects a report (moderator only)
func (uc *usecase) RejectReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID, reason *string) (*ReportResponse, error) {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.RejectReport: %w", err)
	}

	if report.IsResolved != nil && *report.IsResolved {
		return nil, errors.New("usecase.RejectReport: report is already resolved")
	}

	updated, err := uc.repo.UpdateStatus(ctx, id, sqlc.ReportStatusRejected, &moderatorID)
	if err != nil {
		return nil, fmt.Errorf("usecase.RejectReport: %w", err)
	}

	// Notify reporter about rejection
	if uc.notifier != nil {
		_ = uc.notifier.NotifyReportRejected(ctx, updated.ID, updated.ReporterID, reason)
	}

	return ToReportResponse(updated), nil
}

// ArchiveReport archives a report (moderator only)
func (uc *usecase) ArchiveReport(ctx context.Context, id uuid.UUID, moderatorID uuid.UUID) (*ReportResponse, error) {
	report, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.ArchiveReport: %w", err)
	}

	if report.IsResolved != nil && *report.IsResolved {
		return nil, errors.New("usecase.ArchiveReport: report is already resolved")
	}

	updated, err := uc.repo.UpdateStatus(ctx, id, sqlc.ReportStatusArchived, &moderatorID)
	if err != nil {
		return nil, fmt.Errorf("usecase.ArchiveReport: %w", err)
	}

	return ToReportResponse(updated), nil
}

// ListReports lists reports with filters
func (uc *usecase) ListReports(ctx context.Context, filter *ReportFilter, isModerator bool, userID uuid.UUID) ([]ReportResponse, int64, error) {
	if filter == nil {
		filter = &ReportFilter{}
	}

	// Non-moderators can only see their own reports
	if !isModerator {
		filter.ReporterID = &userID
	}

	reports, total, err := uc.repo.ListAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListReports: %w", err)
	}

	result := make([]ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = *ToReportResponse(&r)
	}

	return result, total, nil
}

// ListReportsByStory lists reports for a story
func (uc *usecase) ListReportsByStory(ctx context.Context, storyID uuid.UUID, filter *ReportFilter) ([]ReportResponse, int64, error) {
	if filter == nil {
		filter = &ReportFilter{}
	}

	reports, total, err := uc.repo.ListByStory(ctx, storyID, filter.Limit, filter.GetOffset())
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListReportsByStory: %w", err)
	}

	result := make([]ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = *ToReportResponse(&r)
	}

	return result, total, nil
}

// ListPendingReports lists all pending reports
func (uc *usecase) ListPendingReports(ctx context.Context, filter *ReportFilter) ([]ReportResponse, int64, error) {
	if filter == nil {
		filter = &ReportFilter{}
	}

	reports, total, err := uc.repo.ListPending(ctx, filter.Limit, filter.GetOffset())
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListPendingReports: %w", err)
	}

	result := make([]ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = *ToReportResponse(&r)
	}

	return result, total, nil
}

// GetReportCount returns the count of reports for a story
func (uc *usecase) GetReportCount(ctx context.Context, storyID uuid.UUID) (int64, error) {
	count, err := uc.repo.CountByStory(ctx, storyID)
	if err != nil {
		return 0, fmt.Errorf("usecase.GetReportCount: %w", err)
	}
	return count, nil
}

// GetPendingReportCount returns the count of pending reports
func (uc *usecase) GetPendingReportCount(ctx context.Context) (int64, error) {
	count, err := uc.repo.CountPending(ctx)
	if err != nil {
		return 0, fmt.Errorf("usecase.GetPendingReportCount: %w", err)
	}
	return count, nil
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

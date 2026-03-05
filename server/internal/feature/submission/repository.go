package submission

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/infras/cache"
	"github.com/justblue/samsa/pkg/queryparam"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	GetByExposeID(ctx context.Context, exposeID string) (*sqlc.Submission, error)
	Create(ctx context.Context, submission *sqlc.Submission) (*sqlc.Submission, error)
	Update(ctx context.Context, submission *sqlc.Submission) (*sqlc.Submission, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status sqlc.SubmissionStatus) (*sqlc.Submission, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	List(ctx context.Context, f *SubmissionFilter) ([]*sqlc.Submission, int64, error)
	GetByRequesterID(ctx context.Context, requesterID uuid.UUID, limit, offset int32) ([]*sqlc.Submission, error)
	GetByApproverID(ctx context.Context, approverID uuid.UUID, limit, offset int32) ([]*sqlc.Submission, error)
	GetAvailable(ctx context.Context, limit, offset int32) ([]*sqlc.Submission, int64, error)
	GetTimeoutedCandidates(ctx context.Context, days int) ([]*sqlc.Submission, error)
	BulkMarkTimeouted(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error)
	GetArchiveCandidates(ctx context.Context, days int) ([]*sqlc.Submission, error)
	BulkMarkArchived(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error)
	// SLA tracking
	GetSubmissionsExceedingSLA(ctx context.Context, slaHours int) ([]*sqlc.Submission, error)
	CountSubmissionsExceedingSLA(ctx context.Context, slaHours int) (int64, error)
	GetSLAComplianceStats(ctx context.Context, slaHours int) (*sqlc.GetSLAComplianceStatsRow, error)
	GetAverageProcessingTime(ctx context.Context, days int) (float64, error)
	GetSubmissionsBySLAStatus(ctx context.Context, includeCompliant bool, slaHours int, limit, offset int32) ([]*sqlc.Submission, error)
	GetPendingDuration(ctx context.Context, id uuid.UUID) (int, error)
	BulkUpdateSLABreach(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error)
}

type repository struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(pool *pgxpool.Pool, q *sqlc.Queries, cfg *config.Config, cache *cache.Client) Repository {
	return &repository{
		pool:  pool,
		q:     q,
		cfg:   cfg,
		cache: cache,
	}
}

func buildSubmissionKey(id uuid.UUID) string {
	return fmt.Sprintf("submission:%s", id.String())
}

func (r *repository) cacheSubmission(ctx context.Context, submission *sqlc.Submission) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildSubmissionKey(submission.ID),
		Value: submission,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id uuid.UUID) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildSubmissionKey(id))
}

// Create implements [Repository].
func (r *repository) Create(ctx context.Context, submission *sqlc.Submission) (*sqlc.Submission, error) {
	result, err := r.q.CreateSubmission(ctx, sqlc.CreateSubmissionParams{
		RequesterID: submission.RequesterID,
		ApproverID:  submission.ApproverID,
		ApprovedAt:  submission.ApprovedAt,
		Message:     submission.Message,
		Title:       submission.Title,
		Type:        submission.Type,
		IsDeleted:   submission.IsDeleted,
		Status:      submission.Status,
		Context:     submission.Context,
		CreatedAt:   submission.CreatedAt,
		UpdatedAt:   submission.UpdatedAt,
		DeletedAt:   submission.DeletedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	r.cacheSubmission(ctx, &result)
	return &result, nil
}

// GetByID implements [Repository].
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	if r.cfg.Cache.EnableCache {
		var submission sqlc.Submission
		if err := r.cache.Get(ctx, buildSubmissionKey(id), &submission); err == nil {
			return &submission, nil
		}
	}

	submission, err := r.q.GetSubmissionByID(ctx, sqlc.GetSubmissionByIDParams{
		ID:        id,
		IsDeleted: false,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheSubmission(ctx, &submission)
	return &submission, nil
}

// GetByExposeID implements [Repository].
func (r *repository) GetByExposeID(ctx context.Context, exposeID string) (*sqlc.Submission, error) {
	submission, err := r.q.GetSubmissionByExposeID(ctx, exposeID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByExposeID: %w", err)
	}
	return &submission, nil
}

// Update implements [Repository].
func (r *repository) Update(ctx context.Context, submission *sqlc.Submission) (*sqlc.Submission, error) {
	r.invalidateCache(ctx, submission.ID)

	result, err := r.q.UpdateSubmission(ctx, sqlc.UpdateSubmissionParams{
		ID:          submission.ID,
		RequesterID: submission.RequesterID,
		ApproverID:  submission.ApproverID,
		ApprovedAt:  submission.ApprovedAt,
		Message:     submission.Message,
		Title:       submission.Title,
		Type:        submission.Type,
		IsDeleted:   submission.IsDeleted,
		Status:      submission.Status,
		Context:     submission.Context,
		DeletedAt:   submission.DeletedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.Update: %w", err)
	}

	r.cacheSubmission(ctx, &result)
	return &result, nil
}

// UpdateStatus implements [Repository].
func (r *repository) UpdateStatus(ctx context.Context, id uuid.UUID, status sqlc.SubmissionStatus) (*sqlc.Submission, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.UpdateSubmissionStatus(ctx, sqlc.UpdateSubmissionStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.UpdateStatus: %w", err)
	}

	r.cacheSubmission(ctx, &result)
	return &result, nil
}

// Delete implements [Repository].
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	r.invalidateCache(ctx, id)
	if err := r.q.DeleteSubmission(ctx, id); err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

// SoftDelete implements [Repository].
func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.SoftDeleteSubmission(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("repository.SoftDelete: %w", err)
	}
	return &result, nil
}

// List implements [Repository].
func (r *repository) List(ctx context.Context, f *SubmissionFilter) ([]*sqlc.Submission, int64, error) {
	orderBy := f.ToSQL()
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	args := []any{}
	argIndex := 1

	query := `SELECT id, requester_id, approver_id, approved_at, message, title, type, context, is_deleted, created_at, updated_at, deleted_at, status, expose_id, search_vector FROM submission WHERE deleted_at IS NULL`

	// Filter by IDs
	if len(f.IDs) > 0 {
		placeholders := make([]string, len(f.IDs))
		for i, id := range f.IDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND id IN (%s)", strings.Join(placeholders, ","))
	}

	// Filter by RequesterIDs
	if len(f.RequesterIDs) > 0 {
		placeholders := make([]string, len(f.RequesterIDs))
		for i, id := range f.RequesterIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND requester_id IN (%s)", strings.Join(placeholders, ","))
	}

	// Filter by ApproverIDs
	if len(f.ApproverIDs) > 0 {
		placeholders := make([]string, len(f.ApproverIDs))
		for i, id := range f.ApproverIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		query += fmt.Sprintf(" AND approver_id IN (%s)", strings.Join(placeholders, ","))
	}

	// Filter by Type
	if f.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, f.Type)
		argIndex++
	}

	// Filter by Status (direct enum comparison)
	if f.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, f.Status)
		argIndex++
	}

	// Filter by ExposeID
	if f.ExposeID != "" {
		query += fmt.Sprintf(" AND expose_id = $%d", argIndex)
		args = append(args, f.ExposeID)
		argIndex++
	}

	// Filter by Title (ILIKE)
	if f.Title != "" {
		query += fmt.Sprintf(" AND title ILIKE $%d", argIndex)
		args = append(args, "%"+f.Title+"%")
		argIndex++
	}

	// Full-text search via tsvector
	if f.SearchQuery != nil && *f.SearchQuery != "" {
		query += fmt.Sprintf(" AND search_vector @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, *f.SearchQuery)
		argIndex++
	}

	// Filter by CreatedAt
	if f.CreatedAt.Has(queryparam.OpGte) {
		if values := f.CreatedAt.Values(); len(values) > 0 {
			query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, values[0])
			argIndex++
		}
	}
	if f.CreatedAt.Has(queryparam.OpLte) {
		if values := f.CreatedAt.Values(); len(values) > 0 {
			query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, values[0])
			argIndex++
		}
	}

	countQuery := `SELECT COUNT(*) FROM (` + strings.Replace(query, "SELECT id, requester_id, approver_id, approved_at, message, title, type, context, is_deleted, created_at, updated_at, deleted_at, status, expose_id, search_vector", "SELECT 1", 1) + `) AS count_subquery`

	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argIndex, argIndex+1)
	args = append(args, f.GetLimit(), f.GetOffset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.List query: %w", err)
	}
	defer rows.Close()

	var submissions []*sqlc.Submission
	for rows.Next() {
		var s sqlc.Submission
		if err := rows.Scan(
			&s.ID, &s.RequesterID, &s.ApproverID, &s.ApprovedAt,
			&s.Message, &s.Title, &s.Type, &s.Context,
			&s.IsDeleted, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
			&s.Status, &s.ExposeID, &s.SearchVector,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.List scan: %w", err)
		}
		submissions = append(submissions, &s)
		r.cacheSubmission(ctx, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository.List rows: %w", err)
	}

	// Count
	countArgs := args[:len(args)-2]
	countRows, err := r.pool.Query(ctx, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.List count: %w", err)
	}
	defer countRows.Close()

	var total int64
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("repository.List count scan: %w", err)
		}
	}

	return submissions, total, nil
}

// GetByRequesterID implements [Repository].
func (r *repository) GetByRequesterID(ctx context.Context, requesterID uuid.UUID, limit, offset int32) ([]*sqlc.Submission, error) {
	submissions, err := r.q.GetSubmissionsByRequesterID(ctx, sqlc.GetSubmissionsByRequesterIDParams{
		RequesterID: requesterID,
		RowLimit:    limit,
		RowOffset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetByRequesterID: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
		r.cacheSubmission(ctx, &submissions[i])
	}
	return result, nil
}

// GetByApproverID implements [Repository].
func (r *repository) GetByApproverID(ctx context.Context, approverID uuid.UUID, limit, offset int32) ([]*sqlc.Submission, error) {
	submissions, err := r.q.GetSubmissionsByApproverID(ctx, sqlc.GetSubmissionsByApproverIDParams{
		ApproverID: &approverID,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetByApproverID: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
		r.cacheSubmission(ctx, &submissions[i])
	}
	return result, nil
}

// GetAvailable implements [Repository].
func (r *repository) GetAvailable(ctx context.Context, limit, offset int32) ([]*sqlc.Submission, int64, error) {
	submissions, err := r.q.GetAvailableSubmissions(ctx, sqlc.GetAvailableSubmissionsParams{
		RowLimit:  limit,
		RowOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.GetAvailable: %w", err)
	}

	totalCount, err := r.q.CountAvailableSubmissions(ctx, false)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.GetAvailable count: %w", err)
	}

	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
	}
	return result, totalCount, nil
}

// GetTimeoutedCandidates implements [Repository].
func (r *repository) GetTimeoutedCandidates(ctx context.Context, days int) ([]*sqlc.Submission, error) {
	dayStr := fmt.Sprintf("%d", days)
	submissions, err := r.q.GetTimeoutedCandidates(ctx, sqlc.GetTimeoutedCandidatesParams{
		Column1:   &dayStr,
		IsDeleted: false,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetTimeoutedCandidates: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
	}
	return result, nil
}

// BulkMarkTimeouted implements [Repository].
func (r *repository) BulkMarkTimeouted(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error) {
	if len(ids) == 0 {
		return []*sqlc.Submission{}, nil
	}
	submissions, err := r.q.BulkMarkTimeouted(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("repository.BulkMarkTimeouted: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
		r.cacheSubmission(ctx, &submissions[i])
	}
	return result, nil
}

// GetArchiveCandidates implements [Repository].
func (r *repository) GetArchiveCandidates(ctx context.Context, days int) ([]*sqlc.Submission, error) {
	dayStr := fmt.Sprintf("%d", days)
	submissions, err := r.q.GetArchiveCandidates(ctx, sqlc.GetArchiveCandidatesParams{
		Days:      &dayStr,
		IsDeleted: false,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetArchiveCandidates: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
	}
	return result, nil
}

// BulkMarkArchived implements [Repository].
func (r *repository) BulkMarkArchived(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error) {
	if len(ids) == 0 {
		return []*sqlc.Submission{}, nil
	}
	submissions, err := r.q.BulkMarkArchived(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("repository.BulkMarkArchived: %w", err)
	}
	result := make([]*sqlc.Submission, len(submissions))
	for i := range submissions {
		result[i] = &submissions[i]
		r.invalidateCache(ctx, submissions[i].ID)
	}
	return result, nil
}

// GetSubmissionsExceedingSLA implements [Repository].
func (r *repository) GetSubmissionsExceedingSLA(ctx context.Context, slaHours int) ([]*sqlc.Submission, error) {
	slaHoursStr := fmt.Sprintf("%d", slaHours)
	rows, err := r.q.GetSubmissionsExceedingSLA(ctx, &slaHoursStr)
	if err != nil {
		return nil, fmt.Errorf("repository.GetSubmissionsExceedingSLA: %w", err)
	}
	result := make([]*sqlc.Submission, len(rows))
	for i := range rows {
		result[i] = &rows[i]
		r.invalidateCache(ctx, rows[i].ID)
	}
	return result, nil
}

// CountSubmissionsExceedingSLA implements [Repository].
func (r *repository) CountSubmissionsExceedingSLA(ctx context.Context, slaHours int) (int64, error) {
	slaHoursStr := fmt.Sprintf("%d", slaHours)
	return r.q.CountSubmissionsExceedingSLA(ctx, &slaHoursStr)
}

// GetSLAComplianceStats implements [Repository].
func (r *repository) GetSLAComplianceStats(ctx context.Context, slaHours int) (*sqlc.GetSLAComplianceStatsRow, error) {
	slaHoursStr := fmt.Sprintf("%d", slaHours)
	stats, err := r.q.GetSLAComplianceStats(ctx, &slaHoursStr)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetAverageProcessingTime implements [Repository].
func (r *repository) GetAverageProcessingTime(ctx context.Context, days int) (float64, error) {
	daysStr := fmt.Sprintf("%d", days)
	return r.q.GetAverageProcessingTime(ctx, &daysStr)
}

// GetSubmissionsBySLAStatus implements [Repository].
func (r *repository) GetSubmissionsBySLAStatus(ctx context.Context, includeCompliant bool, slaHours int, limit, offset int32) ([]*sqlc.Submission, error) {
	slaHoursStr := fmt.Sprintf("%d", slaHours)
	rows, err := r.q.GetSubmissionsBySLAStatus(ctx, sqlc.GetSubmissionsBySLAStatusParams{
		Limit:            limit,
		Offset:           offset,
		IncludeCompliant: includeCompliant,
		SlaHours:         &slaHoursStr,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetSubmissionsBySLAStatus: %w", err)
	}
	result := make([]*sqlc.Submission, len(rows))
	for i := range rows {
		result[i] = &rows[i]
		r.invalidateCache(ctx, rows[i].ID)
	}
	return result, nil
}

// GetPendingDuration implements [Repository].
func (r *repository) GetPendingDuration(ctx context.Context, id uuid.UUID) (int, error) {
	duration, err := r.q.GetPendingDuration(ctx, id)
	return int(duration), err
}

// BulkUpdateSLABreach implements [Repository].
func (r *repository) BulkUpdateSLABreach(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error) {
	rows, err := r.q.BulkUpdateSLABreach(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("repository.BulkUpdateSLABreach: %w", err)
	}
	result := make([]*sqlc.Submission, len(rows))
	for i := range rows {
		result[i] = &rows[i]
		r.invalidateCache(ctx, rows[i].ID)
	}
	return result, nil
}

// scanSubmission scans a raw query row into a Submission.
// Use only for hand-crafted queries that exactly match this column order.
func scanSubmission(dest *sqlc.Submission, scan func(...any) error) error {
	return scan(
		&dest.ID, &dest.RequesterID, &dest.ApproverID, &dest.ApprovedAt,
		&dest.Message, &dest.Title, &dest.Type, &dest.Context,
		&dest.IsDeleted, &dest.CreatedAt, &dest.UpdatedAt, &dest.DeletedAt,
		&dest.Status, &dest.ExposeID, &dest.SearchVector,
	)
}

// ensure time is imported (used in interface signature).
var _ = time.Time{}

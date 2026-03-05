package submission_assignment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/infras/cache"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	Create(ctx context.Context, assignment *sqlc.SubmissionAssignment) (*sqlc.SubmissionAssignment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.SubmissionAssignment, error)
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*sqlc.SubmissionAssignment, error)
	GetByAssignedTo(ctx context.Context, assignedTo uuid.UUID) ([]*sqlc.SubmissionAssignment, error)
	GetByAssignedBy(ctx context.Context, assignedBy uuid.UUID) ([]*sqlc.SubmissionAssignment, error)
	Update(ctx context.Context, assignment *sqlc.SubmissionAssignment) (*sqlc.SubmissionAssignment, error)
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

func buildAssignmentKey(id uuid.UUID) string {
	return fmt.Sprintf("submission_assignment:%s", id.String())
}

func (r *repository) cacheAssignment(ctx context.Context, assignment *sqlc.SubmissionAssignment) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildAssignmentKey(assignment.ID),
		Value: assignment,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id uuid.UUID) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildAssignmentKey(id))
}

// Create implements [Repository].
func (r *repository) Create(ctx context.Context, assignment *sqlc.SubmissionAssignment) (*sqlc.SubmissionAssignment, error) {
	params := sqlc.CreateSubmissionAssignmentParams{
		SubmissionID: assignment.SubmissionID,
		AssignedBy:   assignment.AssignedBy,
		AssignedTo:   assignment.AssignedTo,
		AssignedAt:   assignment.AssignedAt,
		CreatedAt:    assignment.CreatedAt,
	}

	result, err := r.q.CreateSubmissionAssignment(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	r.cacheAssignment(ctx, &result)
	return &result, nil
}

// GetByID implements [Repository].
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.SubmissionAssignment, error) {
	if r.cfg.Cache.EnableCache {
		key := buildAssignmentKey(id)
		var assignment sqlc.SubmissionAssignment
		if err := r.cache.Get(ctx, key, &assignment); err == nil {
			return &assignment, nil
		}
	}

	assignment, err := r.q.GetSubmissionAssignment(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheAssignment(ctx, &assignment)
	return &assignment, nil
}

// GetBySubmissionID implements [Repository].
func (r *repository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*sqlc.SubmissionAssignment, error) {
	assignment, err := r.q.GetSubmissionAssignmentBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetBySubmissionID: %w", err)
	}

	r.cacheAssignment(ctx, &assignment)
	return &assignment, nil
}

// GetByAssignedTo implements [Repository].
func (r *repository) GetByAssignedTo(ctx context.Context, assignedTo uuid.UUID) ([]*sqlc.SubmissionAssignment, error) {
	assignments, err := r.q.GetSubmissionAssignmentsByAssignedTo(ctx, &assignedTo)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByAssignedTo: %w", err)
	}

	result := make([]*sqlc.SubmissionAssignment, len(assignments))
	for i := range assignments {
		result[i] = &assignments[i]
		r.cacheAssignment(ctx, &assignments[i])
	}

	return result, nil
}

// GetByAssignedBy implements [Repository].
func (r *repository) GetByAssignedBy(ctx context.Context, assignedBy uuid.UUID) ([]*sqlc.SubmissionAssignment, error) {
	assignments, err := r.q.GetSubmissionAssignmentsByAssignedBy(ctx, &assignedBy)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByAssignedBy: %w", err)
	}

	result := make([]*sqlc.SubmissionAssignment, len(assignments))
	for i := range assignments {
		result[i] = &assignments[i]
		r.cacheAssignment(ctx, &assignments[i])
	}

	return result, nil
}

// Update implements [Repository].
func (r *repository) Update(ctx context.Context, assignment *sqlc.SubmissionAssignment) (*sqlc.SubmissionAssignment, error) {
	r.invalidateCache(ctx, assignment.ID)

	params := sqlc.UpdateSubmissionAssignmentParams{
		ID:         assignment.ID,
		AssignedBy: assignment.AssignedBy,
		AssignedTo: assignment.AssignedTo,
		AssignedAt: assignment.AssignedAt,
	}

	result, err := r.q.UpdateSubmissionAssignment(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("repository.Update: %w", err)
	}

	r.cacheAssignment(ctx, &result)
	return &result, nil
}

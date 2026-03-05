package submission_status_history

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
	Create(ctx context.Context, history *sqlc.SubmissionStatusHistory) (*sqlc.SubmissionStatusHistory, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.SubmissionStatusHistory, error)
	GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]sqlc.SubmissionStatusHistory, error)
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

func buildHistoryKey(id uuid.UUID) string {
	return fmt.Sprintf("submission_status_history:%s", id.String())
}

func (r *repository) cacheHistory(ctx context.Context, history *sqlc.SubmissionStatusHistory) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildHistoryKey(history.ID),
		Value: history,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

// Create implements [Repository].
func (r *repository) Create(ctx context.Context, history *sqlc.SubmissionStatusHistory) (*sqlc.SubmissionStatusHistory, error) {
	params := sqlc.CreateSubmissionStatusHistoryParams{
		SubmissionID: history.SubmissionID,
		ChangedBy:    history.ChangedBy,
		OldStatus:    history.OldStatus,
		NewStatus:    history.NewStatus,
		Reason:       history.Reason,
		CreatedAt:    history.CreatedAt,
	}

	result, err := r.q.CreateSubmissionStatusHistory(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	r.cacheHistory(ctx, &result)
	return &result, nil
}

// GetByID implements [Repository].
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.SubmissionStatusHistory, error) {
	if r.cfg.Cache.EnableCache {
		key := buildHistoryKey(id)
		var history sqlc.SubmissionStatusHistory
		if err := r.cache.Get(ctx, key, &history); err == nil {
			return &history, nil
		}
	}

	history, err := r.q.GetSubmissionStatusHistoryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheHistory(ctx, &history)
	return &history, nil
}

// GetBySubmissionID implements [Repository].
func (r *repository) GetBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]sqlc.SubmissionStatusHistory, error) {
	history, err := r.q.GetSubmissionStatusHistoryBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetBySubmissionID: %w", err)
	}

	for i := range history {
		r.cacheHistory(ctx, &history[i])
	}

	return history, nil
}

package story_vote

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/infras/cache"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Vote CRUD
	UpsertVote(ctx context.Context, arg sqlc.UpsertStoryVoteParams) (*sqlc.StoryVote, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryVote, error)
	GetByStoryAndUser(ctx context.Context, storyID, userID uuid.UUID) (*sqlc.StoryVote, error)
	DeleteByStoryAndUser(ctx context.Context, storyID, userID uuid.UUID) error

	// Vote statistics
	GetStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error)

	// List operations
	ListByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryVote, int64, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]sqlc.StoryVote, int64, error)
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

func buildVoteKey(id uuid.UUID) string {
	return fmt.Sprintf("story_vote:%s", id.String())
}

func (r *repository) cacheVote(ctx context.Context, vote *sqlc.StoryVote) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildVoteKey(vote.ID),
		Value: vote,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

// UpsertVote creates or updates a vote
func (r *repository) UpsertVote(ctx context.Context, arg sqlc.UpsertStoryVoteParams) (*sqlc.StoryVote, error) {
	result, err := r.q.UpsertStoryVote(ctx, arg)
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.UpsertVote: %w", err)
	}

	r.cacheVote(ctx, &result)
	return &result, nil
}

// GetByID retrieves a vote by ID - note: requires custom query
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryVote, error) {
	// Note: This requires a GetStoryVoteByID query to be added to sqlc
	// For now, this is a placeholder
	vote, err := r.q.GetStoryVoteByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheVote(ctx, &vote)
	return &vote, nil
}

// GetByStoryAndUser retrieves a vote by story and user
func (r *repository) GetByStoryAndUser(ctx context.Context, storyID, userID uuid.UUID) (*sqlc.StoryVote, error) {
	vote, err := r.q.GetStoryVote(ctx, sqlc.GetStoryVoteParams{
		StoryID: storyID,
		UserID:  userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByStoryAndUser: %w", err)
	}

	return &vote, nil
}

// DeleteByStoryAndUser deletes a vote by story and user
func (r *repository) DeleteByStoryAndUser(ctx context.Context, storyID, userID uuid.UUID) error {
	err := r.q.DeleteStoryVote(ctx, sqlc.DeleteStoryVoteParams{
		StoryID: storyID,
		UserID:  userID,
	})
	if err != nil {
		return fmt.Errorf("repository.DeleteByStoryAndUser: %w", err)
	}
	return nil
}

// GetStats retrieves vote statistics for a story
func (r *repository) GetStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error) {
	stats, err := r.q.GetStoryVoteStats(ctx, storyID)
	if err != nil {
		return sqlc.GetStoryVoteStatsRow{}, fmt.Errorf("repository.GetStats: %w", err)
	}
	return stats, nil
}

// ListByStory lists votes for a story with pagination
func (r *repository) ListByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryVote, int64, error) {
	// Note: Requires ListStoryVotesByStory query
	votes, err := r.q.ListStoryVotesByStory(ctx, sqlc.ListStoryVotesByStoryParams{
		StoryID: storyID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStory: %w", err)
	}

	count, err := r.q.CountStoryVotesByStory(ctx, storyID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStory count: %w", err)
	}

	return votes, count, nil
}

// ListByUser lists votes by a user with pagination
func (r *repository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]sqlc.StoryVote, int64, error) {
	// Note: Requires ListStoryVotesByUser query
	votes, err := r.q.ListStoryVotesByUser(ctx, sqlc.ListStoryVotesByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByUser: %w", err)
	}

	count, err := r.q.CountStoryVotesByUser(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByUser count: %w", err)
	}

	return votes, count, nil
}

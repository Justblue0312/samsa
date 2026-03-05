package commentreaction

import (
	"context"
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
	Get(ctx context.Context, commentReactionId uuid.UUID) (*sqlc.CommentReaction, error)
	GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) (*[]sqlc.CommentReaction, error)
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.ReactionType]int32, error)
	Upsert(ctx context.Context, commentReaction *sqlc.CommentReaction) (*sqlc.CommentReaction, error)
	Delete(ctx context.Context, commentReaction *sqlc.CommentReaction) error
	CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

type repository struct {
	q     *sqlc.Queries
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(q *sqlc.Queries, cfg *config.Config, cache *cache.Client) Repository {
	return &repository{q: q, cfg: cfg, cache: cache}
}

func buildCommentReactionKey(commentReactionId uuid.UUID) string {
	return fmt.Sprintf("comment_reaction:%s", commentReactionId)
}

func (r *repository) cacheCommentReaction(ctx context.Context, commentReaction *sqlc.CommentReaction) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildCommentReactionKey(commentReaction.ID),
		Value: commentReaction,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, commentReaction *sqlc.CommentReaction) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildCommentReactionKey(commentReaction.ID))
}

// CountTotal implements [Repository].
func (r *repository) CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	count, err := r.q.CountTotalCommentReactions(ctx, sqlc.CountTotalCommentReactionsParams{
		CommentID:  commentID,
		EntityType: entityType,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCommentID implements [Repository].
func (r *repository) CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.ReactionType]int32, error) {
	rows, err := r.q.CountReactionsByCommentID(ctx, sqlc.CountReactionsByCommentIDParams{
		CommentID:  commentID,
		EntityType: entityType,
	})
	if err != nil {
		return nil, err
	}

	counts := make(map[sqlc.ReactionType]int32, len(rows))
	for _, row := range rows {
		counts[row.ReactionType] = int32(row.Total)
	}

	return counts, nil
}

// Delete implements [Repository].
func (r *repository) Delete(ctx context.Context, commentReaction *sqlc.CommentReaction) error {
	r.invalidateCache(ctx, commentReaction)
	return r.q.DeleteCommentReaction(ctx, commentReaction.ID)
}

// Get implements [Repository].
func (r *repository) Get(ctx context.Context, commentReactionId uuid.UUID) (*sqlc.CommentReaction, error) {
	cr, err := r.q.GetCommentReactionByID(ctx, commentReactionId)
	if err != nil {
		return nil, err
	}

	r.cacheCommentReaction(ctx, &cr)

	return &cr, nil
}

// GetByCommentID implements [Repository].
func (r *repository) GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit int32, offset int32) (*[]sqlc.CommentReaction, error) {
	crs, err := r.q.GetReactionsByCommentID(ctx, sqlc.GetReactionsByCommentIDParams{
		CommentID:  commentID,
		EntityType: entityType,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	for _, cr := range crs {
		r.cacheCommentReaction(ctx, &cr)
	}

	return &crs, nil
}

// Upsert implements [Repository].
func (r *repository) Upsert(ctx context.Context, commentReaction *sqlc.CommentReaction) (*sqlc.CommentReaction, error) {
	r.invalidateCache(ctx, commentReaction)

	cr, err := r.q.UpsertCommentReaction(ctx, sqlc.UpsertCommentReactionParams{
		EntityType:   commentReaction.EntityType,
		CommentID:    commentReaction.CommentID,
		UserID:       commentReaction.UserID,
		ReactionType: commentReaction.ReactionType,
		CreatedAt:    commentReaction.CreatedAt,
		UpdatedAt:    commentReaction.UpdatedAt,
	})
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrDuplicateReaction
		}
		return nil, err
	}
	r.cacheCommentReaction(ctx, &cr)
	return &cr, nil
}

package commentvote

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
	Get(ctx context.Context, commentVoteId uuid.UUID) (*sqlc.CommentVote, error)
	GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) (*[]sqlc.CommentVote, error)
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.VoteType]int32, error)
	Upsert(ctx context.Context, commentVote *sqlc.CommentVote) (*sqlc.CommentVote, error)
	Delete(ctx context.Context, commentVote *sqlc.CommentVote) error
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

func buildCommentVoteKey(commentVoteId uuid.UUID) string {
	return fmt.Sprintf("comment_vote:%s", commentVoteId)
}

func (r *repository) cacheCommentVote(ctx context.Context, commentVote *sqlc.CommentVote) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildCommentVoteKey(commentVote.ID),
		Value: commentVote,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, commentVote *sqlc.CommentVote) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildCommentVoteKey(commentVote.ID))
}

// CountTotal implements [Repository].
func (r *repository) CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	count, err := r.q.CountTotalCommentVotesByCommentID(ctx, sqlc.CountTotalCommentVotesByCommentIDParams{
		CommentID:  commentID,
		EntityType: entityType,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountByCommentID implements [Repository].
func (r *repository) CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.VoteType]int32, error) {
	rows, err := r.q.GetCommentVoteCountByCommentID(ctx, sqlc.GetCommentVoteCountByCommentIDParams{
		CommentID:  commentID,
		EntityType: entityType,
	})

	if err != nil {
		return nil, err
	}

	counts := make(map[sqlc.VoteType]int32, len(rows))
	for _, row := range rows {
		counts[row.VoteType] = int32(row.Total)
	}

	return counts, nil
}

// Delete implements [Repository].
func (r *repository) Delete(ctx context.Context, commentVote *sqlc.CommentVote) error {
	r.invalidateCache(ctx, commentVote)
	return r.q.DeleteCommentVote(ctx, commentVote.CommentID)
}

// Get implements [Repository].
func (r *repository) Get(ctx context.Context, commentVoteId uuid.UUID) (*sqlc.CommentVote, error) {
	cr, err := r.q.GetCommentVoteByID(ctx, commentVoteId)
	if err != nil {
		return nil, err
	}

	r.cacheCommentVote(ctx, &cr)

	return &cr, nil
}

// GetByCommentID implements [Repository].
func (r *repository) GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit int32, offset int32) (*[]sqlc.CommentVote, error) {
	cvs, err := r.q.GetCommentVotesByCommentID(ctx, sqlc.GetCommentVotesByCommentIDParams{
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

	for _, cv := range cvs {
		r.cacheCommentVote(ctx, &cv)
	}

	return &cvs, nil
}

// Upsert implements [Repository].
func (r *repository) Upsert(ctx context.Context, commentVote *sqlc.CommentVote) (*sqlc.CommentVote, error) {
	r.invalidateCache(ctx, commentVote)

	cv, err := r.q.UpsertCommentVote(ctx, sqlc.UpsertCommentVoteParams{
		CommentID:  commentVote.CommentID,
		EntityType: commentVote.EntityType,
		UserID:     commentVote.UserID,
		VoteType:   commentVote.VoteType,
		CreatedAt:  commentVote.CreatedAt,
		UpdatedAt:  commentVote.UpdatedAt,
	})
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrDuplicateVote
		}
		return nil, err
	}

	r.cacheCommentVote(ctx, &cv)

	return &cv, nil
}

package comment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/infras/cache"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	GetByID(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool) (*sqlc.Comment, error)
	Create(ctx context.Context, comment *sqlc.Comment) (*sqlc.Comment, error)
	Update(ctx context.Context, comment *sqlc.Comment) (*sqlc.Comment, error)
	SoftDelete(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType) (*sqlc.Comment, error)
	List(ctx context.Context, f *CommentFilter) ([]sqlc.Comment, error)
	GetReplies(ctx context.Context, parentID uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool, limit, offset int32) ([]sqlc.Comment, error)
	GetNestingDepth(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool) (int32, error)
	// Bulk moderation
	BulkDelete(ctx context.Context, ids []uuid.UUID, deletedBy uuid.UUID) ([]sqlc.Comment, error)
	BulkArchive(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error)
	BulkResolve(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error)
	BulkPin(ctx context.Context, ids []uuid.UUID, pinnedBy uuid.UUID) ([]sqlc.Comment, error)
	BulkUnpin(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error)
	// Search
	Search(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, search string, limit, offset int32) ([]sqlc.Comment, error)
	ListWithFilters(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported, isPinned *bool, parentID *uuid.UUID, limit, offset int32) ([]sqlc.Comment, error)
	CountWithFilters(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported *bool) (int64, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error)
}

type repository struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(pool *pgxpool.Pool, q *sqlc.Queries, cfg *config.Config, cache *cache.Client) Repository {
	return &repository{pool: pool, q: q, cfg: cfg, cache: cache}
}

func buildCommentKey(id uuid.UUID, entityType sqlc.EntityType) string {
	return fmt.Sprintf("comment:%s:%s", entityType, id.String())
}

func (r *repository) cacheComment(ctx context.Context, comment *sqlc.Comment) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildCommentKey(comment.ID, comment.EntityType),
		Value: comment,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildCommentKey(id, entityType))
}

// Create implements [Repository].
func (r *repository) Create(ctx context.Context, comment *sqlc.Comment) (*sqlc.Comment, error) {
	params := sqlc.CreateCommentParams{
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsDeleted:     comment.IsDeleted,
		IsResolved:    comment.IsResolved,
		IsArchived:    comment.IsArchived,
		IsReported:    comment.IsReported,
		ReportedAt:    comment.ReportedAt,
		ReportedBy:    comment.ReportedBy,
		IsPinned:      comment.IsPinned,
		PinnedAt:      comment.PinnedAt,
		PinnedBy:      comment.PinnedBy,
		EntityType:    comment.EntityType,
		EntityID:      comment.EntityID,
		Source:        comment.Source,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	}

	result, err := r.q.CreateComment(ctx, params)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetByID implements [Repository].
func (r *repository) GetByID(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool) (*sqlc.Comment, error) {
	if r.cfg.Cache.EnableCache {
		key := buildCommentKey(commentId, entityType)
		var comment sqlc.Comment
		if err := r.cache.Get(ctx, key, &comment); err == nil {
			return &comment, nil
		}
	}

	comment, err := r.q.GetCommentByID(ctx, sqlc.GetCommentByIDParams{
		ID:         commentId,
		EntityType: entityType,
		IsDeleted:  includedDeleted,
	})
	if err != nil {
		return nil, err
	}

	r.cacheComment(ctx, &comment)
	return &comment, nil
}

// GetNestingDepth implements [Repository].
func (r *repository) GetNestingDepth(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool) (int32, error) {
	depth, err := r.q.GetCommentNestingDepth(ctx, sqlc.GetCommentNestingDepthParams{
		ID:         commentId,
		EntityType: entityType,
		IsDeleted:  includedDeleted,
	})
	return depth, err
}

// GetReplies implements [Repository].
func (r *repository) GetReplies(ctx context.Context, parentID uuid.UUID, entityType sqlc.EntityType, includedDeleted *bool, limit, offset int32) ([]sqlc.Comment, error) {
	comment, err := r.q.GetCommentReplies(ctx, sqlc.GetCommentRepliesParams{
		ParentID:   &parentID,
		EntityType: entityType,
		IsDeleted:  includedDeleted,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	for _, c := range comment {
		r.cacheComment(ctx, &c)
	}

	return comment, nil
}

// List implements [Repository].
func (r *repository) List(ctx context.Context, f *CommentFilter) ([]sqlc.Comment, error) {
	orderByValue := f.ToSQL()
	if orderByValue == "" {
		orderByValue = "created_at DESC"
	}

	args := []any{false}
	argIndex := 2

	query := `SELECT * FROM comment WHERE is_deleted = $1`

	entityType := sql.NullString{String: f.EntityType, Valid: f.EntityType != ""}
	args = append(args, entityType)
	argIndex++

	if len(f.EntityID) > 0 {
		entityIDs := make([]string, len(f.EntityID))
		for i, id := range f.EntityID {
			entityIDs[i] = id.String()
		}
		query += fmt.Sprintf(" AND entity_id IN (%s)", strings.Join(entityIDs, ","))
	}

	if len(f.ID) > 0 {
		ids := make([]string, len(f.ID))
		for i, id := range f.ID {
			ids[i] = id.String()
		}
		query += fmt.Sprintf(" AND id IN (%s)", strings.Join(ids, ","))
	}

	if f.IsPinned != nil {
		query += fmt.Sprintf(" AND is_pinned = $%d", argIndex)
		args = append(args, *f.IsPinned)
		argIndex++
	}

	if f.IsReported != nil {
		query += fmt.Sprintf(" AND is_reported = $%d", argIndex)
		args = append(args, *f.IsReported)
		argIndex++
	}

	if f.IsArchived != nil {
		query += fmt.Sprintf(" AND is_archived = $%d", argIndex)
		args = append(args, *f.IsArchived)
		argIndex++
	}

	if f.IsResolved != nil {
		query += fmt.Sprintf(" AND is_resolved = $%d", argIndex)
		args = append(args, *f.IsResolved)
		argIndex++
	}

	countQuery := strings.Replace(query, "SELECT *", "SELECT COUNT(*)", 1)
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderByValue, argIndex, argIndex+1)
	args = append(args, f.GetLimit(), f.GetOffset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []sqlc.Comment
	for rows.Next() {
		var comment sqlc.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.ParentID,
			&comment.Content,
			&comment.Depth,
			&comment.Score,
			&comment.IsDeleted,
			&comment.IsResolved,
			&comment.IsArchived,
			&comment.IsReported,
			&comment.ReportedAt,
			&comment.ReportedBy,
			&comment.IsPinned,
			&comment.PinnedAt,
			&comment.PinnedBy,
			&comment.EntityType,
			&comment.EntityID,
			&comment.Source,
			&comment.ReplyCount,
			&comment.ReactionCount,
			&comment.Metadata,
			&comment.DeletedBy,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
		r.cacheComment(ctx, &comment)
	}

	countRows, err := r.pool.Query(ctx, countQuery, args[:len(args)-2]...)
	if err != nil {
		return nil, err
	}
	defer countRows.Close()

	var totalCount int64
	if countRows.Next() {
		if err := countRows.Scan(&totalCount); err != nil {
			return nil, err
		}
	}

	return comments, nil
}

// SoftDelete implements [Repository].
func (r *repository) SoftDelete(ctx context.Context, commentId uuid.UUID, entityType sqlc.EntityType) (*sqlc.Comment, error) {
	r.invalidateCache(ctx, commentId, entityType)
	result, err := r.q.SoftDeleteComment(ctx, sqlc.SoftDeleteCommentParams{
		ID:         commentId,
		EntityType: entityType,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update implements [Repository].
func (r *repository) Update(ctx context.Context, comment *sqlc.Comment) (*sqlc.Comment, error) {
	r.invalidateCache(ctx, comment.ID, comment.EntityType)

	params := sqlc.UpdateCommentParams{
		EntityType:    comment.EntityType,
		ID:            comment.ID,
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsResolved:    comment.IsResolved,
		IsArchived:    comment.IsArchived,
		IsReported:    comment.IsReported,
		ReportedAt:    comment.ReportedAt,
		ReportedBy:    comment.ReportedBy,
		IsPinned:      comment.IsPinned,
		PinnedAt:      comment.PinnedAt,
		PinnedBy:      comment.PinnedBy,
		EntityID:      comment.EntityID,
		Source:        comment.Source,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	}

	result, err := r.q.UpdateComment(ctx, params)
	if err != nil {
		return nil, err
	}

	r.cacheComment(ctx, &result)

	return &result, nil
}

// BulkDelete implements [Repository].
func (r *repository) BulkDelete(ctx context.Context, ids []uuid.UUID, deletedBy uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.BulkDeleteComments(ctx, sqlc.BulkDeleteCommentsParams{
		Column1:   ids,
		DeletedBy: &deletedBy,
	})
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.invalidateCache(ctx, c.ID, c.EntityType)
	}
	return result, nil
}

// BulkArchive implements [Repository].
func (r *repository) BulkArchive(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.BulkArchiveComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// BulkResolve implements [Repository].
func (r *repository) BulkResolve(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.BulkResolveComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// BulkPin implements [Repository].
func (r *repository) BulkPin(ctx context.Context, ids []uuid.UUID, pinnedBy uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.BulkPinComments(ctx, sqlc.BulkPinCommentsParams{
		Column1:  ids,
		PinnedBy: &pinnedBy,
	})
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// BulkUnpin implements [Repository].
func (r *repository) BulkUnpin(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.BulkUnpinComments(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// Search implements [Repository].
func (r *repository) Search(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, search string, limit, offset int32) ([]sqlc.Comment, error) {
	result, err := r.q.SearchComments(ctx, sqlc.SearchCommentsParams{
		EntityType: entityType,
		EntityID:   entityID,
		Search:     &search,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// ListWithFilters implements [Repository].
func (r *repository) ListWithFilters(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported, isPinned *bool, parentID *uuid.UUID, limit, offset int32) ([]sqlc.Comment, error) {
	result, err := r.q.ListCommentsByEntityWithFilters(ctx, sqlc.ListCommentsByEntityWithFiltersParams{
		EntityType: entityType,
		EntityID:   entityID,
		IsDeleted:  isDeleted,
		IsResolved: isResolved,
		IsArchived: isArchived,
		IsReported: isReported,
		IsPinned:   isPinned,
		ParentID:   parentID,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

// CountWithFilters implements [Repository].
func (r *repository) CountWithFilters(ctx context.Context, entityType sqlc.EntityType, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported *bool) (int64, error) {
	result, err := r.q.CountCommentsWithFilters(ctx, sqlc.CountCommentsWithFiltersParams{
		EntityType: entityType,
		EntityID:   entityID,
		IsDeleted:  isDeleted,
		IsResolved: isResolved,
		IsArchived: isArchived,
		IsReported: isReported,
	})
	return result, err
}

// GetByIDs implements [Repository].
func (r *repository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.Comment, error) {
	result, err := r.q.GetCommentsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range result {
		r.cacheComment(ctx, &c)
	}
	return result, nil
}

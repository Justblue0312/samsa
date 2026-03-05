package tag

//go:generate mockgen -destination=mocks/repository_mock.go -source=repository.go -package=mocks

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

type Repository interface {
	Create(ctx context.Context, tag *sqlc.Tag) (*sqlc.Tag, error)
	GetByID(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) (*sqlc.Tag, error)
	GetByName(ctx context.Context, name string, entityType sqlc.EntityType) (*sqlc.Tag, error)
	Update(ctx context.Context, tag *sqlc.Tag, entityType sqlc.EntityType) (*sqlc.Tag, error)
	Delete(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) error
	GetTagsByEntityID(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool) ([]*sqlc.Tag, error)
	GetTagsByOwnerID(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]*sqlc.Tag, error)
	GetTagsByIDs(ctx context.Context, tagIDs []uuid.UUID, entityType sqlc.EntityType) ([]*sqlc.Tag, error)
	CountTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType) (int, error)
	CountTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType) (int, error)
	SearchTags(ctx context.Context, entityType sqlc.EntityType, searchQuery *string, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]*sqlc.Tag, error)
	ListTags(ctx context.Context, f *TagFilter, entityType sqlc.EntityType) ([]*sqlc.Tag, int64, error)
	UpsertTag(ctx context.Context, tag *sqlc.Tag) (*sqlc.Tag, error)
	GetEntityIDsByTagNames(ctx context.Context, tagNames []string, entityType sqlc.EntityType) ([]uuid.UUID, error)
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

func buildTagKey(id uuid.UUID) string {
	return fmt.Sprintf("tag:%s", id.String())
}

func (r *repository) cacheTag(ctx context.Context, tag *sqlc.Tag) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildTagKey(tag.ID),
		Value: tag,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) Create(ctx context.Context, tag *sqlc.Tag) (*sqlc.Tag, error) {
	params := sqlc.CreateTagParams{
		ID:            tag.ID,
		OwnerID:       tag.OwnerID,
		Name:          tag.Name,
		Description:   tag.Description,
		Color:         tag.Color,
		EntityType:    tag.EntityType,
		EntityID:      tag.EntityID,
		IsHidden:      tag.IsHidden,
		IsSystem:      tag.IsSystem,
		IsRecommended: tag.IsRecommended,
		CreatedAt:     tag.CreatedAt,
		UpdatedAt:     tag.UpdatedAt,
	}

	result, err := r.q.CreateTag(ctx, params)
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	r.cacheTag(ctx, &result)

	return &result, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) (*sqlc.Tag, error) {
	if r.cfg.Cache.EnableCache {
		key := fmt.Sprintf("tag:%s", id.String())
		var tag sqlc.Tag
		if err := r.cache.Get(ctx, key, &tag); err == nil {
			return &tag, nil
		}
	}

	tag, err := r.q.GetTagByID(ctx, sqlc.GetTagByIDParams{
		ID:         id,
		EntityType: entityType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	r.cacheTag(ctx, &tag)

	return &tag, nil
}

func (r *repository) GetByName(ctx context.Context, name string, entityType sqlc.EntityType) (*sqlc.Tag, error) {
	tag, err := r.q.GetTagByNameAndType(ctx, sqlc.GetTagByNameAndTypeParams{
		Name:       name,
		EntityType: entityType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByName: %w", err)
	}

	return &tag, nil
}

func (r *repository) Update(ctx context.Context, tag *sqlc.Tag, entityType sqlc.EntityType) (*sqlc.Tag, error) {
	if r.cfg.Cache.EnableCache {
		_ = r.cache.Delete(ctx, buildTagKey(tag.ID))
	}

	params := sqlc.UpdateTagParams{
		ID:            tag.ID,
		Name:          tag.Name,
		Description:   tag.Description,
		Color:         tag.Color,
		IsHidden:      tag.IsHidden,
		IsSystem:      tag.IsSystem,
		IsRecommended: tag.IsRecommended,
		UpdatedAt:     tag.UpdatedAt,
		EntityType:    entityType,
	}

	result, err := r.q.UpdateTag(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.Update: %w", err)
	}

	r.cacheTag(ctx, &result)

	return &result, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID, entityType sqlc.EntityType) error {
	if r.cfg.Cache.EnableCache {
		_ = r.cache.Delete(ctx, fmt.Sprintf("tag:%s", id.String()))
	}

	err := r.q.DeleteTag(ctx, sqlc.DeleteTagParams{
		ID:         id,
		EntityType: entityType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

func (r *repository) GetTagsByEntityID(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool) ([]*sqlc.Tag, error) {
	tags, err := r.q.GetTagsByEntityID(ctx, sqlc.GetTagsByEntityIDParams{
		EntityID:      entityID,
		EntityType:    entityType,
		IsHidden:      isHidden,
		IsSystem:      isSystem,
		IsRecommended: isRecommended,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetTagsByEntityID: %w", err)
	}

	result := make([]*sqlc.Tag, len(tags))
	for i := range tags {
		result[i] = &tags[i]
	}

	return result, nil
}

func (r *repository) GetTagsByOwnerID(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]*sqlc.Tag, error) {
	tags, err := r.q.GetTagsByOwnerID(ctx, sqlc.GetTagsByOwnerIDParams{
		OwnerID:       ownerID,
		RowLimit:      limit,
		RowOffset:     offset,
		EntityType:    entityType,
		IsHidden:      isHidden,
		IsSystem:      isSystem,
		IsRecommended: isRecommended,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetTagsByOwnerID: %w", err)
	}

	result := make([]*sqlc.Tag, len(tags))
	for i := range tags {
		result[i] = &tags[i]
	}

	return result, nil
}

func (r *repository) GetTagsByIDs(ctx context.Context, tagIDs []uuid.UUID, entityType sqlc.EntityType) ([]*sqlc.Tag, error) {
	tags, err := r.q.GetTagsByIDs(ctx, sqlc.GetTagsByIDsParams{
		TagIds:     tagIDs,
		EntityType: entityType,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetTagsByIDs: %w", err)
	}

	result := make([]*sqlc.Tag, len(tags))
	for i := range tags {
		result[i] = &tags[i]
	}

	return result, nil
}

func (r *repository) CountTagsByEntity(ctx context.Context, entityID uuid.UUID, entityType sqlc.EntityType) (int, error) {
	count, err := r.q.CountTagsByEntity(ctx, sqlc.CountTagsByEntityParams{
		EntityID:   entityID,
		EntityType: entityType,
	})
	if err != nil {
		return 0, fmt.Errorf("repository.CountTagsByEntity: %w", err)
	}

	return int(count), nil
}

func (r *repository) CountTagsByOwner(ctx context.Context, ownerID uuid.UUID, entityType sqlc.EntityType) (int, error) {
	count, err := r.q.CountTagsByOwner(ctx, sqlc.CountTagsByOwnerParams{
		OwnerID:    ownerID,
		EntityType: entityType,
	})
	if err != nil {
		return 0, fmt.Errorf("repository.CountTagsByOwner: %w", err)
	}

	return int(count), nil
}

func (r *repository) SearchTags(ctx context.Context, entityType sqlc.EntityType, searchQuery *string, isHidden, isSystem, isRecommended *bool, limit, offset int32) ([]*sqlc.Tag, error) {
	tags, err := r.q.SearchTags(ctx, sqlc.SearchTagsParams{
		Limit:         limit,
		Offset:        offset,
		EntityType:    entityType,
		SearchQuery:   searchQuery,
		IsHidden:      isHidden,
		IsSystem:      isSystem,
		IsRecommended: isRecommended,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.SearchTags: %w", err)
	}

	result := make([]*sqlc.Tag, len(tags))
	for i := range tags {
		result[i] = &tags[i]
	}

	return result, nil
}

func (r *repository) ListTags(ctx context.Context, f *TagFilter, entityType sqlc.EntityType) ([]*sqlc.Tag, int64, error) {
	orderByValue := f.ToSQL()
	if orderByValue == "" {
		orderByValue = "name ASC"
	}

	args := []any{entityType}
	countArgs := []any{entityType}
	argIndex := 2

	query := `SELECT * FROM tag WHERE entity_type = $1`

	if f.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *f.OwnerID)
		countArgs = append(countArgs, *f.OwnerID)
		argIndex++
	}
	if f.EntityID != nil {
		query += fmt.Sprintf(" AND entity_id = $%d", argIndex)
		args = append(args, *f.EntityID)
		countArgs = append(countArgs, *f.EntityID)
		argIndex++
	}
	if f.IsHidden != nil {
		query += fmt.Sprintf(" AND is_hidden = $%d", argIndex)
		args = append(args, *f.IsHidden)
		countArgs = append(countArgs, *f.IsHidden)
		argIndex++
	}
	if f.IsSystem != nil {
		query += fmt.Sprintf(" AND is_system = $%d", argIndex)
		args = append(args, *f.IsSystem)
		countArgs = append(countArgs, *f.IsSystem)
		argIndex++
	}
	if f.IsRecommended != nil {
		query += fmt.Sprintf(" AND is_recommended = $%d", argIndex)
		args = append(args, *f.IsRecommended)
		countArgs = append(countArgs, *f.IsRecommended)
		argIndex++
	}

	countQuery := `SELECT COUNT(*) FROM (` + query + `) AS count_subquery`
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderByValue, argIndex, argIndex+1)
	args = append(args, f.GetLimit(), f.GetOffset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListTags query: %w", err)
	}
	defer rows.Close()

	var tags []*sqlc.Tag
	for rows.Next() {
		var tag sqlc.Tag
		if err := rows.Scan(
			&tag.ID,
			&tag.OwnerID,
			&tag.Name,
			&tag.Description,
			&tag.Color,
			&tag.EntityType,
			&tag.EntityID,
			&tag.IsHidden,
			&tag.IsSystem,
			&tag.IsRecommended,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.ListTags scan: %w", err)
		}
		tags = append(tags, &tag)
	}

	countRows, err := r.db.Query(ctx, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListTags count: %w", err)
	}
	defer countRows.Close()

	var totalCount int64
	if countRows.Next() {
		if err := countRows.Scan(&totalCount); err != nil {
			return nil, 0, fmt.Errorf("repository.ListTags count scan: %w", err)
		}
	}

	return tags, totalCount, nil
}

func (r *repository) UpsertTag(ctx context.Context, tag *sqlc.Tag) (*sqlc.Tag, error) {
	params := sqlc.UpsertTagParams{
		ID:            tag.ID,
		OwnerID:       tag.OwnerID,
		Name:          tag.Name,
		Description:   tag.Description,
		Color:         tag.Color,
		EntityType:    tag.EntityType,
		EntityID:      tag.EntityID,
		IsHidden:      tag.IsHidden,
		IsSystem:      tag.IsSystem,
		IsRecommended: tag.IsRecommended,
		CreatedAt:     tag.CreatedAt,
		UpdatedAt:     tag.UpdatedAt,
	}

	result, err := r.q.UpsertTag(ctx, params)
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.UpsertTag: %w", err)
	}

	r.cacheTag(ctx, &result)

	return &result, nil
}

func (r *repository) GetEntityIDsByTagNames(ctx context.Context, tagNames []string, entityType sqlc.EntityType) ([]uuid.UUID, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}

	entityIDs, err := r.q.GetTagsByNames(ctx, sqlc.GetTagsByNamesParams{
		Names:      tagNames,
		EntityType: entityType,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetEntityIDsByTagNames: %w", err)
	}

	return entityIDs, nil
}

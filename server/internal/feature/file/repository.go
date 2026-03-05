package file

//go:generate mockgen -destination=mocks/repository_mock.go -source=repository.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/infras/cache"
)

type FileRepository interface {
	Create(ctx context.Context, file *sqlc.File) (*sqlc.File, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.File, error)
	GetByPath(ctx context.Context, path string) (*sqlc.File, error)
	Update(ctx context.Context, file *sqlc.File) (*sqlc.File, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *FileFilter) ([]*sqlc.File, int64, error)
	// File sharing
	Share(ctx context.Context, id uuid.UUID) (*sqlc.File, error)
	Unshare(ctx context.Context, id uuid.UUID) (*sqlc.File, error)
	GetSharedFiles(ctx context.Context, limit, offset int32) ([]sqlc.File, error)
	// File validation
	GetByOwnerAndType(ctx context.Context, ownerID uuid.UUID, mimeType string, limit, offset int32) ([]sqlc.File, error)
	GetByMimeType(ctx context.Context, mimeType string, limit, offset int32) ([]sqlc.File, error)
	CountByMimeType(ctx context.Context, mimeType string) (int64, error)
	GetTotalSizeByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
	// Soft delete
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.File, error)
	Restore(ctx context.Context, id uuid.UUID) (*sqlc.File, error)
	// Filtered list
	ListWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool, limit, offset int32) ([]sqlc.File, error)
	CountWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool) (int64, error)
}

type repository struct {
	q     *sqlc.Queries
	db    sqlc.DBTX
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(db sqlc.DBTX, cfg *config.Config, c *cache.Client) FileRepository {
	return &repository{
		q:     sqlc.New(db),
		db:    db,
		cfg:   cfg,
		cache: c,
	}
}

func buildFileKey(id uuid.UUID) string {
	return fmt.Sprintf("file:%s", id.String())
}

func (r *repository) cacheFile(ctx context.Context, file *sqlc.File) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildFileKey(file.ID),
		Value: file,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id uuid.UUID) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildFileKey(id))
}

func trimFileStrings(f *sqlc.File) {
	if f == nil {
		return
	}
	f.Name = strings.TrimSpace(f.Name)
	if f.MimeType != nil {
		trimmed := strings.TrimSpace(*f.MimeType)
		f.MimeType = &trimmed
	}
	if f.Service != nil {
		trimmed := strings.TrimSpace(*f.Service)
		f.Service = &trimmed
	}
}

// Create implements [FileRepository].
func (r *repository) Create(ctx context.Context, file *sqlc.File) (*sqlc.File, error) {
	result, err := r.q.CreateFile(ctx, sqlc.CreateFileParams{
		OwnerID:    file.OwnerID,
		Name:       file.Name,
		Path:       file.Path,
		MimeType:   file.MimeType,
		Size:       file.Size,
		Reference:  file.Reference,
		Payload:    file.Payload,
		Service:    file.Service,
		Source:     file.Source,
		IsArchived: file.IsArchived,
		CreatedAt:  file.CreatedAt,
		UpdatedAt:  file.UpdatedAt,
	})
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	trimFileStrings(&result)
	r.cacheFile(ctx, &result)

	return &result, nil
}

// Delete implements [FileRepository].
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	r.invalidateCache(ctx, id)

	err := r.q.DeleteFile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

// GetByID implements [FileRepository].
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.File, error) {
	if r.cfg.Cache.EnableCache {
		var file sqlc.File
		if err := r.cache.Get(ctx, buildFileKey(id), &file); err == nil {
			return &file, nil
		}
	}

	file, err := r.q.GetFileByID(ctx, sqlc.GetFileByIDParams{
		ID:        id,
		IsDeleted: false,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}

	trimFileStrings(&file)
	r.cacheFile(ctx, &file)

	return &file, nil
}

// GetByPath implements [FileRepository].
func (r *repository) GetByPath(ctx context.Context, path string) (*sqlc.File, error) {
	file, err := r.q.GetFileByPath(ctx, path)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByPath: %w", err)
	}

	trimFileStrings(&file)
	return &file, nil
}

// List implements [FileRepository].
func (r *repository) List(ctx context.Context, filter *FileFilter) ([]*sqlc.File, int64, error) {
	orderByValue := filter.ToSQL()
	if orderByValue == "" {
		orderByValue = "created_at DESC"
	}

	args := []any{filter.IncludedDeleted}
	argIndex := 2

	query := `SELECT * FROM file WHERE ($1 OR NOT is_deleted)`

	if filter.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filter.OwnerID)
		argIndex++
	}
	if filter.IsArchived != nil {
		query += fmt.Sprintf(" AND is_archived = $%d", argIndex)
		args = append(args, *filter.IsArchived)
		argIndex++
	}
	if len(filter.FileIDs) > 0 {
		query += fmt.Sprintf(" AND id = ANY($%d)", argIndex)
		args = append(args, filter.FileIDs)
		argIndex++
	}

	countQuery := "SELECT count(*) FROM (" + query + ") as count"
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderByValue, argIndex, argIndex+1)
	args = append(args, filter.GetLimit(), filter.GetOffset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.List query: %w", err)
	}
	defer rows.Close()

	var files []*sqlc.File
	for rows.Next() {
		var file sqlc.File
		if err := rows.Scan(
			&file.ID,
			&file.OwnerID,
			&file.Name,
			&file.Path,
			&file.MimeType,
			&file.Size,
			&file.Reference,
			&file.Payload,
			&file.Service,
			&file.Source,
			&file.IsDeleted,
			&file.IsArchived,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("repository.List scan: %w", err)
		}
		trimFileStrings(&file)
		files = append(files, &file)
	}

	countRows, err := r.db.Query(ctx, countQuery, args[:argIndex-1]...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.List count: %w", err)
	}
	defer countRows.Close()

	var totalCount int64
	if countRows.Next() {
		if err := countRows.Scan(&totalCount); err != nil {
			return nil, 0, fmt.Errorf("repository.List count scan: %w", err)
		}
	}

	return files, totalCount, nil
}

// Update implements [FileRepository].
func (r *repository) Update(ctx context.Context, file *sqlc.File) (*sqlc.File, error) {
	r.invalidateCache(ctx, file.ID)

	result, err := r.q.UpdateFile(ctx, sqlc.UpdateFileParams{
		ID:         file.ID,
		Name:       file.Name,
		Path:       file.Path,
		MimeType:   file.MimeType,
		Size:       file.Size,
		Reference:  file.Reference,
		Payload:    file.Payload,
		Service:    file.Service,
		Source:     file.Source,
		IsArchived: file.IsArchived,
		UpdatedAt:  file.UpdatedAt,
		IsDeleted:  file.IsDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if common.IsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("repository.Update: %w", err)
	}

	trimFileStrings(&result)
	r.cacheFile(ctx, &result)

	return &result, nil
}

// Share implements [FileRepository].
func (r *repository) Share(ctx context.Context, id uuid.UUID) (*sqlc.File, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.ShareFile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.Share: %w", err)
	}

	r.cacheFile(ctx, &result)
	return &result, nil
}

// Unshare implements [FileRepository].
func (r *repository) Unshare(ctx context.Context, id uuid.UUID) (*sqlc.File, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.UnshareFile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.Unshare: %w", err)
	}

	r.cacheFile(ctx, &result)
	return &result, nil
}

// GetSharedFiles implements [FileRepository].
func (r *repository) GetSharedFiles(ctx context.Context, limit, offset int32) ([]sqlc.File, error) {
	result, err := r.q.GetSharedFiles(ctx, sqlc.GetSharedFilesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetSharedFiles: %w", err)
	}

	for i := range result {
		r.cacheFile(ctx, &result[i])
	}
	return result, nil
}

// GetByOwnerAndType implements [FileRepository].
func (r *repository) GetByOwnerAndType(ctx context.Context, ownerID uuid.UUID, mimeType string, limit, offset int32) ([]sqlc.File, error) {
	result, err := r.q.GetFilesByOwnerAndType(ctx, sqlc.GetFilesByOwnerAndTypeParams{
		OwnerID:  ownerID,
		MimeType: &mimeType,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetByOwnerAndType: %w", err)
	}

	for i := range result {
		r.cacheFile(ctx, &result[i])
	}
	return result, nil
}

// GetByMimeType implements [FileRepository].
func (r *repository) GetByMimeType(ctx context.Context, mimeType string, limit, offset int32) ([]sqlc.File, error) {
	result, err := r.q.GetFilesByMimeType(ctx, sqlc.GetFilesByMimeTypeParams{
		MimeType: &mimeType,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.GetByMimeType: %w", err)
	}

	for i := range result {
		r.cacheFile(ctx, &result[i])
	}
	return result, nil
}

// CountByMimeType implements [FileRepository].
func (r *repository) CountByMimeType(ctx context.Context, mimeType string) (int64, error) {
	return r.q.CountFilesByMimeType(ctx, &mimeType)
}

// GetTotalSizeByOwner implements [FileRepository].
func (r *repository) GetTotalSizeByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return r.q.GetTotalSizeByOwner(ctx, ownerID)
}

// SoftDelete implements [FileRepository].
func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.File, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.SoftDeleteFile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.SoftDelete: %w", err)
	}

	r.cacheFile(ctx, &result)
	return &result, nil
}

// Restore implements [FileRepository].
func (r *repository) Restore(ctx context.Context, id uuid.UUID) (*sqlc.File, error) {
	r.invalidateCache(ctx, id)

	result, err := r.q.RestoreFile(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.Restore: %w", err)
	}

	r.cacheFile(ctx, &result)
	return &result, nil
}

// ListWithFilters implements [FileRepository].
func (r *repository) ListWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool, limit, offset int32) ([]sqlc.File, error) {
	args := sqlc.ListFilesWithFiltersParams{
		Limit:  limit,
		Offset: offset,
	}

	if ownerID != nil {
		ownerUUID, err := uuid.Parse(*ownerID)
		if err == nil {
			args.OwnerID = &ownerUUID
		}
	}
	if mimeType != nil {
		args.MimeType = mimeType
	}
	if reference != nil {
		args.Reference = reference
	}
	if isArchived != nil {
		args.IsArchived = isArchived
	}

	result, err := r.q.ListFilesWithFilters(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("repository.ListWithFilters: %w", err)
	}

	for i := range result {
		r.cacheFile(ctx, &result[i])
	}
	return result, nil
}

// CountWithFilters implements [FileRepository].
func (r *repository) CountWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool) (int64, error) {
	args := sqlc.CountFilesWithFiltersParams{}

	if ownerID != nil {
		ownerUUID, err := uuid.Parse(*ownerID)
		if err == nil {
			args.OwnerID = &ownerUUID
		}
	}
	if mimeType != nil {
		args.MimeType = mimeType
	}
	if reference != nil {
		args.Reference = reference
	}
	if isArchived != nil {
		args.IsArchived = isArchived
	}

	return r.q.CountFilesWithFilters(ctx, args)
}

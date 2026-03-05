package author

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/infras/cache"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Author, error)
	GetBySlug(ctx context.Context, slug string) (*sqlc.Author, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*sqlc.Author, error)
	Create(ctx context.Context, author *sqlc.Author) (*sqlc.Author, error)
	Update(ctx context.Context, author *sqlc.Author) (*sqlc.Author, error)
	SoftDelete(ctx context.Context, userId, authorId uuid.UUID) error
	Restore(ctx context.Context, userId, authorId uuid.UUID) error
	Delete(ctx context.Context, userId, authorId uuid.UUID) error
	List(ctx context.Context, f *AuthorFilter) (*[]sqlc.Author, int64, error)
}

type repository struct {
	pool  *pgxpool.Pool
	q     *sqlc.Queries
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(q *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config, cache *cache.Client) Repository {
	return &repository{
		q:     q,
		pool:  pool,
		cfg:   cfg,
		cache: cache,
	}
}

func buildAuthorKey(id uuid.UUID) string {
	return fmt.Sprintf("author:%s", id.String())
}

func (r *repository) cacheAuthor(ctx context.Context, author *sqlc.Author) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildAuthorKey(author.ID),
		Value: author,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id uuid.UUID) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Delete(ctx, buildAuthorKey(id))
}

// Create implements [AuthorRepository].
func (r *repository) Create(ctx context.Context, author *sqlc.Author) (*sqlc.Author, error) {
	result, err := r.q.CreateAuthor(ctx, sqlc.CreateAuthorParams{
		UserID:                        author.UserID,
		MediaID:                       author.MediaID,
		StageName:                     author.StageName,
		Gender:                        author.Gender,
		Slug:                          author.Slug,
		FirstName:                     author.FirstName,
		LastName:                      author.LastName,
		DOB:                           author.DOB,
		Phone:                         author.Phone,
		Bio:                           author.Bio,
		Description:                   author.Description,
		AcceptedTermsOfService:        author.AcceptedTermsOfService,
		EmailNewslettersAndChangelogs: author.EmailNewslettersAndChangelogs,
		EmailPromotionsAndEvents:      author.EmailPromotionsAndEvents,
		IsRecommended:                 author.IsRecommended,
		IsDeleted:                     author.IsDeleted,
		Stats:                         author.Stats,
		CreatedAt:                     author.CreatedAt,
		UpdatedAt:                     author.UpdatedAt,
		DeletedAt:                     author.DeletedAt,
	})
	if err != nil {
		// Translate unique constraint violation on slug to domain error.
		if common.IsUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}

	r.cacheAuthor(ctx, &result)

	return &result, nil
}

// SoftDelete implements [AuthorRepository].
func (r *repository) SoftDelete(ctx context.Context, userId, authorId uuid.UUID) error {
	r.invalidateCache(ctx, authorId)

	return r.q.SoftDeleteAuthor(ctx, sqlc.SoftDeleteAuthorParams{
		ID:     authorId,
		UserID: userId,
	})
}

// Restore implements [AuthorRepository].
func (r *repository) Restore(ctx context.Context, userId, authorId uuid.UUID) error {
	r.invalidateCache(ctx, authorId)

	return r.q.RestoreAuthor(ctx, sqlc.RestoreAuthorParams{
		ID:     authorId,
		UserID: userId,
	})
}

// Delete implements [AuthorRepository].
func (r *repository) Delete(ctx context.Context, userId, authorId uuid.UUID) error {
	r.invalidateCache(ctx, authorId)

	return r.q.DeleteAuthor(ctx, sqlc.DeleteAuthorParams{
		ID:     authorId,
		UserID: userId,
	})
}

// GetByID implements [AuthorRepository].
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Author, error) {
	if r.cfg.Cache.EnableCache {
		var author sqlc.Author
		if err := r.cache.Get(ctx, buildAuthorKey(id), &author); err == nil {
			return &author, nil
		}
	}

	isDeleted := false
	author, err := r.q.GetAuthorByID(ctx, sqlc.GetAuthorByIDParams{
		ID:        id,
		IsDeleted: isDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuthorNotFound
		}
		return nil, err
	}

	r.cacheAuthor(ctx, &author)

	return &author, nil
}

// GetBySlug implements [AuthorRepository].
func (r *repository) GetBySlug(ctx context.Context, slug string) (*sqlc.Author, error) {
	isDeleted := false
	author, err := r.q.GetAuthorBySlug(ctx, sqlc.GetAuthorBySlugParams{
		Slug:      slug,
		IsDeleted: isDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuthorNotFound
		}
		return nil, err
	}

	return &author, nil
}

// GetByUserID implements [AuthorRepository].
func (r *repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*sqlc.Author, error) {
	isDeleted := false
	author, err := r.q.GetAuthorByUserID(ctx, sqlc.GetAuthorByUserIDParams{
		UserID:    userID,
		IsDeleted: isDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuthorNotFound
		}
		return nil, err
	}

	return &author, nil
}

// List implements [AuthorRepository].
func (r *repository) List(ctx context.Context, f *AuthorFilter) (*[]sqlc.Author, int64, error) {
	orderByValue := f.ToSQL()
	if orderByValue == "" {
		orderByValue = "created_at DESC"
	}

	args := []any{false}
	argIndex := 2

	query := `SELECT * FROM author WHERE is_deleted = $1`

	if f.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *f.UserID)
		argIndex++
	}
	if f.IsRecommended != nil {
		query += fmt.Sprintf(" AND is_recommended = $%d", argIndex)
		args = append(args, *f.IsRecommended)
		argIndex++
	}
	if f.SearchQuery != nil && *f.SearchQuery != "" {
		query += fmt.Sprintf(" AND stage_name ILIKE $%d", argIndex)
		args = append(args, "%"+*f.SearchQuery+"%")
		argIndex++
	}

	countQuery := strings.Replace(query, "SELECT *", "SELECT COUNT(*)", 1)
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderByValue, argIndex, argIndex+1)
	args = append(args, f.GetLimit(), f.GetOffset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var authors []sqlc.Author
	for rows.Next() {
		var author sqlc.Author
		if err := rows.Scan(
			&author.ID,
			&author.UserID,
			&author.MediaID,
			&author.StageName,
			&author.Gender,
			&author.Slug,
			&author.FirstName,
			&author.LastName,
			&author.DOB,
			&author.Phone,
			&author.Bio,
			&author.Description,
			&author.AcceptedTermsOfService,
			&author.EmailNewslettersAndChangelogs,
			&author.EmailPromotionsAndEvents,
			&author.IsRecommended,
			&author.IsDeleted,
			&author.Stats,
			&author.CreatedAt,
			&author.UpdatedAt,
			&author.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		authors = append(authors, author)
		// Cache the author in the cache
		r.cacheAuthor(ctx, &author)
	}

	countRows, err := r.pool.Query(ctx, countQuery, args[:len(args)-2]...)
	if err != nil {
		return nil, 0, err
	}
	defer countRows.Close()

	var totalCount int64
	if countRows.Next() {
		if err := countRows.Scan(&totalCount); err != nil {
			return nil, 0, err
		}
	}

	return &authors, totalCount, nil
}

// Update implements [AuthorRepository].
func (r *repository) Update(ctx context.Context, author *sqlc.Author) (*sqlc.Author, error) {
	r.invalidateCache(ctx, author.ID)

	result, err := r.q.UpdateAuthor(ctx, sqlc.UpdateAuthorParams{
		ID:                            author.ID,
		UserID:                        author.UserID,
		MediaID:                       author.MediaID,
		StageName:                     author.StageName,
		Gender:                        author.Gender,
		Slug:                          author.Slug,
		FirstName:                     author.FirstName,
		LastName:                      author.LastName,
		DOB:                           author.DOB,
		Phone:                         author.Phone,
		Bio:                           author.Bio,
		Description:                   author.Description,
		AcceptedTermsOfService:        author.AcceptedTermsOfService,
		EmailNewslettersAndChangelogs: author.EmailNewslettersAndChangelogs,
		EmailPromotionsAndEvents:      author.EmailPromotionsAndEvents,
		IsRecommended:                 author.IsRecommended,
		IsDeleted:                     author.IsDeleted,
		Stats:                         author.Stats,
		DeletedAt:                     author.DeletedAt,
	})
	if err != nil {
		return nil, err
	}

	r.cacheAuthor(ctx, &result)

	return &result, nil
}

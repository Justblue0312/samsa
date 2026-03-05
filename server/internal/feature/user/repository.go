package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/infras/cache"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	Create(ctx context.Context, user *sqlc.User) (*sqlc.User, error)
	GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*sqlc.User, error)
	GetByEmail(ctx context.Context, email string, includeDeleted bool) (*sqlc.User, error)
	Update(ctx context.Context, user *sqlc.User) (*sqlc.User, error)
}

type repository struct {
	q     *sqlc.Queries
	cfg   *config.Config
	cache *cache.Client
}

func NewRepository(q *sqlc.Queries, cfg *config.Config, c *cache.Client) Repository {
	return &repository{q: q, cfg: cfg, cache: c}
}
func buildUserKey(id uuid.UUID) string {
	return "user:" + id.String()
}

func buildUserEmailKey(email string) string {
	return "user:email:" + email
}

func (r *repository) cacheUser(ctx context.Context, user *sqlc.User) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildUserKey(user.ID),
		Value: user,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
	_ = r.cache.Set(ctx, &cache.Item{
		Key:   buildUserEmailKey(user.Email),
		Value: user,
		TTL:   r.cfg.Cache.QueryCacheTTL,
	})
}

func (r *repository) invalidateCache(ctx context.Context, id *uuid.UUID, email *string) {
	if !r.cfg.Cache.EnableCache {
		return
	}
	if id != nil {
		_ = r.cache.Delete(ctx, buildUserKey(*id))
	}
	if email != nil {
		_ = r.cache.Delete(ctx, buildUserEmailKey(*email))
	}
}

func (r *repository) Create(ctx context.Context, user *sqlc.User) (*sqlc.User, error) {
	params := sqlc.CreateUserParams{
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		PasswordHash:   user.PasswordHash,
		IsDeleted:      user.IsDeleted,
		IsActive:       user.IsActive,
		IsAdmin:        user.IsAdmin,
		IsStaff:        user.IsStaff,
		IsAuthor:       user.IsAuthor,
		IsBanned:       user.IsBanned,
		BannedAt:       user.BannedAt,
		BanReason:      user.BanReason,
		LastLoginAt:    user.LastLoginAt,
		RateLimitGroup: user.RateLimitGroup,
		CreatedAt:      common.Ptr(time.Now()),
	}

	result, err := r.q.CreateUser(ctx, params)
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	r.cacheUser(ctx, &result)

	return &result, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string, includeDeleted bool) (*sqlc.User, error) {
	if r.cfg.Cache.EnableCache {
		key := buildUserEmailKey(email)
		var user sqlc.User
		if err := r.cache.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	user, err := r.q.GetUserByEmail(ctx, sqlc.GetUserByEmailParams{
		Email:     email,
		IsDeleted: includeDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	r.cacheUser(ctx, &user)

	return &user, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*sqlc.User, error) {
	if r.cfg.Cache.EnableCache {
		key := buildUserKey(id)
		var user sqlc.User
		if err := r.cache.Get(ctx, key, &user); err == nil {
			return &user, nil
		}
	}

	user, err := r.q.GetUserByID(ctx, sqlc.GetUserByIDParams{
		UserID:    id,
		IsDeleted: includeDeleted,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	r.cacheUser(ctx, &user)

	return &user, nil
}

func (r *repository) Update(ctx context.Context, user *sqlc.User) (*sqlc.User, error) {
	r.invalidateCache(ctx, &user.ID, &user.Email)

	params := sqlc.UpdateUserParams{
		ID:             user.ID,
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		PasswordHash:   user.PasswordHash,
		IsDeleted:      user.IsDeleted,
		IsActive:       user.IsActive,
		IsAdmin:        user.IsAdmin,
		IsStaff:        user.IsStaff,
		IsAuthor:       user.IsAuthor,
		IsBanned:       user.IsBanned,
		BannedAt:       user.BannedAt,
		BanReason:      user.BanReason,
		LastLoginAt:    user.LastLoginAt,
		RateLimitGroup: user.RateLimitGroup,
		DeletedAt:      user.DeletedAt,
	}

	result, err := r.q.UpdateUser(ctx, params)
	if err != nil {
		if common.IsUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	r.cacheUser(ctx, &result)

	return &result, nil
}

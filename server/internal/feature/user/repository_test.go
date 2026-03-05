package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	pool := testkit.NewDB(t)
	queries := sqlc.New(pool)
	cfg := testkit.SetupConfig()
	repo := user.NewRepository(queries, cfg, nil)

	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		email := "test-create-" + uuid.New().String()[:8] + "@example.com"
		u := &sqlc.User{
			Email:          email,
			PasswordHash:   "hash",
			RateLimitGroup: sqlc.RateLimitGroupDefault,
		}

		result, err := repo.Create(ctx, u)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, email, strings.TrimSpace(result.Email))
	})

	t.Run("returns ErrEmailTaken when email exists", func(t *testing.T) {
		existing := factory.User(t, pool, factory.UserOpts{})

		u := &sqlc.User{
			Email:          existing.Email,
			PasswordHash:   "hash",
			RateLimitGroup: sqlc.RateLimitGroupDefault,
		}

		result, err := repo.Create(ctx, u)
		assert.ErrorIs(t, err, user.ErrEmailTaken)
		assert.Nil(t, result)
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	pool := testkit.NewDB(t)
	queries := sqlc.New(pool)
	cfg := testkit.SetupConfig()
	repo := user.NewRepository(queries, cfg, nil)

	ctx := context.Background()

	t.Run("returns user when found", func(t *testing.T) {
		existing := factory.User(t, pool, factory.UserOpts{})

		result, err := repo.GetByID(ctx, existing.ID, false)
		require.NoError(t, err)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.Email, result.Email)
	})

	t.Run("returns ErrNotFound when not found", func(t *testing.T) {
		result, err := repo.GetByID(ctx, uuid.New(), false)
		assert.ErrorIs(t, err, user.ErrNotFound)
		assert.Nil(t, result)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	pool := testkit.NewDB(t)
	queries := sqlc.New(pool)
	cfg := testkit.SetupConfig()
	repo := user.NewRepository(queries, cfg, nil)

	ctx := context.Background()

	t.Run("returns user by email", func(t *testing.T) {
		existing := factory.User(t, pool, factory.UserOpts{})

		result, err := repo.GetByEmail(ctx, existing.Email, false)
		require.NoError(t, err)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.Email, result.Email)
	})

	t.Run("returns ErrNotFound when not found", func(t *testing.T) {
		result, err := repo.GetByEmail(ctx, "nonexistent@example.com", false)
		assert.ErrorIs(t, err, user.ErrNotFound)
		assert.Nil(t, result)
	})
}

func TestUserRepository_Update(t *testing.T) {
	pool := testkit.NewDB(t)
	queries := sqlc.New(pool)
	cfg := testkit.SetupConfig()
	repo := user.NewRepository(queries, cfg, nil)

	ctx := context.Background()

	t.Run("updates user successfully", func(t *testing.T) {
		existing := factory.User(t, pool, factory.UserOpts{})

		// Use the retrieved DB item so we have all required fields (like password hash)
		toUpdate, err := repo.GetByID(ctx, existing.ID, false)
		require.NoError(t, err)

		toUpdate.IsActive = false
		toUpdate.RateLimitGroup = sqlc.RateLimitGroupRestricted

		result, err := repo.Update(ctx, toUpdate)
		require.NoError(t, err)

		assert.False(t, result.IsActive)
		assert.Equal(t, sqlc.RateLimitGroupRestricted, result.RateLimitGroup)
	})

	t.Run("returns ErrEmailTaken on unique constraint violation", func(t *testing.T) {
		user1 := factory.User(t, pool, factory.UserOpts{})
		user2 := factory.User(t, pool, factory.UserOpts{})

		// Try to update user2's email to user1's email
		toUpdate, err := repo.GetByID(ctx, user2.ID, false)
		require.NoError(t, err)

		toUpdate.Email = user1.Email

		result, err := repo.Update(ctx, toUpdate)
		assert.ErrorIs(t, err, user.ErrEmailTaken)
		assert.Nil(t, result)
	})
}

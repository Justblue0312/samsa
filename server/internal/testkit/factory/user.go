package factory

import (
	"context"
	"testing"

	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// UserOpts controls which fields are customised when creating a test user.
// Any zero-value field gets a sensible default.
type UserOpts struct {
	Email          string
	PasswordHash   string
	IsActive       bool
	IsAdmin        bool
	RateLimitGroup sqlc.RateLimitGroup
}

// User inserts a user into the DB and returns the created model.
func User(t *testing.T, db sqlc.DBTX, opts UserOpts) *sqlc.User {
	t.Helper()

	if opts.Email == "" {
		opts.Email = randEmail()
	}
	if opts.RateLimitGroup == "" {
		opts.RateLimitGroup = sqlc.RateLimitGroupDefault
	}

	q := sqlc.New(db)
	n := now()

	user, err := q.CreateUser(context.Background(), sqlc.CreateUserParams{
		Email:          opts.Email,
		EmailVerified:  false,
		PasswordHash:   opts.PasswordHash,
		IsDeleted:      false,
		IsActive:       true,
		IsAdmin:        opts.IsAdmin,
		IsStaff:        false,
		IsAuthor:       false,
		IsBanned:       false,
		BannedAt:       nil,
		BanReason:      nil,
		LastLoginAt:    n,
		RateLimitGroup: opts.RateLimitGroup,
		CreatedAt:      n,
	})
	require.NoError(t, err, "factory: failed to create test user")

	return &user
}

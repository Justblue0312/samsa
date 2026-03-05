package oauthaccount

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
)

type Repository interface {
	GetByProviderAndUserID(ctx context.Context, provider sqlc.OAuthProvider, userID uuid.UUID) (*[]sqlc.OAuthAccount, error)
	CountOtherAccounts(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID) (int64, error)
	Create(ctx context.Context, account *sqlc.OAuthAccount) (*sqlc.OAuthAccount, error)
	Delete(ctx context.Context, account *sqlc.OAuthAccount) error
}

type repository struct {
	q   *sqlc.Queries
	cfg *config.Config
}

func NewRepository(q *sqlc.Queries, cfg *config.Config) Repository {
	return &repository{q: q, cfg: cfg}
}

func (r *repository) CountOtherAccounts(ctx context.Context, userID uuid.UUID, accountIDs []uuid.UUID) (int64, error) {
	count, err := r.q.CountOtherAccounts(ctx, sqlc.CountOtherAccountsParams{
		UserID:     userID,
		AccountIds: accountIDs,
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) GetByProviderAndUserID(ctx context.Context, provider sqlc.OAuthProvider, userID uuid.UUID) (*[]sqlc.OAuthAccount, error) {
	accounts, err := r.q.GetAccountByProviderAndUserID(ctx, sqlc.GetAccountByProviderAndUserIDParams{
		Provider: provider,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}
	return &accounts, nil
}

func (r *repository) Create(ctx context.Context, account *sqlc.OAuthAccount) (*sqlc.OAuthAccount, error) {
	acc, err := r.q.CreateAccount(ctx, sqlc.CreateAccountParams{
		UserID:                account.UserID,
		Provider:              account.Provider,
		AccessToken:           account.AccessToken,
		ExpiresAt:             account.ExpiresAt,
		RefreshToken:          account.RefreshToken,
		RefreshTokenExpiresAt: account.RefreshTokenExpiresAt,
		AccountID:             account.AccountID,
		AccountEmail:          account.AccountEmail,
		AccountUsername:       account.AccountUsername,
		AccountAvatarUrl:      account.AccountAvatarUrl,
		CreatedAt:             common.Ptr(time.Now()),
	})
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *repository) Delete(ctx context.Context, account *sqlc.OAuthAccount) error {
	return r.q.DeleteAccountByID(ctx, account.ID)
}

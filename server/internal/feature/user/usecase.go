package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	oauthaccount "github.com/justblue/samsa/internal/feature/oauth_account"
	"github.com/justblue/samsa/internal/infras/cache"
	"github.com/justblue/samsa/internal/infras/redis"
)

var (
	ErrNotFound                       = errors.New("user not found")
	ErrEmailTaken                     = errors.New("email already taken")
	ErrBanned                         = errors.New("user is banned")
	ErrOAuthAccountNotFound           = errors.New("oauth account not found")
	ErrCannotDisconnectLastAuthMethod = errors.New(`
cannot disconnect last auth method as it is the only authentication method.
please verify you email or connect another authentication method before disconnecting this one.`)
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	DisconnectOAuthAccountProvider(ctx context.Context, user *sqlc.User, provider sqlc.OAuthProvider) error
}

type usecase struct {
	cfg              *config.Config
	cache            *cache.Client
	presence         *redis.PresenceStore
	userRepo         Repository
	oauthAccountRepo oauthaccount.Repository
}

func NewUseCase(cfg *config.Config, cache *cache.Client, userRepo Repository, oauthAccountRepo oauthaccount.Repository, presence *redis.PresenceStore) UseCase {
	return &usecase{
		cfg:              cfg,
		cache:            cache,
		presence:         presence,
		userRepo:         userRepo,
		oauthAccountRepo: oauthAccountRepo,
	}
}

func (u *usecase) DisconnectOAuthAccountProvider(ctx context.Context, user *sqlc.User, provider sqlc.OAuthProvider) error {
	oauthAccounts, err := u.oauthAccountRepo.GetByProviderAndUserID(ctx, provider, user.ID)
	if err != nil {
		return err
	}
	if oauthAccounts != nil && len(*oauthAccounts) == 0 {
		return ErrOAuthAccountNotFound
	}

	otherAccountIds := make([]uuid.UUID, 0, len(*oauthAccounts))
	for _, account := range *oauthAccounts {
		otherAccountIds = append(otherAccountIds, account.ID)
	}

	countOtherAccounts, err := u.oauthAccountRepo.CountOtherAccounts(ctx, user.ID, otherAccountIds)
	if err != nil {
		return err
	}

	if countOtherAccounts == 0 && !user.EmailVerified {
		return ErrCannotDisconnectLastAuthMethod
	}

	for _, account := range *oauthAccounts {
		if err := u.oauthAccountRepo.Delete(ctx, &account); err != nil {
			return err
		}
	}

	return nil
}

func (u *usecase) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	return u.presence.IsOnline(ctx, userID)
}

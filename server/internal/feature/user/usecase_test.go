package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	oauthaccountMocks "github.com/justblue/samsa/internal/feature/oauth_account/mocks"
	"github.com/justblue/samsa/internal/feature/user"
	userMocks "github.com/justblue/samsa/internal/feature/user/mocks"
	"github.com/justblue/samsa/internal/infras/redis"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUseCase_DisconnectOAuthAccountProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cfg := testkit.SetupConfig()

	rOpts := redis.NewRedisOpts(cfg)
	rdb, _ := redis.New(ctx, rOpts)

	mockUserRepo := userMocks.NewMockRepository(ctrl)
	mockOAuthRepo := oauthaccountMocks.NewMockRepository(ctrl)
	presenceStore := redis.NewPresenceStore(rdb)

	uc := user.NewUseCase(cfg, nil, mockUserRepo, mockOAuthRepo, presenceStore)

	t.Run("successfully disconnects provider", func(t *testing.T) {
		u := &sqlc.User{
			ID:            uuid.New(),
			EmailVerified: true,
		}

		provider := sqlc.OauthProviderGoogle
		accounts := &[]sqlc.OAuthAccount{
			{ID: uuid.New(), UserID: u.ID, Provider: provider},
		}

		mockOAuthRepo.EXPECT().
			GetByProviderAndUserID(ctx, provider, u.ID).
			Return(accounts, nil)

		mockOAuthRepo.EXPECT().
			CountOtherAccounts(ctx, u.ID, gomock.Any()).
			Return(int64(0), nil)

		// Verification passed (email verified is true), so we expect delete
		mockOAuthRepo.EXPECT().
			Delete(ctx, &(*accounts)[0]).
			Return(nil)

		err := uc.DisconnectOAuthAccountProvider(ctx, u, provider)
		require.NoError(t, err)
	})

	t.Run("fails when provider accounts don't exist", func(t *testing.T) {
		u := &sqlc.User{ID: uuid.New()}
		provider := sqlc.OauthProviderGithub

		mockOAuthRepo.EXPECT().
			GetByProviderAndUserID(ctx, provider, u.ID).
			Return(&[]sqlc.OAuthAccount{}, nil)

		err := uc.DisconnectOAuthAccountProvider(ctx, u, provider)
		assert.ErrorIs(t, err, user.ErrOAuthAccountNotFound)
	})

	t.Run("fails when disconnecting last auth method", func(t *testing.T) {
		u := &sqlc.User{
			ID:            uuid.New(),
			EmailVerified: false, // The user has no verified email, making this their only login method
		}

		provider := sqlc.OauthProviderGithub
		accounts := &[]sqlc.OAuthAccount{
			{ID: uuid.New(), UserID: u.ID, Provider: provider},
		}

		mockOAuthRepo.EXPECT().
			GetByProviderAndUserID(ctx, provider, u.ID).
			Return(accounts, nil)

		// Meaning they have *no other* oauth accounts
		mockOAuthRepo.EXPECT().
			CountOtherAccounts(ctx, u.ID, gomock.Any()).
			Return(int64(0), nil)

		err := uc.DisconnectOAuthAccountProvider(ctx, u, provider)
		assert.ErrorIs(t, err, user.ErrCannotDisconnectLastAuthMethod)
	})
}

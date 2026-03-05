package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/auth"
	oauthaccount "github.com/justblue/samsa/internal/feature/oauth_account"
	"github.com/justblue/samsa/internal/feature/session"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/internal/transport/worker"
	"github.com/justblue/samsa/pkg/security/token"
	"github.com/justblue/samsa/pkg/subject"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type UseCase interface {
	ProcessCallback(ctx context.Context, accessToken, refreshToken string, expiresAt *int64, amd *auth.SessionMetadata) (*auth.UserSessionInfo, error)
}

type usecase struct {
	q            *sqlc.Queries
	cfg          *config.Config
	userRepo     user.Repository
	sessionRepo  session.Repository
	accountRepo  oauthaccount.Repository
	workerClient worker.Client
}

func NewUseCase(
	q *sqlc.Queries,
	cfg *config.Config,
	userRepo user.Repository,
	sessionRepo session.Repository,
	accountRepo oauthaccount.Repository,
	workerClient worker.Client,
) UseCase {
	return &usecase{
		q:            q,
		cfg:          cfg,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		accountRepo:  accountRepo,
		workerClient: workerClient,
	}
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

func (u *usecase) ProcessCallback(ctx context.Context, accessToken, refreshToken string, expiresAt *int64, amd *auth.SessionMetadata) (*auth.UserSessionInfo, error) {
	userInfo, err := u.getGoogleUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get google user info: %w", err)
	}

	accountID := userInfo.ID

	existingAccount, err := u.q.GetAccountByProviderAndAccountID(ctx, sqlc.GetAccountByProviderAndAccountIDParams{
		Provider:  sqlc.OauthProviderGoogle,
		AccountID: accountID,
	})

	var userModel *sqlc.User

	if err == nil && existingAccount.UserID != uuid.Nil {
		uModel, err := u.userRepo.GetByID(ctx, existingAccount.UserID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		userModel = uModel

		_, err = u.q.UpdateAccount(ctx, sqlc.UpdateAccountParams{
			ID:                    existingAccount.ID,
			UserID:                existingAccount.UserID,
			Provider:              existingAccount.Provider,
			AccessToken:           accessToken,
			ExpiresAt:             common.UnixToPtrTime(expiresAt),
			RefreshToken:          &refreshToken,
			RefreshTokenExpiresAt: common.UnixToPtrTimeWithDelta(expiresAt, 30*24*time.Hour),
			AccountID:             accountID,
			AccountEmail:          &userInfo.Email,
			AccountUsername:       &userInfo.Email,
			AccountAvatarUrl:      &userInfo.Picture,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update oauth account: %w", err)
		}
	} else if err == pgx.ErrNoRows {
		uModel, err := u.userRepo.GetByEmail(ctx, userInfo.Email, false)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("failed to check user email: %w", err)
		}

		if err == pgx.ErrNoRows {
			newUser, err := u.q.CreateUser(ctx, sqlc.CreateUserParams{
				Email:         userInfo.Email,
				EmailVerified: true,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
			userModel = &newUser
		} else {
			userModel = uModel
		}

		_, err = u.accountRepo.Create(ctx, &sqlc.OAuthAccount{
			UserID:                userModel.ID,
			Provider:              sqlc.OauthProviderGoogle,
			AccessToken:           accessToken,
			ExpiresAt:             common.UnixToPtrTime(expiresAt),
			RefreshToken:          &refreshToken,
			RefreshTokenExpiresAt: common.UnixToPtrTimeWithDelta(expiresAt, 30*24*time.Hour),
			AccountID:             accountID,
			AccountEmail:          &userInfo.Email,
			AccountUsername:       &userInfo.Email,
			AccountAvatarUrl:      &userInfo.Picture,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create oauth account: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to check existing oauth account: %w", err)
	}

	userModel.EmailVerified = true
	if _, err = u.userRepo.Update(ctx, userModel); err != nil {
		return nil, fmt.Errorf("failed to update user model: %w", err)
	}

	if err := u.workerClient.Enqueue(ctx, user.NewTaskOnAfterSignUp(), user.OnAfterSignUpPayload{UserID: userModel.ID}); err != nil {
		return nil, fmt.Errorf("failed to enqueue task on after sign up: %w", err)
	}

	return u.createSession(ctx, userModel, amd)
}

func (u *usecase) createSession(ctx context.Context, user *sqlc.User, amd *auth.SessionMetadata) (*auth.UserSessionInfo, error) {
	var userInfo auth.UserSessionInfo
	scopes := []string{string(subject.WebReadScope), string(subject.WebWriteScope)}
	token, hash, err := token.GenerateTokenHashPair(u.cfg.SecretKey, u.cfg.UserSessionTokenPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate token")
	}
	var md []byte
	if amd.Metadata != nil {
		md, err = json.Marshal(amd.Metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal session metadata")
		}
	}
	now := time.Now().UTC()
	expiredAt := now.Add(u.cfg.UserSessionTTL).UTC()

	sess, err := u.sessionRepo.Create(ctx, &sqlc.Session{
		UserID:     user.ID,
		Token:      hash,
		IPAddress:  amd.IPAddress,
		UserAgent:  amd.UserAgent,
		DeviceInfo: amd.DeviceInfo,
		Scopes:     scopes,
		Metadata:   md,
		IsActive:   true,
		ExpiresAt:  &expiredAt,
		CreatedAt:  &now,
		UpdatedAt:  &time.Time{},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session")
	}

	userInfo = auth.UserSessionInfo{
		Token:   token,
		User:    user,
		Session: sess,
	}
	return &userInfo, nil
}

func (u *usecase) getGoogleUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

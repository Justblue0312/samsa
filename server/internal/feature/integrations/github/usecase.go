package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	"github.com/pkg/errors"
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

type GitHubUserInfo struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

func (u *usecase) ProcessCallback(ctx context.Context, accessToken, refreshToken string, expiresAt *int64, amd *auth.SessionMetadata) (*auth.UserSessionInfo, error) {
	userInfo, err := u.getGitHubUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get github user info: %w", err)
	}

	accountID := fmt.Sprintf("%d", userInfo.ID)

	existingAccount, err := u.q.GetAccountByProviderAndAccountID(ctx, sqlc.GetAccountByProviderAndAccountIDParams{
		Provider:  sqlc.OauthProviderGithub,
		AccountID: accountID,
	})

	var userModel *sqlc.User

	if err == nil && existingAccount.UserID != uuid.Nil {
		// Existing account, get user
		uModel, err := u.userRepo.GetByID(ctx, existingAccount.UserID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		userModel = uModel

		// Update account tokens
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
			AccountUsername:       &userInfo.Login,
			AccountAvatarUrl:      &userInfo.AvatarURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update oauth account: %w", err)
		}
	} else if err == pgx.ErrNoRows {
		// New account, find or create user by email
		uModel, err := u.userRepo.GetByEmail(ctx, userInfo.Email, false)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("failed to check user email: %w", err)
		}

		if err == pgx.ErrNoRows {
			// Create new user
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

		// Create oauth account
		_, err = u.accountRepo.Create(ctx, &sqlc.OAuthAccount{
			UserID:                userModel.ID,
			Provider:              sqlc.OauthProviderGithub,
			AccessToken:           accessToken,
			ExpiresAt:             common.UnixToPtrTime(expiresAt),
			RefreshToken:          &refreshToken,
			RefreshTokenExpiresAt: common.UnixToPtrTimeWithDelta(expiresAt, 30*24*time.Hour),
			AccountID:             accountID,
			AccountEmail:          &userInfo.Email,
			AccountUsername:       &userInfo.Login,
			AccountAvatarUrl:      &userInfo.AvatarURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create oauth account: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to check existing oauth account: %w", err)
	}

	userModel.EmailVerified = true
	if _, err := u.userRepo.Update(ctx, userModel); err != nil {
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

func (u *usecase) getGitHubUserInfo(ctx context.Context, accessToken string) (*GitHubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	if userInfo.Email == "" {
		email, err := u.getGitHubPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, err
		}
		userInfo.Email = email
	}

	return &userInfo, nil
}

func (u *usecase) getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no primary verified email found")
}

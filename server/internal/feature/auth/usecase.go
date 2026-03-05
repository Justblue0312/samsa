package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/session"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/internal/infras/cache"
	"github.com/justblue/samsa/internal/transport/worker"
	"github.com/justblue/samsa/pkg/security/pwd"
	"github.com/justblue/samsa/pkg/security/token"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/pkg/errors"
)

const (
	verificationCodeLength = 6
	forgotPasswordTTL      = 15
)

var (
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidSession          = errors.New("invalid session")
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	ErrExpiredLink             = errors.New("link is expired")
	ErrInvalidCode             = errors.New("invalid code")
)

type SessionMetadata struct {
	Metadata   *map[string]string `json:"metadata"`
	IPAddress  *string            `json:"ip_address"`
	UserAgent  *string            `json:"user_agent"`
	DeviceInfo *string            `json:"device_info"`
}

type UserSessionInfo struct {
	Token   string
	User    *sqlc.User
	Session *sqlc.Session
}

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks
type UseCase interface {
	Login(ctx context.Context, req *LoginRequest, amd *SessionMetadata) (*UserSessionInfo, error)
	Logout(ctx context.Context, user *sqlc.User, sess *sqlc.Session) error
	Register(ctx context.Context, req *RegisterRequest, amd *SessionMetadata) (*UserSessionInfo, error)

	ChangePassword(ctx context.Context, user *sqlc.User, req *ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *ResetPasswordRequest, code string, amd *SessionMetadata) (*UserSessionInfo, error)

	SendVerificationEmail(ctx context.Context, user *sqlc.User) error
	ConfirmVerificationEmail(ctx context.Context, code string, user *sqlc.User) error
}

type usecase struct {
	userRepo    user.Repository
	sessionRepo session.Repository
	cfg         *config.Config
	cache       *cache.Client
	worker      worker.Client
}

func NewUsecase(
	cfg *config.Config,
	cache *cache.Client,
	workerClient worker.Client,
	userRepo user.Repository,
	sessionRepo session.Repository,
) UseCase {
	return &usecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
		cache:       cache,
		worker:      workerClient,
	}
}

func (u *usecase) createSession(ctx context.Context, user *sqlc.User, amd *SessionMetadata) (*UserSessionInfo, error) {
	var userInfo UserSessionInfo
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

	userInfo = UserSessionInfo{
		Token:   token,
		User:    user,
		Session: sess,
	}
	return &userInfo, nil
}

func (u *usecase) sendEmail(ctx context.Context, toEmail, template, emailSubject, url string, expiresIn time.Duration) error {
	key := fmt.Sprintf("%s:%s", template, toEmail)
	code, _ := common.GenerateCode(verificationCodeLength)

	err := u.cache.Set(ctx, &cache.Item{
		Key:   key,
		Value: code,
		TTL:   expiresIn,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to store verification code")
	}

	returnTo := u.cfg.GetReturnURL(fmt.Sprintf("%s?code=%s", url, code))
	pl := SendMailPayload{
		Email:        toEmail,
		TemplateName: template,
		Subject:      emailSubject,
		Metadata: map[string]string{
			"email":     toEmail,
			"url":       returnTo,
			"expiresIn": expiresIn.String(),
		},
	}
	err = u.worker.Enqueue(ctx, NewTaskSendEmailDefinition(), pl)
	if err != nil {
		return errors.Wrapf(err, "failed to enqueue email task")
	}

	return nil
}

func (u *usecase) Login(ctx context.Context, req *LoginRequest, amd *SessionMetadata) (*UserSessionInfo, error) {
	user, err := u.userRepo.GetByEmail(ctx, req.Email, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user by email")
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := pwd.Verify(user.PasswordHash, req.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return u.createSession(ctx, user, amd)
}

func (u *usecase) Logout(ctx context.Context, usr *sqlc.User, sess *sqlc.Session) error {
	err := u.sessionRepo.DeleteByUserId(ctx, usr.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete session")
	}
	return nil
}

func (u *usecase) Register(ctx context.Context, req *RegisterRequest, amd *SessionMetadata) (*UserSessionInfo, error) {
	existingUser, err := u.userRepo.GetByEmail(ctx, req.Email, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check existing user")
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	pwdHash, err := pwd.Hash(req.Password)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to hash password")
	}

	user, err := u.userRepo.Create(ctx, &sqlc.User{
		Email:          req.Email,
		PasswordHash:   pwdHash,
		RateLimitGroup: sqlc.RateLimitGroupDefault,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create user")
	}

	return u.createSession(ctx, user, amd)
}

func (u *usecase) SendVerificationEmail(ctx context.Context, usr *sqlc.User) error {
	return u.sendEmail(ctx, usr.Email, "verify_email.html", "Verify your email address", "/verification-email", time.Minute)
}

func (u *usecase) ConfirmVerificationEmail(ctx context.Context, code string, usr *sqlc.User) error {
	key := "verify_email:" + usr.ID.String()
	var storedCode string
	err := u.cache.Get(ctx, key, storedCode)
	if err != nil {
		return ErrExpiredLink
	}

	if storedCode != code {
		return ErrInvalidVerificationCode
	}

	_, err = u.userRepo.Update(ctx, &sqlc.User{
		ID:            usr.ID,
		EmailVerified: true,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to mark user email as verified")
	}

	err = u.cache.Delete(ctx, key)
	if err != nil {
		return errors.Wrapf(err, "failed to delete verification code")
	}

	return nil
}

func (u *usecase) ChangePassword(ctx context.Context, usr *sqlc.User, req *ChangePasswordRequest) error {
	pwdHash, err := pwd.Hash(req.NewPassword)
	if err != nil {
		return errors.Wrapf(err, "failed to hash new password")
	}

	usr.PasswordHash = pwdHash
	_, err = u.userRepo.Update(ctx, usr)
	if err != nil {
		return errors.Wrapf(err, "failed to update user password")
	}

	return nil
}

func (u *usecase) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	return u.sendEmail(ctx, req.Email, "forgot_password.html", "Reset Password", "/reset-password", time.Minute*forgotPasswordTTL)
}

func (u *usecase) ResetPassword(ctx context.Context, req *ResetPasswordRequest, code string, amd *SessionMetadata) (*UserSessionInfo, error) {
	key := "forgot_password:" + req.Email
	var storedCode string
	err := u.cache.Get(ctx, key, storedCode)
	if err != nil {
		return nil, ErrExpiredLink
	}
	if code != storedCode {
		return nil, ErrInvalidCode
	}

	pwdHashed, err := pwd.Hash(req.Password)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to hash password")
	}

	user, err := u.userRepo.Update(ctx, &sqlc.User{
		Email:        req.Email,
		PasswordHash: pwdHashed,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update user password")
	}

	err = u.cache.Delete(ctx, key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete reset code")
	}

	return u.createSession(ctx, user, amd)
}

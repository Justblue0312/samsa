package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/auth"
	authMocks "github.com/justblue/samsa/internal/feature/auth/mocks"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupTest(t *testing.T) (*authMocks.MockUseCase, *auth.HTTPHandler) {
	ctrl := gomock.NewController(t)
	mockUsecase := authMocks.NewMockUseCase(ctrl)
	cfg := &config.Config{
		UserSessionCookieName: "samsa_session",
		UserSessionDomain:     "localhost",
		UserSessionTTL:        24 * time.Hour,
	}
	v := validator.New()
	handler := auth.NewHTTPHandler(cfg, v, mockUsecase)

	// In Go 1.20+, testing.T has Cleanup built-in that finishes the ctrl
	t.Cleanup(func() {
		ctrl.Finish()
	})

	return mockUsecase, handler
}

func TestHTTPHandler_Login(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	t.Run("success", func(t *testing.T) {
		reqBody := auth.LoginRequest{
			Email:    "test@example.com",
			Password: "Password123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		userInfo := &auth.UserSessionInfo{
			Token: "test-token",
			User:  &sqlc.User{ID: uuid.New()},
		}

		mockUsecase.EXPECT().
			Login(gomock.Any(), &reqBody, gomock.Any()).
			Return(userInfo, nil)

		handler.Login(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusFound, res.StatusCode)

		cookies := res.Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "test-token", cookies[0].Value)
	})

	t.Run("invalid credentials returns unauthorized", func(t *testing.T) {
		reqBody := auth.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			Login(gomock.Any(), &reqBody, gomock.Any()).
			Return(nil, auth.ErrInvalidCredentials)

		handler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error on bad request", func(t *testing.T) {
		reqBody := auth.LoginRequest{
			Email: "invalid-email",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_Logout(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	testUser := &sqlc.User{ID: uuid.New()}
	testSession := &sqlc.Session{ID: uuid.New()}
	subj := &subject.AuthSubject{
		User:    testUser,
		Session: testSession,
		Scopes:  []subject.Scope{subject.WebReadScope},
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			Logout(gomock.Any(), testUser, testSession).
			Return(nil)

		handler.Logout(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusFound, res.StatusCode)

		cookies := res.Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "", cookies[0].Value)
		assert.True(t, cookies[0].Expires.Before(time.Now()))
	})

	t.Run("fails when subject not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()

		handler.Logout(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHTTPHandler_Register(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	t.Run("success", func(t *testing.T) {
		reqBody := auth.RegisterRequest{
			Email:    "new@example.com",
			Password: "Password123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		userInfo := &auth.UserSessionInfo{
			Token: "new-token",
			User:  &sqlc.User{ID: uuid.New()},
		}

		mockUsecase.EXPECT().
			Register(gomock.Any(), &reqBody, gomock.Any()).
			Return(userInfo, nil)

		handler.Register(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusFound, res.StatusCode)

		cookies := res.Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "new-token", cookies[0].Value)
	})

	t.Run("fails if email is taken", func(t *testing.T) {
		reqBody := auth.RegisterRequest{
			Email:    "existing@example.com",
			Password: "Password123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			Register(gomock.Any(), &reqBody, gomock.Any()).
			Return(nil, user.ErrEmailTaken)

		handler.Register(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestHTTPHandler_SendVerificationEmail(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	testUser := &sqlc.User{ID: uuid.New()}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/verify/send", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			SendVerificationEmail(gomock.Any(), testUser).
			Return(nil)

		handler.SendVerificationEmail(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_ChangePassword(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	testUser := &sqlc.User{ID: uuid.New(), PasswordHash: "hash"}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("success", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			OldPassword: "OldPassword1!",
			NewPassword: "NewPassword1!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/password/change", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			ChangePassword(gomock.Any(), testUser, &reqBody).
			Return(nil)

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails with incorrect current password", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			OldPassword: "WrongPassword!",
			NewPassword: "NewPassword1!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/password/change", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			ChangePassword(gomock.Any(), testUser, &reqBody).
			Return(auth.ErrInvalidCredentials)

		handler.ChangePassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHTTPHandler_ForgotPassword(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	t.Run("success", func(t *testing.T) {
		reqBody := auth.ForgotPasswordRequest{
			Email: "test@example.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/password/forgot", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			ForgotPassword(gomock.Any(), &reqBody).
			Return(nil)

		handler.ForgotPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_ResetPassword(t *testing.T) {
	mockUsecase, handler := setupTest(t)

	t.Run("success", func(t *testing.T) {
		reqBody := auth.ResetPasswordRequest{
			Email:    "test@example.com",
			Password: "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/password/reset?code=valid-code", bytes.NewReader(body))

		// Setup chi routing context to provide URLParam("code")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "valid-code")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		userInfo := &auth.UserSessionInfo{
			Token: "reset-token",
			User:  &sqlc.User{ID: uuid.New()},
		}

		mockUsecase.EXPECT().
			ResetPassword(gomock.Any(), &reqBody, "valid-code", gomock.Any()).
			Return(userInfo, nil)

		handler.ResetPassword(w, req)

		res := w.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusFound, res.StatusCode)

		cookies := res.Cookies()
		require.NotEmpty(t, cookies)
		assert.Equal(t, "reset-token", cookies[0].Value)
	})

	t.Run("fails with invalid or expired code", func(t *testing.T) {
		reqBody := auth.ResetPasswordRequest{
			Email:    "test@example.com",
			Password: "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/password/reset?code=invalid-code", bytes.NewReader(body))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("code", "invalid-code")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			ResetPassword(gomock.Any(), &reqBody, "invalid-code", gomock.Any()).
			Return(nil, auth.ErrExpiredLink)

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

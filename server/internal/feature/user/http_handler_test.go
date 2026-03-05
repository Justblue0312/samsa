package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/user"
	userMocks "github.com/justblue/samsa/internal/feature/user/mocks"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHTTPHandler_DisconnectProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := userMocks.NewMockUseCase(ctrl)
	handler := user.NewHTTPHandler(mockUsecase, nil, nil) // no validator or cfg needed here

	testUser := &sqlc.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		EmailVerified: true,
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("successfully disconnects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/?provider=github", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			DisconnectOAuthAccountProvider(gomock.Any(), gomock.Any(), sqlc.OauthProviderGithub).
			Return(nil)

		handler.DisconnectProvider(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]string
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "OAuth account provider disconnected successfully", response["message"])
	})

	t.Run("returns conflict when disconnecting last auth method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/?provider=github", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			DisconnectOAuthAccountProvider(gomock.Any(), gomock.Any(), sqlc.OauthProviderGithub).
			Return(user.ErrCannotDisconnectLastAuthMethod)

		handler.DisconnectProvider(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("returns not found when account doesn't exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/?provider=google", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		mockUsecase.EXPECT().
			DisconnectOAuthAccountProvider(gomock.Any(), gomock.Any(), sqlc.OauthProviderGoogle).
			Return(user.ErrOAuthAccountNotFound)

		handler.DisconnectProvider(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns unauth when subject missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/?provider=google", nil)
		w := httptest.NewRecorder()

		handler.DisconnectProvider(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns validation error on bad query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/?provider=invalid", nil)
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, subj)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DisconnectProvider(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

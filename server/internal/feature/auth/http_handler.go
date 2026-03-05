package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/session"
	"github.com/justblue/samsa/internal/feature/user"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
)

func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, user.ErrNotFound) || errors.Is(err, session.ErrNotFound) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	if errors.Is(err, user.ErrBanned) || errors.Is(err, ErrExpiredLink) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, user.ErrEmailTaken) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrInvalidSession) || errors.Is(err, ErrInvalidVerificationCode) || errors.Is(err, ErrInvalidCode) {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	respond.Error(w, apierror.Internal())
}

type HTTPHandler struct {
	u         UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func NewHTTPHandler(cfg *config.Config, v *validator.Validate, u UseCase) *HTTPHandler {
	return &HTTPHandler{
		u:         u,
		cfg:       cfg,
		validator: v,
	}
}

func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	userInfo, err := h.u.Login(ctx, &req, getMetadata(r))
	if err != nil {
		mapError(w, err)
		return
	}

	h.HandleLoginResponse(w, r, userInfo, config.EmptyPath)
}

func (h *HTTPHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if err := h.u.Logout(ctx, subj.User, subj.Session); err != nil {
		mapError(w, err)
		return
	}

	h.HandleLoginResponse(w, r, nil, "/")
}

func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	userInfo, err := h.u.Register(ctx, &req, getMetadata(r))
	if err != nil {
		mapError(w, err)
		return
	}

	h.HandleLoginResponse(w, r, userInfo, config.EmptyPath)
}

func (h *HTTPHandler) SendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if err := h.u.SendVerificationEmail(ctx, subj.User); err != nil {
		slog.Error("failed to send email",
			"to", subj.User.Email,
			"error", err.Error(),
		)
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *HTTPHandler) ConfirmVerificationEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := chi.URLParam(r, "code")
	if code == "" {
		respond.Error(w, apierror.BadRequest("missing code"))
		return
	}

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if err := h.u.ConfirmVerificationEmail(ctx, code, subj.User); err != nil {
		slog.Error("failed to send email",
			"to", subj.User.Email,
			"error", err.Error(),
		)
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *HTTPHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.u.ChangePassword(ctx, subj.User, &req); err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *HTTPHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.u.ForgotPassword(ctx, &req); err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *HTTPHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := chi.URLParam(r, "code")
	if code == "" {
		respond.Error(w, apierror.BadRequest("missing code"))
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	userInfo, err := h.u.ResetPassword(ctx, &req, code, getMetadata(r))
	if err != nil {
		mapError(w, err)
		return
	}

	h.HandleLoginResponse(w, r, userInfo, config.EmptyPath)
}

func (h *HTTPHandler) HandleLoginResponse(w http.ResponseWriter, r *http.Request, userInfo *UserSessionInfo, returnTo string) {
	returnURL := h.cfg.GetReturnURL(returnTo)
	secure := h.cfg.IsProduction()

	var expiresAt time.Time
	if userInfo != nil && userInfo.Session != nil && userInfo.Session.ExpiresAt != nil {
		expiresAt = *userInfo.Session.ExpiresAt
	} else {
		expiresAt = time.Now().Add(h.cfg.UserSessionTTL)
		// If logging out or session missing, we want to clear the cookie.
		if userInfo == nil || userInfo.Token == "" {
			expiresAt = time.Now().Add(-24 * time.Hour)
		}
	}

	token := ""
	if userInfo != nil {
		token = userInfo.Token
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.UserSessionCookieName,
		Value:    token,
		Expires:  expiresAt,
		Path:     "/",
		Domain:   h.cfg.UserSessionDomain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, returnURL, http.StatusFound)
}

// getMetadata returns session metadata from the request.
func getMetadata(r *http.Request) *SessionMetadata {
	return &SessionMetadata{
		UserAgent:  common.Ptr(r.UserAgent()),
		IPAddress:  common.Ptr(r.RemoteAddr),
		DeviceInfo: common.Ptr(r.Header.Get("Device-Info")),
	}
}

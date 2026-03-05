package github

import (
	"net/http"
	"time"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/auth"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

type HTTPHandler struct {
	cfg         *config.Config
	oauthConfig *oauth2.Config
	rdb         *redis.Client
	useCase     UseCase
}

func NewHTTPHandler(cfg *config.Config, oauthConfig *oauth2.Config, rdb *redis.Client, useCase UseCase) *HTTPHandler {
	return &HTTPHandler{
		cfg:         cfg,
		oauthConfig: oauthConfig,
		rdb:         rdb,
		useCase:     useCase,
	}
}

func (h *HTTPHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	state, err := common.TokenURLSafe(32)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	stateData := map[string]any{
		"provider": string(sqlc.OauthProviderGithub),
	}
	if err := common.StoreState(r.Context(), h.rdb, state, sqlc.OauthProviderGithub, stateData, h.cfg.OAuth2.OAuthStateTTL); err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	common.SetLoginCookie(w, r, h.cfg.OAuth2.OAuthStateCookieKey, state, h.cfg.OAuth2.OAuthStateTTL, h.cfg.Session.Secure)

	http.Redirect(w, r, h.oauthConfig.AuthCodeURL(state), http.StatusFound)
}

func (h *HTTPHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		respond.Error(w, apierror.BadRequest("missing state parameter"))
		return
	}

	stateData, err := common.RetrieveState(r.Context(), h.rdb, state, sqlc.OauthProviderGithub)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid or expired state"))
		return
	}

	if stateData["provider"] != string(sqlc.OauthProviderGithub) {
		respond.Error(w, apierror.BadRequest("provider mismatch"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respond.Error(w, apierror.BadRequest("missing code parameter"))
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	var expiresAt *int64
	if !token.Expiry.IsZero() {
		e := token.Expiry.Unix()
		expiresAt = &e
	}

	amd := &auth.SessionMetadata{
		UserAgent:  common.Ptr(r.UserAgent()),
		IPAddress:  common.Ptr(r.RemoteAddr),
		DeviceInfo: common.Ptr(r.Header.Get("Device-Info")),
	}

	userInfo, err := h.useCase.ProcessCallback(
		r.Context(), token.AccessToken, token.RefreshToken, expiresAt, amd)
	if err != nil || userInfo == nil {
		respond.Error(w, apierror.Internal())
		return
	}

	// Clean up state
	_ = common.DeleteState(r.Context(), h.rdb, state, sqlc.OauthProviderGithub)
	common.ClearLoginCookie(w, r, h.cfg.OAuth2.OAuthStateCookieKey, h.cfg.Session.Secure)

	h.HandleLoginResponse(w, r, userInfo, config.EmptyPath)
}

func (h *HTTPHandler) HandleLoginResponse(w http.ResponseWriter, r *http.Request, userInfo *auth.UserSessionInfo, returnTo string) {
	returnURL := h.cfg.GetReturnURL(returnTo)
	secure := h.cfg.IsProduction()

	var expiresAt time.Time
	if userInfo.Session != nil && userInfo.Session.ExpiresAt != nil {
		expiresAt = *userInfo.Session.ExpiresAt
	} else {
		expiresAt = time.Now().Add(h.cfg.UserSessionTTL)
		// If logging out or session missing, we want to clear the cookie.
		if userInfo.Token == "" {
			expiresAt = time.Now().Add(-24 * time.Hour)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.UserSessionCookieName,
		Value:    userInfo.Token,
		Expires:  expiresAt,
		Path:     "/",
		Domain:   h.cfg.UserSessionDomain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, returnURL, http.StatusFound)
}

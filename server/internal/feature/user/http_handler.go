package user

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/queryparam"
	"github.com/justblue/samsa/pkg/respond"
)

//go:generate mockgen -destination=mocks/mock_http_handler.go -source=http_handler.go -package=mocks

type HTTPHandler struct {
	u         UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func NewHTTPHandler(u UseCase, cfg *config.Config, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		u:         u,
		cfg:       cfg,
		validator: v,
	}
}

func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrOAuthAccountNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrCannotDisconnectLastAuthMethod) || errors.Is(err, ErrEmailTaken) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	if errors.Is(err, ErrBanned) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	respond.Error(w, apierror.Internal())
}

// Me returns the authenticated user's information.
//
//	@Summary		Get current user
//	@Description	Retrieves the authenticated user's information. Requires user actor.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserResponse
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/me [get]
func (h *HTTPHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	resp := ConvertUserResponse(sub.User)

	respond.OK(w, resp)
}

// MeScopes returns the scopes of the authenticated user.
//
//	@Summary		Get current user scopes
//	@Description	Retrieves the OAuth scopes granted to the authenticated user. Requires user actor.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserScopeResponse
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/me/scopes [get]
func (h *HTTPHandler) MeScopes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	respond.JSON(w, http.StatusOK, UserScopeResponse{Scopes: subj.Scopes})
}

type DisconnectProviderQueryParams struct {
	Provider sqlc.OAuthProvider `json:"provider" query:"provider" validate:"required,oneof=github google"`
}

// DisconnectProvider disconnects the authenticated user's OAuth account provider.
//
//	@Summary		Disconnect OAuth provider
//	@Description	Disconnects an OAuth provider (github or google) from the authenticated user's account. Requires user actor. Cannot disconnect the last authentication method.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			provider	path		sqlc.OAuthProvider	true	"OAuth provider (github or google)"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		409			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/me/oauth-accounts/{provider} [delete]
func (h *HTTPHandler) DisconnectProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var q DisconnectProviderQueryParams
	if err := queryparam.DecodeRequest(&q, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	err = h.u.DisconnectOAuthAccountProvider(ctx, sub.User, q.Provider)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "OAuth account provider disconnected successfully"})
}

// IsOnline checks if a user is currently online.
//
//	@Summary		Check user online status
//	@Description	Checks if a user is currently online. Requires user actor.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string	true	"User UUID"
//	@Success		200		{object}	map[string]bool
//	@Failure		400		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/{user_id}/online [get]
func (h *HTTPHandler) IsOnline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid user id"))
		return
	}

	isOnline, err := h.u.IsOnline(ctx, userID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]bool{"online": isOnline})
}

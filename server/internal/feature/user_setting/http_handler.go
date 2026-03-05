package user_setting

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
)

type HTTPHandler struct {
	u         UseCase
	validator *validator.Validate
}

func NewHTTPHandler(u UseCase, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		u:         u,
		validator: v,
	}
}

func (h *HTTPHandler) mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUserSettingNotFound):
		respond.Error(w, apierror.NotFound(err.Error()))
	default:
		respond.Error(w, apierror.Internal())
	}
}

// GetMeSettings retrieves the authenticated user's settings.
//
//	@Summary		Get current user settings
//	@Description	Retrieves the settings (preference, editor, notification) for the authenticated user. Requires user actor.
//	@Tags			user-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserSettingResponse
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/me/settings [get]
func (h *HTTPHandler) GetMeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	settings, err := h.u.Get(ctx, sub.User)
	if err != nil {
		h.mapError(w, err)
		return
	}

	resp, err := ConvertToUserSettingResponse(settings)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// UpdateMeSettings updates the authenticated user's settings.
//
//	@Summary		Update current user settings
//	@Description	Updates the settings (preference, editor, notification) for the authenticated user. Requires user actor.
//	@Tags			user-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UserSettingUpdate	true	"User settings update"
//	@Success		200		{object}	UserSettingResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/me/settings [patch]
func (h *HTTPHandler) UpdateMeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var update UserSettingUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respond.Error(w, apierror.BadRequest("invalid request body"))
		return
	}

	if err := h.validator.Struct(update); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	settings, err := h.u.Update(ctx, sub.User, update)
	if err != nil {
		h.mapError(w, err)
		return
	}

	resp, err := ConvertToUserSettingResponse(settings)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}
	respond.JSON(w, http.StatusOK, resp)
}

// ResetMeSettings resets the authenticated user's settings to defaults.
//
//	@Summary		Reset current user settings
//	@Description	Resets all user settings (preference, editor, notification) to their default values. Requires user actor.
//	@Tags			user-settings
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]any
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/me/settings/reset [delete]
func (h *HTTPHandler) ResetMeSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if err := h.u.Reset(ctx, sub.User); err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]any{"message": "User settings reset successfully"})
}

package spinnet

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/pkg/apierror"
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
	switch {
	case err == ErrSpinnetNotFound:
		respond.Error(w, apierror.NotFound("spinnet not found"))
	case err == ErrSpinnetExists:
		respond.Error(w, apierror.Conflict("spinnet with this smart syntax already exists"))
	case err == ErrPermissionDenied:
		respond.Error(w, apierror.Forbidden())
	case err == ErrInvalidSmartSyntax:
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
	default:
		respond.Error(w, apierror.Internal())
	}
}

// GetByID retrieves a spinnet by ID.
//
//	@Summary		Get spinnet by ID
//	@Description	Retrieves a spinnet by its UUID.
//	@Tags			spinnets
//	@Accept			json
//	@Produce		json
//	@Param			spinnet_id	path		string	true	"Spinnet UUID"
//	@Success		200			{object}	SpinnetResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Router			/spinnets/{spinnet_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "spinnet_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid spinnet id"))
		return
	}

	resp, err := h.u.GetByID(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetBySyntax retrieves a spinnet by smart syntax.
//
//	@Summary		Get spinnet by smart syntax
//	@Description	Retrieves a spinnet by its smart syntax (e.g., /welcome).
//	@Tags			spinnets
//	@Accept			json
//	@Produce		json
//	@Param			syntax	path		string	true	"Smart syntax (e.g., /welcome)"
//	@Success		200		{object}	SpinnetResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Router			/spinnets/syntax/{syntax} [get]
func (h *HTTPHandler) GetBySyntax(w http.ResponseWriter, r *http.Request) {
	syntax := chi.URLParam(r, "syntax")
	if syntax == "" {
		respond.Error(w, apierror.BadRequest("smart syntax is required"))
		return
	}

	resp, err := h.u.GetBySmartSyntax(r.Context(), syntax)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// List retrieves spinnets with pagination.
//
//	@Summary		List spinnets
//	@Description	Retrieves a paginated list of spinnets.
//	@Tags			spinnets
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Limit"
//	@Param			offset	query		int	false	"Offset"
//	@Success		200		{object}	SpinnetListResponse
//	@Router			/spinnets [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	var params ListSpinnetsParams
	if err := decodeListSpinnetsParams(r, &params); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.List(r.Context(), params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Create creates a new spinnet.
//
//	@Summary		Create spinnet
//	@Description	Creates a new spinnet template.
//	@Tags			spinnets
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateSpinnetRequest	true	"Spinnet creation request"
//	@Success		201		{object}	SpinnetResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		409		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Router			/spinnets [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateSpinnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	// Set owner_id from authenticated user if not provided
	if req.OwnerID == nil {
		req.OwnerID = &sub.User.ID
	}

	resp, err := h.u.Create(ctx, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// Update updates an existing spinnet.
//
//	@Summary		Update spinnet
//	@Description	Updates a spinnet template.
//	@Tags			spinnets
//	@Accept			json
//	@Produce		json
//	@Param			spinnet_id	path		string					true	"Spinnet UUID"
//	@Param			request		body		UpdateSpinnetRequest	true	"Spinnet update request"
//	@Success		200			{object}	SpinnetResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		422			{object}	apierror.APIError
//	@Router			/spinnets/{spinnet_id} [put]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "spinnet_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid spinnet id"))
		return
	}

	var req UpdateSpinnetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.Update(r.Context(), id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete deletes a spinnet.
//
//	@Summary		Delete spinnet
//	@Description	Deletes a spinnet template.
//	@Tags			spinnets
//	@Param			spinnet_id	path	string	true	"Spinnet UUID"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		404	{object}	apierror.APIError
//	@Router			/spinnets/{spinnet_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "spinnet_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid spinnet id"))
		return
	}

	err = h.u.Delete(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

func decodeListSpinnetsParams(r *http.Request, params *ListSpinnetsParams) error {
	query := r.URL.Query()

	if limitStr := query.Get("limit"); limitStr != "" {
		var limit int32
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			return err
		}
		params.Limit = limit
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		var offset int32
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			return err
		}
		params.Offset = offset
	}

	return nil
}

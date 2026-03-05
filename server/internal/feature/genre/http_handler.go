package genre

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
)

type HTTPHandler struct {
	uc        UseCase
	validator *validator.Validate
}

func NewHTTPHandler(uc UseCase, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		uc:        uc,
		validator: v,
	}
}

func (h *HTTPHandler) mapError(w http.ResponseWriter, err error) {
	// Add specific error mappings if needed
	respond.Error(w, apierror.Internal().WithMessage(err.Error()))
}

func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateGenreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	res, err := h.uc.CreateGenre(r.Context(), req)
	if err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, res)
}

func (h *HTTPHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid genre id"))
		return
	}

	res, err := h.uc.GetGenre(r.Context(), id)
	if err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListGenres(r.Context())
	if err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid genre id"))
		return
	}

	var req UpdateGenreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	res, err := h.uc.UpdateGenre(r.Context(), id, req)
	if err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid genre id"))
		return
	}

	if err := h.uc.DeleteGenre(r.Context(), id); err != nil {
		h.mapError(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *HTTPHandler) AddToStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var req struct {
		GenreID uuid.UUID `json:"genre_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.uc.AddGenreToStory(r.Context(), storyID, req.GenreID); err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, nil)
}

func (h *HTTPHandler) RemoveFromStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}
	genreID, err := uuid.Parse(chi.URLParam(r, "genreID"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid genre id"))
		return
	}

	if err := h.uc.RemoveGenreFromStory(r.Context(), storyID, genreID); err != nil {
		h.mapError(w, err)
		return
	}

	respond.NoContent(w)
}

func (h *HTTPHandler) GetStoryGenres(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	res, err := h.uc.GetStoryGenres(r.Context(), storyID)
	if err != nil {
		h.mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

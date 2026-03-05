package chapter

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
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
	if err == ErrPermissionDenied {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if err == ErrChapterNotFound {
		respond.Error(w, apierror.NotFound("chapter not found"))
		return
	}
	respond.Error(w, apierror.Internal())
}

// Create creates a new chapter.
//
//	@Summary		Create chapter
//	@Description	Creates a new chapter for a story. Requires `user actor`.
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateChapterRequest	true	"Chapter creation request"
//	@Success		201		{object}	ChapterResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/chapters [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateChapter(ctx, req.StoryID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetByID retrieves a chapter by ID.
//
//	@Summary		Get chapter by ID
//	@Description	Retrieves a chapter by its UUID.
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chapter_id	path		string	true	"Chapter UUID"
//	@Success		200			{object}	ChapterResponse
//	@Failure		404			{object}	apierror.APIError
//	@Router			/chapters/{chapter_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	resp, err := h.u.GetChapter(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.NotFound("chapter not found"))
		return
	}

	respond.OK(w, resp)
}

// List retrieves chapters for a story.
//
//	@Summary		List chapters
//	@Description	Retrieves chapters for a story.
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Param			story_id		query	string	true	"Story UUID"
//	@Param			is_published	query	boolean	false	"Filter by published status"
//	@Success		200				{array}	ChapterResponse
//	@Router			/chapters [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	var params ListChaptersParams
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListChapters(r.Context(), params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing chapter.
//
//	@Summary		Update chapter
//	@Description	Updates chapter details. Requires `user actor` and ownership.
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chapter_id	path		string					true	"Chapter UUID"
//	@Param			request		body		UpdateChapterRequest	true	"Chapter update request"
//	@Success		200			{object}	ChapterResponse
//	@Router			/chapters/{chapter_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	var req UpdateChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateChapter(ctx, sub.User.ID, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete soft deletes a chapter.
//
//	@Summary		Delete chapter
//	@Description	Soft deletes a chapter. Requires `user actor` and ownership.
//	@Tags			chapters
//	@Security		BearerAuth
//	@Param			chapter_id	path	string	true	"Chapter UUID"
//	@Success		204
//	@Router			/chapters/{chapter_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	err = h.u.DeleteChapter(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// Publish publishes a chapter.
//
//	@Summary		Publish chapter
//	@Description	Publishes a chapter. Requires `user actor` and ownership.
//	@Tags			chapters
//	@Security		BearerAuth
//	@Param			chapter_id	path		string	true	"Chapter UUID"
//	@Success		200			{object}	ChapterResponse
//	@Router			/chapters/{chapter_id}/publish [patch]
func (h *HTTPHandler) Publish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	resp, err := h.u.PublishChapter(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Unpublish unpublishes a chapter.
//
//	@Summary		Unpublish chapter
//	@Description	Unpublishes a chapter. Requires `user actor` and ownership.
//	@Tags			chapters
//	@Security		BearerAuth
//	@Param			chapter_id	path		string	true	"Chapter UUID"
//	@Success		200			{object}	ChapterResponse
//	@Router			/chapters/{chapter_id}/unpublish [patch]
func (h *HTTPHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	resp, err := h.u.UnpublishChapter(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Reorder reorders a chapter.
//
//	@Summary		Reorder chapter
//	@Description	Reorders a chapter within a story. Requires `user actor` and ownership.
//	@Tags			chapters
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chapter_id	path		string					true	"Chapter UUID"
//	@Param			request		body		ReorderChapterRequest	true	"Reorder request"
//	@Success		200			{object}	ChapterResponse
//	@Router			/chapters/{chapter_id}/reorder [patch]
func (h *HTTPHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	var req ReorderChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ReorderChapter(ctx, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// IncrementView increments the view count for a chapter.
//
//	@Summary		Increment chapter views
//	@Description	Increments the view count for a chapter.
//	@Tags			chapters
//	@Produce		json
//	@Param			chapter_id	path		string	true	"Chapter UUID"
//	@Success		200			{object}	ChapterResponse
//	@Router			/chapters/{chapter_id}/view [post]
func (h *HTTPHandler) IncrementView(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "chapter_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid chapter id"))
		return
	}

	resp, err := h.u.IncrementView(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

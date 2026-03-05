package story

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
	respond.Error(w, apierror.Internal())
}

// CreateStory creates a new story.
//
//	@Summary		Create story
//	@Description	Creates a new story. Requires `user actor`.
//	@Tags			stories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateStoryRequest	true	"Story creation request"
//	@Success		201		{object}	StoryResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/stories [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateStory(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetByID retrieves a story by ID.
//
//	@Summary		Get story by ID
//	@Description	Retrieves a story by its UUID.
//	@Tags			stories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	StoryResponse
//	@Failure		404			{object}	apierror.APIError
//	@Router			/stories/{story_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.GetStory(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.NotFound("story not found"))
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing story.
//
//	@Summary		Update story
//	@Description	Updates story details. Requires `user actor` and ownership.
//	@Tags			stories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string				true	"Story UUID"
//	@Param			request		body		UpdateStoryRequest	true	"Story update request"
//	@Success		200			{object}	StoryResponse
//	@Router			/stories/{story_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var req UpdateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateStory(ctx, sub.User.ID, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete soft deletes a story.
//
//	@Summary		Delete story
//	@Description	Soft deletes a story. Requires `user actor` and ownership.
//	@Tags			stories
//	@Security		BearerAuth
//	@Param			story_id	path	string	true	"Story UUID"
//	@Success		204
//	@Router			/stories/{story_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	err = h.u.DeleteStory(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// List retrieves a list of stories for the current user.
//
//	@Summary		List current user stories
//	@Description	Retrieves a list of stories owned by the current user. Requires `user actor`.
//	@Tags			stories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query	int	false	"Limit"
//	@Param			offset	query	int	false	"Offset"
//	@Success		200		{array}	StoryResponse
//	@Router			/stories [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var params ListStoriesParams
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListUserStories(ctx, sub.User.ID, params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Publish publishes a story.
//
//	@Summary		Publish story
//	@Description	Changes story status to `published`. Requires `user actor` and ownership.
//	@Tags			stories
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	StoryResponse
//	@Router			/stories/{story_id}/publish [patch]
func (h *HTTPHandler) Publish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.PublishStory(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Archive archives a story.
//
//	@Summary		Archive story
//	@Description	Changes story status to `archived`. Requires `user actor` and ownership.
//	@Tags			stories
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	StoryResponse
//	@Router			/stories/{story_id}/archive [patch]
func (h *HTTPHandler) Archive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.ArchiveStory(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Vote votes on a story.
//
//	@Summary		Vote on story
//	@Description	Casts or updates a vote on a story. Requires `user actor`.
//	@Tags			stories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string				true	"Story UUID"
//	@Param			request		body		map[string]int32	true	"Rating (1-5)"
//	@Success		200			{object}	sqlc.StoryVote
//	@Router			/stories/{story_id}/vote [post]
func (h *HTTPHandler) Vote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var req struct {
		Rating int32 `json:"rating" validate:"required,min=1,max=5"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.VoteStory(ctx, sub.User.ID, id, req.Rating)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// DeleteVote removes a vote from a story.
//
//	@Summary		Remove vote
//	@Description	Removes the current user's vote from a story. Requires `user actor`.
//	@Tags			stories
//	@Security		BearerAuth
//	@Param			story_id	path	string	true	"Story UUID"
//	@Success		204
//	@Router			/stories/{story_id}/vote [delete]
func (h *HTTPHandler) DeleteVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	err = h.u.RemoveVote(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// GetVoteStats retrieves vote statistics for a story.
//
//	@Summary		Get story vote stats
//	@Description	Retrieves count and average rating for a story.
//	@Tags			stories
//	@Produce		json
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	sqlc.GetStoryVoteStatsRow
//	@Router			/stories/{story_id}/vote/stats [get]
func (h *HTTPHandler) GetVoteStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.GetVoteStats(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

package story_vote

import (
	"encoding/json"
	"errors"
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
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrUnauthorized) {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	if errors.Is(err, ErrForbidden) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, ErrInvalidRating) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

// CreateVote creates or updates a vote on a story.
//
//	@Summary		Create or update vote
//	@Description	Creates a new vote or updates an existing vote for a story. Requires `user` actor and `story.vote:write` scope.
//	@Tags			story-votes
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateVoteRequest	true	"Vote creation request"
//	@Success		201		{object}	VoteResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/story-votes [post]
func (h *HTTPHandler) CreateVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateVote(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetVote retrieves a vote by ID.
//
//	@Summary		Get vote by ID
//	@Description	Retrieves a vote by its UUID. Requires `user` actor and `story.vote:read` scope.
//	@Tags			story-votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			vote_id	path		string	true	"Vote UUID"
//	@Success		200		{object}	VoteResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/story-votes/{vote_id} [get]
func (h *HTTPHandler) GetVote(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "vote_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid vote id"))
		return
	}

	resp, err := h.u.GetVote(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetUserVote retrieves the current user's vote for a story.
//
//	@Summary		Get user's vote
//	@Description	Retrieves the authenticated user's vote for a specific story. Requires `user` actor and `story.vote:read` scope.
//	@Tags			story-votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	VoteResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/my-vote [get]
func (h *HTTPHandler) GetUserVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.GetUserVote(ctx, storyID, sub.User.ID)
	if err != nil {
		mapError(w, err)
		return
	}

	if resp == nil {
		respond.Error(w, apierror.NotFound("no vote found for this story"))
		return
	}

	respond.OK(w, resp)
}

// DeleteUserVote deletes the current user's vote for a story.
//
//	@Summary		Delete user's vote
//	@Description	Deletes the authenticated user's vote for a specific story. Requires `user` actor and `story.vote:write` scope.
//	@Tags			story-votes
//	@Security		BearerAuth
//	@Param			story_id	path	string	true	"Story UUID"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/stories/{story_id}/vote [delete]
func (h *HTTPHandler) DeleteUserVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	err = h.u.DeleteUserVote(ctx, storyID, sub.User.ID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// GetVoteStats retrieves vote statistics for a story.
//
//	@Summary		Get vote statistics
//	@Description	Retrieves vote count and average rating for a story. Public endpoint.
//	@Tags			story-votes
//	@Produce		json
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	VoteStatsResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/vote-stats [get]
func (h *HTTPHandler) GetVoteStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	resp, err := h.u.GetVoteStats(r.Context(), storyID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// ListStoryVotes lists votes for a story.
//
//	@Summary		List story votes
//	@Description	Retrieves a paginated list of votes for a story. Requires `user` actor and `story.vote:read` scope.
//	@Tags			story-votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Param			sort_field	query		string	false	"Sort field (rating, created_at)"
//	@Param			sort_order	query		string	false	"Sort order (asc, desc)"
//	@Success		200			{object}	VoteListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/votes [get]
func (h *HTTPHandler) ListStoryVotes(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var filter VoteFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	votes, total, err := h.u.ListStoryVotes(r.Context(), storyID, &filter)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := VoteListResponse{
		Votes: votes,
		Meta:  queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}

	respond.OK(w, resp)
}

// ListUserVotes lists votes by a user.
//
//	@Summary		List user votes
//	@Description	Retrieves a paginated list of votes cast by a user. Requires `user` actor and `story.vote:read` scope.
//	@Tags			story-votes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		string	true	"User UUID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Param			sort_field	query		string	false	"Sort field (rating, created_at)"
//	@Param			sort_order	query		string	false	"Sort order (asc, desc)"
//	@Success		200			{object}	VoteListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/users/{user_id}/votes [get]
func (h *HTTPHandler) ListUserVotes(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "user_id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid user id"))
		return
	}

	var filter VoteFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	votes, total, err := h.u.ListUserVotes(r.Context(), userID, &filter)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := VoteListResponse{
		Votes: votes,
		Meta:  queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}

	respond.OK(w, resp)
}

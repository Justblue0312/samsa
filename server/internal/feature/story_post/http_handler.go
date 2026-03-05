package story_post

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
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
	respond.Error(w, apierror.Internal())
}

// CreatePost creates a new story post.
//
//	@Summary		Create story post
//	@Description	Creates a new post (announcement/update) for a story or creator.
//	@Tags			story-posts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateStoryPostRequest	true	"Post creation request"
//	@Success		201		{object}	StoryPostResponse
//	@Router			/story-posts [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateStoryPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreatePost(r.Context(), req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetByID retrieves a post by ID.
//
//	@Summary		Get post by ID
//	@Description	Retrieves a story post by its UUID.
//	@Tags			story-posts
//	@Accept			json
//	@Produce		json
//	@Param			post_id	path		string	true	"Post UUID"
//	@Success		200		{object}	StoryPostResponse
//	@Router			/story-posts/{post_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "post_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid post id"))
		return
	}

	resp, err := h.u.GetPost(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.NotFound("post not found"))
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing post.
//
//	@Summary		Update post
//	@Description	Updates post content or notify status.
//	@Tags			story-posts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		string					true	"Post UUID"
//	@Param			request	body		UpdateStoryPostRequest	true	"Post update request"
//	@Success		200		{object}	StoryPostResponse
//	@Router			/story-posts/{post_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "post_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid post id"))
		return
	}

	var req UpdateStoryPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdatePost(r.Context(), id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete deletes a post.
//
//	@Summary		Delete post
//	@Description	Deletes a story post.
//	@Tags			story-posts
//	@Security		BearerAuth
//	@Param			post_id	path	string	true	"Post UUID"
//	@Success		204
//	@Router			/story-posts/{post_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "post_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid post id"))
		return
	}

	err = h.u.DeletePost(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// ListByAuthor retrieves posts by author.
//
//	@Summary		List author posts
//	@Description	Retrieves all posts by a specific author.
//	@Tags			story-posts
//	@Produce		json
//	@Param			author_id	path	string	true	"Author UUID"
//	@Param			limit		query	int		false	"Limit"
//	@Param			offset		query	int		false	"Offset"
//	@Success		200			{array}	StoryPostResponse
//	@Router			/authors/{author_id}/posts [get]
func (h *HTTPHandler) ListByAuthor(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "author_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid author id"))
		return
	}

	var params struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListAuthorPosts(r.Context(), id, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// ListByStory retrieves posts by story.
//
//	@Summary		List story posts
//	@Description	Retrieves all posts linked to a specific story.
//	@Tags			story-posts
//	@Produce		json
//	@Param			story_id	path	string	true	"Story UUID"
//	@Param			limit		query	int		false	"Limit"
//	@Param			offset		query	int		false	"Offset"
//	@Success		200			{array}	StoryPostResponse
//	@Router			/stories/{story_id}/posts [get]
func (h *HTTPHandler) ListByStory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var params struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListStoryPosts(r.Context(), id, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// ListByStoryFiltered retrieves posts by story with filters.
//
//	@Summary		List story posts with filters
//	@Description	Retrieves posts linked to a story with optional filters.
//	@Tags			story-posts
//	@Produce		json
//	@Param			story_id		path		string	true	"Story UUID"
//	@Param			author_id		query		string	false	"Filter by author UUID"
//	@Param			include_deleted	query		bool	false	"Include deleted posts"
//	@Param			page			query		int		false	"Page number"
//	@Param			limit			query		int		false	"Items per page"
//	@Success		200				{object}	StoryPostListResponse
//	@Router			/stories/{story_id}/posts/filter [get]
func (h *HTTPHandler) ListByStoryFiltered(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	var filter struct {
		AuthorID       *uuid.UUID `query:"author_id"`
		IncludeDeleted bool       `query:"include_deleted"`
		Page           int32      `query:"page"`
		Limit          int32      `query:"limit"`
	}
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	offset := (filter.Page - 1) * filter.Limit
	posts, total, err := h.u.ListStoryPostsFiltered(r.Context(), &storyID, filter.AuthorID, filter.IncludeDeleted, filter.Limit, offset)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := StoryPostListResponse{
		Posts: posts,
		Meta:  queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}
	respond.OK(w, resp)
}

// Restore restores a deleted post.
//
//	@Summary		Restore post
//	@Description	Restores a soft-deleted story post.
//	@Tags			story-posts
//	@Security		BearerAuth
//	@Param			post_id	path		string	true	"Post UUID"
//	@Success		200		{object}	StoryPostResponse
//	@Router			/story-posts/{post_id}/restore [post]
func (h *HTTPHandler) Restore(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "post_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid post id"))
		return
	}

	resp, err := h.u.RestorePost(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// PermanentlyDelete permanently deletes a post.
//
//	@Summary		Permanently delete post
//	@Description	Permanently deletes a story post (admin only).
//	@Tags			story-posts
//	@Security		BearerAuth
//	@Param			post_id	path	string	true	"Post UUID"
//	@Success		204
//	@Router			/story-posts/{post_id}/permanent [delete]
func (h *HTTPHandler) PermanentlyDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "post_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid post id"))
		return
	}

	err = h.u.PermanentlyDeletePost(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// BulkDelete deletes multiple posts.
//
//	@Summary		Bulk delete posts
//	@Description	Deletes multiple story posts by IDs.
//	@Tags			story-posts
//	@Security		BearerAuth
//	@Param			request	body	BulkDeleteRequest	true	"Post IDs to delete"
//	@Success		204
//	@Router			/story-posts/bulk-delete [post]
func (h *HTTPHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	err := h.u.BulkDeletePosts(r.Context(), req.IDs)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// GetByIDs retrieves multiple posts by IDs.
//
//	@Summary		Get posts by IDs
//	@Description	Retrieves multiple story posts by their UUIDs.
//	@Tags			story-posts
//	@Param			ids	query	string	true	"Comma-separated post UUIDs"
//	@Success		200	{array}	StoryPostResponse
//	@Router			/story-posts/batch [get]
func (h *HTTPHandler) GetByIDs(w http.ResponseWriter, r *http.Request) {
	idsStr := r.URL.Query().Get("ids")
	if idsStr == "" {
		respond.Error(w, apierror.BadRequest("missing ids parameter"))
		return
	}

	idStrs := strings.Split(idsStr, ",")
	ids := make([]uuid.UUID, len(idStrs))
	for i, s := range idStrs {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			respond.Error(w, apierror.BadRequest("invalid uuid: "+s))
			return
		}
		ids[i] = id
	}

	posts, err := h.u.GetPostsByIDs(r.Context(), ids)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, posts)
}

// CountStoryPosts returns the count of posts for a story.
//
//	@Summary		Count story posts
//	@Description	Returns the total count of posts for a story.
//	@Tags			story-posts
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	map[string]int64
//	@Router			/stories/{story_id}/posts/count [get]
func (h *HTTPHandler) CountStoryPosts(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	count, err := h.u.CountStoryPosts(r.Context(), storyID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, map[string]int64{"count": count})
}

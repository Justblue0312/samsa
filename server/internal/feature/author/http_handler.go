package author

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

type HTTPHandler struct {
	u         UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func NewHandler(cfg *config.Config, v *validator.Validate, u UseCase) *HTTPHandler {
	return &HTTPHandler{u: u, cfg: cfg, validator: v}
}

// GetMyAuthor retrieves the author's profile for the currently authenticated user.
//
//	@Summary		Get current author's profile
//	@Description	Retrieves the author profile for the authenticated user.
//	@Description	Requires a valid JWT token with user actor and `author:read` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	AuthorReposonse
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/authors/me [get]
func (h *HTTPHandler) GetMyAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	author, err := h.u.GetByUserID(ctx, subj.User.ID)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

// ListAuthors retrieves a paginated list of authors.
//
//	@Summary		List authors
//	@Description	Retrieves a list of authors with pagination and optional filters.
//	@Description	Supports anonymous and authenticated users. Required scopes: `author:read`.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int			false	"Page number (default: 1)"
//	@Param			limit			query		int			false	"Items per page (default: 20, max: 100)"
//	@Param			order_by		query		string		false	"Sort field (created_at, updated_at, stage_name)"	example(created_at:desc)
//	@Param			user_id			query		uuid.UUID	false	"Filter by user ID"									example(5e11bb83-a033-4adf-a5a2-18e400c56672)
//	@Param			is_recommended	query		bool		false	"Filter by recommended status"						example(true)
//	@Param			search_query	query		string		false	"Search by stage name"								example(foo)
//	@Success		200				{object}	AuthorResponses
//	@Failure		400				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/authors [get]
func (h *HTTPHandler) ListAuthors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	authors, total, err := h.u.List(ctx, filter)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorResponses{
		Authors: *authors,
		Meta:    queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}
	respond.OK(w, resp)
}

// GetAuthor retrieves an author by their ID.
//
//	@Summary		Get author by ID
//	@Description	Retrieves an author profile by their UUID.
//	@Description	Requires user actor and `author:read` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			author_id	path		string	true	"Author UUID"
//	@Success		200			{object}	AuthorReposonse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/authors/{author_id} [get]
func (h *HTTPHandler) GetAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "author_id")
	if idStr == "" {
		respond.Error(w, apierror.BadRequest("author_id is required"))
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid author_id"))
		return
	}

	author, err := h.u.GetByID(ctx, id)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

// GetAuthorBySlug retrieves an author by their unique slug.
//
//	@Summary		Get author by slug
//	@Description	Retrieves an author profile by their unique slug.
//	@Description	Requires user actor and `author:read` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			slug	path		string	true	"Author slug"
//	@Success		200		{object}	AuthorReposonse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/authors/slug/{slug} [get]
func (h *HTTPHandler) GetAuthorBySlug(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		respond.Error(w, apierror.BadRequest("slug is required"))
		return
	}

	author, err := h.u.GetBySlug(ctx, slug)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

// CreateAuthor creates a new author profile.
//
//	@Summary		Create author
//	@Description	Creates a new author profile for the authenticated user.
//	@Description	Requires user actor and `author:write` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateAuthorRequest	true	"Author creation request"
//	@Success		201		{object}	AuthorReposonse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/authors [post]
func (h *HTTPHandler) CreateAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateAuthorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("invalid request body"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	author, err := h.u.Create(ctx, subj.User, &req)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

// UpdateAuthor updates an existing author profile.
//
//	@Summary		Update author
//	@Description	Updates an author profile by their UUID.
//	@Description	Requires user actor and `author:write` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			author_id	path		string				true	"Author UUID"
//	@Param			request		body		UpdateAuthorRequest	true	"Author update request"
//	@Success		200			{object}	AuthorReposonse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/authors/{author_id} [patch]
func (h *HTTPHandler) UpdateAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "author_id")
	if idStr == "" {
		respond.Error(w, apierror.BadRequest("author_id is required"))
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid author_id"))
		return
	}

	var req UpdateAuthorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("invalid request body"))
		return
	}

	author, err := h.u.Update(ctx, subj.User, id, &req)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

// DeleteAuthor soft deletes an author profile.
//
//	@Summary		Delete author
//	@Description	Soft deletes an author profile by their UUID.
//	@Description	Requires moderator actor and `author:write` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			author_id	path	string	true	"Author UUID"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/authors/{author_id} [delete]
func (h *HTTPHandler) DeleteAuthor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "author_id")
	if idStr == "" {
		respond.Error(w, apierror.BadRequest("author_id is required"))
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid author_id"))
		return
	}

	if err := h.u.Delete(ctx, subj.User, id); err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusNoContent, nil)
}

// SetRecommended sets the recommended status for an author.
//
//	@Summary		Set author recommended status
//	@Description	Sets the recommended status for an author.
//	@Description	Requires moderator actor and `author:read` scope.
//	@Tags			authors
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			author_id	path		string					true	"Author UUID"
//	@Param			request		body		SetRecommendedRequest	true	"Recommended status request"
//	@Success		200			{object}	AuthorReposonse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/authors/{author_id}/recommend [patch]
func (h *HTTPHandler) SetRecommended(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subj, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	authorIdStr := chi.URLParam(r, "author_id")
	if authorIdStr == "" {
		respond.Error(w, apierror.BadRequest("author_id is required"))
		return
	}

	authorId, err := uuid.Parse(authorIdStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid author_id"))
		return
	}

	var req SetRecommendedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("invalid request body"))
		return
	}

	author, err := h.u.SetRecommended(ctx, subj.User, authorId, req.IsRecommended)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	resp := AuthorReposonse{Author: *author}
	respond.OK(w, resp)
}

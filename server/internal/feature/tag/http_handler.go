package tag

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrUnauthorized) {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotOwner) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, ErrAlreadyExists) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

// CreateTag creates a new tag.
//
//	@Summary		Create tag
//	@Description	Creates a new tag. Requires `user` actor and `tag:write` scope.
//	@Tags			tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateTagRequest	true	"Tag creation request"
//	@Success		201		{object}	TagResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		409		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/tags [post]
func (h *HTTPHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateTag(ctx, sub.User.ID, &req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetTag retrieves a tag by ID.
//
//	@Summary		Get tag by ID
//	@Description	Retrieves a tag by its UUID. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tag_id		path		string	true	"Tag UUID"
//	@Param			entity_type	query		string	false	"Entity type filter"
//	@Success		200			{object}	TagResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/tags/{tag_id} [get]
func (h *HTTPHandler) GetTag(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "tag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid tag id"))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	resp, err := h.u.GetTag(r.Context(), id, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// UpdateTag updates a tag.
//
//	@Summary		Update tag
//	@Description	Updates a tag. Only the tag owner can update. Requires `user` actor and `tag:write` scope.
//	@Tags			tags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tag_id	path		string				true	"Tag UUID"
//	@Param			request	body		UpdateTagRequest	true	"Tag update request"
//	@Success		200		{object}	TagResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		403		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/tags/{tag_id} [patch]
func (h *HTTPHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "tag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid tag id"))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateTag(ctx, id, sub.User.ID, &req, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// DeleteTag deletes a tag.
//
//	@Summary		Delete tag
//	@Description	Deletes a tag. Only the tag owner can delete. Requires `user` actor and `tag:write` scope.
//	@Tags			tags
//	@Security		BearerAuth
//	@Param			tag_id		path	string	true	"Tag UUID"
//	@Param			entity_type	query	string	false	"Entity type filter"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		403	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/tags/{tag_id} [delete]
func (h *HTTPHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "tag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid tag id"))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	err = h.u.DeleteTag(ctx, id, sub.User.ID, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// ListTags lists tags with filters.
//
//	@Summary		List tags
//	@Description	Retrieves a paginated list of tags with optional filters. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page number"
//	@Param			limit			query		int		false	"Items per page"
//	@Param			owner_id		query		string	false	"Filter by owner UUID"
//	@Param			entity_id		query		string	false	"Filter by entity UUID"
//	@Param			entity_type		query		string	false	"Filter by entity type"
//	@Param			is_hidden		query		bool	false	"Filter by hidden status"
//	@Param			is_system		query		bool	false	"Filter by system tags"
//	@Param			is_recommended	query		bool	false	"Filter by recommended tags"
//	@Success		200				{object}	TagListResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/tags [get]
func (h *HTTPHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	tags, total, err := h.u.ListTags(ctx, filter, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := TagListResponse{
		Tags:  tags,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}

	respond.OK(w, resp)
}

// SearchTags searches for tags.
//
//	@Summary		Search tags
//	@Description	Searches for tags by name. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q			query		string	false	"Search query"
//	@Param			entity_type	query		string	false	"Entity type filter"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	TagListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/tags/search [get]
func (h *HTTPHandler) SearchTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query().Get("q")
	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	var params struct {
		Page  int32 `json:"page"`
		Limit int32 `json:"limit"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}

	tags, total, err := h.u.SearchTags(ctx, entityType, &query, nil, nil, nil, params.Limit, int32(params.Page)*params.Limit)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := TagListResponse{
		Tags:  tags,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
	}

	respond.OK(w, resp)
}

// GetTagsByEntity retrieves tags for an entity.
//
//	@Summary		Get entity tags
//	@Description	Retrieves all tags for a specific entity. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity_type	path		string	true	"Entity type (story, chapter, comment, submission)"
//	@Param			entity_id	path		string	true	"Entity UUID"
//	@Success		200			{array}		TagResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/entities/{entity_type}/{entity_id}/tags [get]
func (h *HTTPHandler) GetTagsByEntity(w http.ResponseWriter, r *http.Request) {
	entityTypeStr := chi.URLParam(r, "entity_type")
	entityIDStr := chi.URLParam(r, "entity_id")

	entityType := sqlc.EntityType(entityTypeStr)
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid entity id"))
		return
	}

	tags, err := h.u.GetTagsByEntity(r.Context(), entityID, entityType, nil, nil, nil)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, tags)
}

// GetTagsByOwner retrieves tags by owner.
//
//	@Summary		Get owner tags
//	@Description	Retrieves all tags owned by a specific user. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner_id	path		string	true	"Owner UUID"
//	@Param			entity_type	query		string	false	"Entity type filter"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	TagListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/owners/{owner_id}/tags [get]
func (h *HTTPHandler) GetTagsByOwner(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := chi.URLParam(r, "owner_id")
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid owner id"))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	var params struct {
		Page  int32 `json:"page"`
		Limit int32 `json:"limit"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}

	tags, total, err := h.u.GetTagsByOwner(r.Context(), ownerID, entityType, nil, nil, nil, params.Limit, int32(params.Page)*params.Limit)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := TagListResponse{
		Tags:  tags,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
	}

	respond.OK(w, resp)
}

// GetTagsByIDs retrieves tags by IDs.
//
//	@Summary		Get tags by IDs
//	@Description	Retrieves tags by their UUIDs. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			ids			query		string	true	"Comma-separated tag UUIDs"
//	@Param			entity_type	query		string	false	"Entity type filter"
//	@Success		200			{array}		TagResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/tags/batch [get]
func (h *HTTPHandler) GetTagsByIDs(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		respond.Error(w, apierror.BadRequest("ids parameter is required"))
		return
	}

	entityType := sqlc.EntityType(r.URL.Query().Get("entity_type"))
	if entityType == "" {
		entityType = sqlc.EntityTypeStory
	}

	idStrs := strings.Split(idsParam, ",")
	tagIDs := make([]uuid.UUID, 0, len(idStrs))
	for _, idStr := range idStrs {
		id, err := uuid.Parse(strings.TrimSpace(idStr))
		if err != nil {
			respond.Error(w, apierror.BadRequest("invalid tag id: "+idStr))
			return
		}
		tagIDs = append(tagIDs, id)
	}

	tags, err := h.u.GetTagsByIDs(r.Context(), tagIDs, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, tags)
}

// GetTagCount gets the count of tags for an entity.
//
//	@Summary		Get tag count
//	@Description	Retrieves the count of tags for a specific entity. Requires `user` actor and `tag:read` scope.
//	@Tags			tags
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity_type	path		string	true	"Entity type"
//	@Param			entity_id	path		string	true	"Entity UUID"
//	@Success		200			{object}	map[string]int
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/entities/{entity_type}/{entity_id}/tags/count [get]
func (h *HTTPHandler) GetTagCount(w http.ResponseWriter, r *http.Request) {
	entityTypeStr := chi.URLParam(r, "entity_type")
	entityIDStr := chi.URLParam(r, "entity_id")

	entityType := sqlc.EntityType(entityTypeStr)
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid entity id"))
		return
	}

	count, err := h.u.CountTagsByEntity(r.Context(), entityID, entityType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, map[string]int64{"count": count})
}

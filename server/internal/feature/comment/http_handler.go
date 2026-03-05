package comment

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
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrParentNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrNotOwner) || errors.Is(err, ErrNotModerator) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, ErrDepthExceeded) || errors.Is(err, ErrEntityMismatch) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	if errors.Is(err, ErrAlreadyDeleted) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

type CreateCommentRequestPayload struct {
	EntityType string     `json:"entity_type" validate:"required"`
	EntityID   uuid.UUID  `json:"entity_id" validate:"required"`
	ParentID   *uuid.UUID `json:"parent_id"`
	Content    string     `json:"content" validate:"required"`
	Source     string     `json:"source"`
}

type UpdateCommentRequestPayload struct {
	Content string `json:"content" validate:"required"`
	Source  string `json:"source"`
}

func (h *HTTPHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var payload CreateCommentRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	req := &CreateCommentRequest{
		EntityType: payload.EntityType,
		EntityID:   payload.EntityID,
		ParentID:   payload.ParentID,
		Content:    payload.Content,
		Source:     payload.Source,
	}

	result, err := h.u.Create(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var filter CommentFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&filter); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.List(ctx, &filter)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) GetComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "comment_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	sub, _ := common.GetUserSubject(ctx)
	includeDeleted := sub != nil && sub.IsModerator()

	result, err := h.u.GetByID(ctx, id, entityType, includeDeleted)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "comment_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	var payload UpdateCommentRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	req := &UpdateCommentRequest{
		Content: payload.Content,
		Source:  payload.Source,
	}

	result, err := h.u.Update(ctx, id, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "comment_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	isModerator := sub.IsModerator()

	err = h.u.Delete(ctx, id, sub.User.ID, isModerator)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "comment deleted successfully"})
}

func (h *HTTPHandler) Moderate(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sub, err := common.GetUserSubject(ctx)
		if err != nil {
			respond.Error(w, apierror.Unauthorized())
			return
		}

		idStr := chi.URLParam(r, "comment_id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			respond.Error(w, apierror.BadRequest("invalid comment id"))
			return
		}

		var modAction ModerationAction
		switch action {
		case "report":
			modAction = ModerationActionReport
		case "restore":
			modAction = ModerationActionRestore
		case "pin":
			modAction = ModerationActionPin
		case "unpin":
			modAction = ModerationActionUnpin
		case "resolve":
			modAction = ModerationActionResolve
		case "archive":
			modAction = ModerationActionArchive
		default:
			respond.Error(w, apierror.BadRequest("invalid action"))
			return
		}

		result, err := h.u.Moderate(ctx, id, modAction, sub.User.ID, sub.IsModerator())
		if err != nil {
			mapError(w, err)
			return
		}

		respond.OK(w, result)
	}
}

var _ sqlc.VoteType
var _ sqlc.ReactionType

type BulkCommentRequest struct {
	Ids []uuid.UUID `json:"ids" validate:"required,min=1"`
}

// BulkDeleteComments deletes multiple comments by ID.
//
//	@Summary		Bulk delete comments
//	@Description	Bulk delete comments (moderator only or own comments). Requires `comment:moderate` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkCommentRequest	true	"Comment IDs to delete"
//	@Success		200		{object}	[]CommentResponse
//	@Router			/comments/bulk/delete [post]
func (h *HTTPHandler) BulkDeleteComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req BulkCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.BulkDelete(ctx, req.Ids, "", sub.User.ID, sub.IsModerator())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// BulkArchiveComments archives multiple comments by ID.
//
//	@Summary		Bulk archive comments
//	@Description	Bulk archive comments (moderator only). Requires `comment:moderate` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkCommentRequest	true	"Comment IDs to archive"
//	@Success		200		{object}	[]CommentResponse
//	@Router			/comments/bulk/archive [post]
func (h *HTTPHandler) BulkArchiveComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req BulkCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.BulkArchive(ctx, req.Ids, "", sub.IsModerator())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// BulkResolveComments resolves multiple comments by ID.
//
//	@Summary		Bulk resolve comments
//	@Description	Bulk resolve comments (moderator only). Requires `comment:moderate` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkCommentRequest	true	"Comment IDs to resolve"
//	@Success		200		{object}	[]CommentResponse
//	@Router			/comments/bulk/resolve [post]
func (h *HTTPHandler) BulkResolveComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req BulkCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.BulkResolve(ctx, req.Ids, "", sub.IsModerator())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// BulkPinComments pins multiple comments by ID.
//
//	@Summary		Bulk pin comments
//	@Description	Bulk pin comments (moderator only). Requires `comment:moderate` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkCommentRequest	true	"Comment IDs to pin"
//	@Success		200		{object}	[]CommentResponse
//	@Router			/comments/bulk/pin [post]
func (h *HTTPHandler) BulkPinComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req BulkCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.BulkPin(ctx, req.Ids, "", sub.User.ID, sub.IsModerator())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// BulkUnpinComments unpins multiple comments by ID.
//
//	@Summary		Bulk unpin comments
//	@Description	Bulk unpin comments (moderator only). Requires `comment:moderate` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkCommentRequest	true	"Comment IDs to unpin"
//	@Success		200		{object}	[]CommentResponse
//	@Router			/comments/bulk/unpin [post]
func (h *HTTPHandler) BulkUnpinComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req BulkCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.BulkUnpin(ctx, req.Ids, "", sub.IsModerator())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// SearchComments searches comments by content.
//
//	@Summary		Search comments
//	@Description	Search comments by content within an entity. Requires `comment:read` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity_type	query		string	true	"Entity type (story, author, etc.)"
//	@Param			entity_id	query		string	true	"Entity UUID"
//	@Param			search		query		string	true	"Search query"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	CommentListResponse
//	@Router			/comments/search [get]
func (h *HTTPHandler) SearchComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	entityIDStr := r.URL.Query().Get("entity_id")
	if entityIDStr == "" {
		respond.Error(w, apierror.BadRequest("entity_id is required"))
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid entity_id"))
		return
	}

	search := r.URL.Query().Get("search")
	if search == "" {
		respond.Error(w, apierror.BadRequest("search query is required"))
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

	result, err := h.u.Search(ctx, entityType, entityID, search, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// ListCommentsWithFilters lists comments with advanced filters.
//
//	@Summary		List comments with filters
//	@Description	List comments with advanced filtering options. Requires `comment:read` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			entity_type	query		string	true	"Entity type"
//	@Param			entity_id	query		string	true	"Entity UUID"
//	@Param			is_deleted	query		bool	false	"Filter by deleted status"
//	@Param			is_resolved	query		bool	false	"Filter by resolved status"
//	@Param			is_archived	query		bool	false	"Filter by archived status"
//	@Param			is_reported	query		bool	false	"Filter by reported status"
//	@Param			is_pinned	query		bool	false	"Filter by pinned status"
//	@Param			parent_id	query		string	false	"Filter by parent comment ID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	CommentListResponse
//	@Router			/comments/filter [get]
func (h *HTTPHandler) ListCommentsWithFilters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	entityIDStr := r.URL.Query().Get("entity_id")
	if entityIDStr == "" {
		respond.Error(w, apierror.BadRequest("entity_id is required"))
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid entity_id"))
		return
	}

	var isDeleted, isResolved, isArchived, isReported, isPinned *bool
	var parentID *uuid.UUID

	if v := r.URL.Query().Get("is_deleted"); v != "" {
		val := v == "true"
		isDeleted = &val
	}
	if v := r.URL.Query().Get("is_resolved"); v != "" {
		val := v == "true"
		isResolved = &val
	}
	if v := r.URL.Query().Get("is_archived"); v != "" {
		val := v == "true"
		isArchived = &val
	}
	if v := r.URL.Query().Get("is_reported"); v != "" {
		val := v == "true"
		isReported = &val
	}
	if v := r.URL.Query().Get("is_pinned"); v != "" {
		val := v == "true"
		isPinned = &val
	}
	if v := r.URL.Query().Get("parent_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			respond.Error(w, apierror.BadRequest("invalid parent_id"))
			return
		}
		parentID = &id
	}

	var params struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.ListWithFilters(ctx, entityType, entityID, isDeleted, isResolved, isArchived, isReported, isPinned, parentID, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

// GetCommentsByIDs retrieves comments by their IDs.
//
//	@Summary		Get comments by IDs
//	@Description	Retrieve multiple comments by their UUIDs. Requires `comment:read` scope.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			ids	query		string	true	"Comma-separated comment UUIDs"
//	@Success		200	{object}	[]CommentResponse
//	@Router			/comments/batch [get]
func (h *HTTPHandler) GetCommentsByIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		respond.Error(w, apierror.BadRequest("ids parameter is required"))
		return
	}

	idStrs := strings.Split(idsParam, ",")
	ids := make([]uuid.UUID, 0, len(idStrs))
	for _, idStr := range idStrs {
		id, err := uuid.Parse(strings.TrimSpace(idStr))
		if err != nil {
			respond.Error(w, apierror.BadRequest("invalid id: "+idStr))
			return
		}
		ids = append(ids, id)
	}

	result, err := h.u.GetByIDs(ctx, ids)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

package commentreaction

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
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

func NewHTTPHandler(u UseCase, cfg *config.Config, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		u:         u,
		cfg:       cfg,
		validator: v,
	}
}

func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrDuplicateReaction) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	if errors.Is(err, ErrReactionNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

type ReactionRequest struct {
	ReactionType sqlc.ReactionType `json:"reaction_type" validate:"required,oneof=like love haha wow sad angry support"`
}

type ReactionListResponse struct {
	Reactions []sqlc.CommentReaction    `json:"reactions"`
	Meta      queryparam.PaginationMeta `json:"meta"`
}

func (h *HTTPHandler) GetReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reactionIDStr := chi.URLParam(r, "comment_reaction_id")
	reactionID, err := uuid.Parse(reactionIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid reaction id"))
		return
	}

	result, err := h.u.Get(ctx, reactionID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) GetReactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	commentIDStr := chi.URLParam(r, "comment_id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	params := queryparam.PaginationParams{}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	reactions, total, err := h.u.GetByCommentID(ctx, commentID, sqlc.EntityType(entityType), params.GetLimit(), params.GetOffset())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, ReactionListResponse{
		Reactions: *reactions,
		Meta:      queryparam.NewPaginationMeta(params.Page, params.Limit, total),
	})
}

func (h *HTTPHandler) React(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	commentIDStr := chi.URLParam(r, "comment_id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	var payload ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.React(ctx, sub.User.ID, commentID, sqlc.EntityType(entityType), payload.ReactionType)
	if err != nil {
		mapError(w, err)
		return
	}

	if result == nil {
		respond.JSON(w, http.StatusOK, map[string]string{"message": "reaction removed"})
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) Unreact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	commentIDStr := chi.URLParam(r, "comment_id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid comment id"))
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		respond.Error(w, apierror.BadRequest("entity_type is required"))
		return
	}

	err = h.u.Unreact(ctx, sub.User.ID, commentID, sqlc.EntityType(entityType))
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "reaction removed"})
}

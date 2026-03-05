package commentvote

import (
	"encoding/json"
	"errors"
	"net/http"

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
	if errors.Is(err, ErrDuplicateVote) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

type VoteRequest struct {
	VoteType sqlc.VoteType `json:"vote_type" validate:"required,oneof=up down"`
}

type VoteListResponse struct {
	Votes []sqlc.CommentVote        `json:"votes"`
	Meta  queryparam.PaginationMeta `json:"meta"`
}

func (h *HTTPHandler) GetVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	voteIDStr := chi.URLParam(r, "comment_vote_id")
	voteID, err := uuid.Parse(voteIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid vote id"))
		return
	}

	result, err := h.u.Get(ctx, voteID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, result)
}

func (h *HTTPHandler) GetVotes(w http.ResponseWriter, r *http.Request) {
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

	votes, total, err := h.u.GetByCommentID(ctx, commentID, sqlc.EntityType(entityType), params.GetLimit(), params.GetOffset())
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, VoteListResponse{
		Votes: votes,
		Meta:  queryparam.NewPaginationMeta(params.Page, params.Limit, total),
	})
}

func (h *HTTPHandler) Vote(w http.ResponseWriter, r *http.Request) {
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

	var payload VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&payload); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	input := &VoteInput{
		CommentID:  commentID,
		EntityType: sqlc.EntityType(entityType),
		UserID:     sub.User.ID,
		VoteType:   payload.VoteType,
	}

	result, err := h.u.Vote(ctx, input)
	if err != nil {
		mapError(w, err)
		return
	}

	if result.Removed {
		respond.JSON(w, http.StatusOK, map[string]string{"message": result.Message})
		return
	}

	respond.OK(w, result.Vote)
}

func (h *HTTPHandler) Unvote(w http.ResponseWriter, r *http.Request) {
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

	err = h.u.Unvote(ctx, commentID, sqlc.EntityType(entityType), sub.User.ID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "vote removed"})
}

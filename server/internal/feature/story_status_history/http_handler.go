package story_status_history

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/respond"
)

//go:generate mockgen -destination=mocks/mock_http_handler.go -source=http_handler.go -package=mocks

type HTTPHandler struct {
	u   UseCase
	cfg *config.Config
}

func NewHTTPHandler(u UseCase, cfg *config.Config) *HTTPHandler {
	return &HTTPHandler{
		u:   u,
		cfg: cfg,
	}
}

func mapError(w http.ResponseWriter, err error) {
	if err == ErrNotFound {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if err == ErrUnauthorized || err == ErrForbidden {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	respond.Error(w, apierror.Internal())
}

// GetStoryStatusHistory retrieves the status history for a story.
//
//	@Summary		Get story status history
//	@Description	Retrieves the status change history for a story. Requires `user` actor and `story:read` scope.
//	@Tags			story-status-history
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	StatusHistoryListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/status-history [get]
func (h *HTTPHandler) GetStoryStatusHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	// Parse pagination params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	offset := (page - 1) * limit

	history, total, err := h.u.ListStoryStatusHistoryPaginated(r.Context(), storyID, int32(limit), int32(offset))
	if err != nil {
		mapError(w, err)
		return
	}

	resp := StatusHistoryListResponse{
		History: history,
		Total:   total,
	}

	respond.OK(w, resp)
}

// GetStatusHistoryEntry retrieves a specific status history entry.
//
//	@Summary		Get status history entry
//	@Description	Retrieves a specific status history entry by ID. Requires `user` actor and `story:read` scope.
//	@Tags			story-status-history
//	@Produce		json
//	@Security		BearerAuth
//	@Param			history_id	path		string	true	"Status History UUID"
//	@Success		200			{object}	StatusHistoryResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/status-history/{history_id} [get]
func (h *HTTPHandler) GetStatusHistoryEntry(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "history_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid history id"))
		return
	}

	entry, err := h.u.GetStatusHistory(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, entry)
}

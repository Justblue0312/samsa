package notification

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/pkg/apierror"
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

func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	res, err := h.u.List(ctx, sub.User, filter)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) GetUnread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	limitParam := chi.URLParam(r, "limit")
	pageParam := chi.URLParam(r, "page")

	limit := config.APIPaginationDefaultLimit
	if limitParam != "" {
		l, err := strconv.ParseInt(limitParam, 10, 32)
		if err != nil {
			respond.Error(w, apierror.BadRequest(err.Error()))
			return
		}
		limit = int32(l)
	}

	page := config.APIPaginationDefaultPage
	if pageParam != "" {
		p, err := strconv.ParseInt(pageParam, 10, 32)
		if err != nil {
			respond.Error(w, apierror.BadRequest(err.Error()))
			return
		}
		page = int32(p)
	}

	res, err := h.u.GetUnread(ctx, sub.User, limit, page)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "notification_id")
	notiID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	res, err := h.u.GetByID(ctx, sub.User, notiID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, res)
}

func (h *HTTPHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "notification_id")
	notiID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	// Use new method that also notifies WS clients
	err = h.u.MarkAsReadWithNotify(ctx, sub.User, notiID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "notification marked as read"})
}

func (h *HTTPHandler) MarkAsUnread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "notification_id")
	notiID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	err = h.u.MarkAsUnread(ctx, sub.User, notiID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "notification marked as unread"})
}

func (h *HTTPHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	err = h.u.MarkAllAsRead(ctx, sub.User)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "all notifications marked as read"})
}

func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "notification_id")
	notiID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	// Use new method that also notifies WS clients
	err = h.u.DeleteWithNotify(ctx, sub.User, notiID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "notification deleted"})
}

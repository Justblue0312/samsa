package story_report

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
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotModerator) || errors.Is(err, ErrNotReporter) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrInvalidTransition) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

// CreateReport creates a new report.
//
//	@Summary		Create report
//	@Description	Creates a new report for a story. Requires `user` actor and `story.report:write` scope.
//	@Tags			story-reports
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateReportRequest	true	"Report creation request"
//	@Success		201		{object}	ReportResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		409		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/story-reports [post]
func (h *HTTPHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateReport(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetReport retrieves a report by ID.
//
//	@Summary		Get report by ID
//	@Description	Retrieves a report by its UUID. Requires `user` actor and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			report_id	path		string	true	"Report UUID"
//	@Success		200			{object}	ReportResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports/{report_id} [get]
func (h *HTTPHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	resp, err := h.u.GetReport(ctx, id)
	if err != nil {
		mapError(w, err)
		return
	}

	// Check if user can view this report (reporter or moderator)
	if resp.ReporterID != sub.User.ID && !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	respond.OK(w, resp)
}

// UpdateReport updates a report.
//
//	@Summary		Update report
//	@Description	Updates a report. Reporter can update description, moderators can update status. Requires `user` actor and `story.report:write` scope.
//	@Tags			story-reports
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			report_id	path		string				true	"Report UUID"
//	@Param			request		body		UpdateReportRequest	true	"Report update request"
//	@Success		200			{object}	ReportResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports/{report_id} [patch]
func (h *HTTPHandler) UpdateReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	var req UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	isModerator := sub.IsModerator()
	resp, err := h.u.UpdateReport(ctx, id, sub.User.ID, isModerator, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// DeleteReport deletes a report.
//
//	@Summary		Delete report
//	@Description	Deletes a report. Only reporter or moderator can delete. Requires `user` actor and `story.report:write` scope.
//	@Tags			story-reports
//	@Security		BearerAuth
//	@Param			report_id	path	string	true	"Report UUID"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		403	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/story-reports/{report_id} [delete]
func (h *HTTPHandler) DeleteReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	isModerator := sub.IsModerator()
	err = h.u.DeleteReport(ctx, id, sub.User.ID, isModerator)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// ResolveReport resolves a report.
//
//	@Summary		Resolve report
//	@Description	Resolves a report (moderator only). Requires `user` actor, moderator role, and `story.report:write` scope.
//	@Tags			story-reports
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			report_id	path		string					true	"Report UUID"
//	@Param			request		body		ResolveReportRequest	false	"Resolution notes"
//	@Success		200			{object}	ReportResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports/{report_id}/resolve [post]
func (h *HTTPHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	var req ResolveReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ResolveReport(ctx, id, sub.User.ID, req.ResolutionNotes)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// RejectReport rejects a report.
//
//	@Summary		Reject report
//	@Description	Rejects a report (moderator only). Requires `user` actor, moderator role, and `story.report:write` scope.
//	@Tags			story-reports
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			report_id	path		string				true	"Report UUID"
//	@Param			request		body		RejectReportRequest	false	"Rejection reason"
//	@Success		200			{object}	ReportResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports/{report_id}/reject [post]
func (h *HTTPHandler) RejectReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	var req RejectReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.RejectReport(ctx, id, sub.User.ID, req.RejectionReason)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// ArchiveReport archives a report.
//
//	@Summary		Archive report
//	@Description	Archives a report (moderator only). Requires `user` actor, moderator role, and `story.report:write` scope.
//	@Tags			story-reports
//	@Security		BearerAuth
//	@Param			report_id	path		string	true	"Report UUID"
//	@Success		200			{object}	ReportResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports/{report_id}/archive [post]
func (h *HTTPHandler) ArchiveReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "report_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid report id"))
		return
	}

	resp, err := h.u.ArchiveReport(ctx, id, sub.User.ID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// ListReports lists reports with filters.
//
//	@Summary		List reports
//	@Description	Retrieves a paginated list of reports. Moderators see all, users see their own. Requires `user` actor and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Param			story_id	query		string	false	"Filter by story UUID"
//	@Param			status		query		string	false	"Filter by status (pending, resolved, rejected, archived)"
//	@Param			is_resolved	query		bool	false	"Filter by resolved status"
//	@Success		200			{object}	ReportListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/story-reports [get]
func (h *HTTPHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	isModerator := sub.IsModerator()
	reports, total, err := h.u.ListReports(ctx, filter, isModerator, sub.User.ID)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := ReportListResponse{
		Reports: reports,
		Meta:    queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}

	respond.OK(w, resp)
}

// ListStoryReports lists reports for a story.
//
//	@Summary		List story reports
//	@Description	Retrieves a paginated list of reports for a story (moderator only). Requires `user` actor, moderator role, and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	ReportListResponse
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/reports [get]
func (h *HTTPHandler) ListStoryReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	reports, total, err := h.u.ListReportsByStory(ctx, storyID, filter)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := ReportListResponse{
		Reports: reports,
		Meta:    queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}

	respond.OK(w, resp)
}

// ListPendingReports lists all pending reports.
//
//	@Summary		List pending reports
//	@Description	Retrieves a paginated list of pending reports (moderator only). Requires `user` actor, moderator role, and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"Page number"
//	@Param			limit	query		int	false	"Items per page"
//	@Success		200		{object}	ReportListResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		403		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/story-reports/pending [get]
func (h *HTTPHandler) ListPendingReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	reports, total, err := h.u.ListPendingReports(ctx, filter)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := ReportListResponse{
		Reports: reports,
		Meta:    queryparam.NewPaginationMeta(filter.Page, filter.Limit, total),
	}

	respond.OK(w, resp)
}

// GetReportCount gets the count of reports for a story.
//
//	@Summary		Get report count
//	@Description	Retrieves the count of reports for a story (moderator only). Requires `user` actor, moderator role, and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Success		200			{object}	map[string]int64
//	@Failure		400			{object}	apierror.APIError
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		500			{object}	apierror.APIError
//	@Router			/stories/{story_id}/reports/count [get]
func (h *HTTPHandler) GetReportCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	count, err := h.u.GetReportCount(ctx, storyID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, map[string]int64{"count": count})
}

// GetPendingReportCount gets the count of pending reports.
//
//	@Summary		Get pending report count
//	@Description	Retrieves the count of pending reports (moderator only). Requires `user` actor, moderator role, and `story.report:read` scope.
//	@Tags			story-reports
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	map[string]int64
//	@Failure		401	{object}	apierror.APIError
//	@Failure		403	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/story-reports/pending/count [get]
func (h *HTTPHandler) GetPendingReportCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	count, err := h.u.GetPendingReportCount(ctx)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, map[string]int64{"count": count})
}

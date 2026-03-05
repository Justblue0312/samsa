package submission

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/queryparam"
	"github.com/justblue/samsa/pkg/respond"
)

type HTTPHandler struct {
	useCase   UseCase
	validator *validator.Validate
}

func NewHTTPHandler(useCase UseCase, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		useCase:   useCase,
		validator: v,
	}
}

func mapError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		respond.Error(w, apierror.NotFound(err.Error()))
		return
	}
	if errors.Is(err, ErrAlreadyClaimed) || errors.Is(err, ErrAlreadyApproved) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	if errors.Is(err, ErrNotPending) || errors.Is(err, ErrInvalidContext) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	if errors.Is(err, ErrNotRequester) || errors.Is(err, ErrNotApprover) || errors.Is(err, ErrNotAssignee) {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrInvalidTransition) {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

// GetSubmissions retrieves a paginated list of submissions.
//
//	@Summary		List submissions
//	@Description	Retrieves a list of submissions with pagination and optional filters. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page number"
//	@Param			limit			query		int		false	"Items per page"
//	@Param			order_by		query		string	false	"Sort field"
//	@Param			requester_id	query		string	false	"Filter by requester UUID"
//	@Param			approver_id		query		string	false	"Filter by approver UUID"
//	@Param			title			query		string	false	"Filter by title"
//	@Param			type			query		string	false	"Filter by submission_type"
//	@Param			status			query		string	false	"Filter by submission_status"
//	@Param			expose_id		query		string	false	"Filter by expose_id (SUB-XXXX)"
//	@Param			search_query	query		string	false	"FTS across title + message"
//	@Success		200				{object}	SubmissionResponses
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions [get]
func (h *HTTPHandler) GetSubmissions(w http.ResponseWriter, r *http.Request) {
	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid filter parameters"))
		return
	}

	submissions, totalCount, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	resp := SubmissionResponses{
		Submissions: responses,
		Meta:        queryparam.NewPaginationMeta(filter.Page, filter.Limit, totalCount),
	}

	respond.JSON(w, http.StatusOK, resp)
}

// GetMySubmissions retrieves a paginated list of submissions created by the authenticated user.
//
//	@Summary		List own submissions
//	@Description	Retrieves a list of submissions for the authenticated user with pagination and optional filters. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page number"
//	@Param			limit			query		int		false	"Items per page"
//	@Param			order_by		query		string	false	"Sort field"
//	@Param			approver_id		query		string	false	"Filter by approver UUID"
//	@Param			title			query		string	false	"Filter by title"
//	@Param			type			query		string	false	"Filter by submission_type"
//	@Param			status			query		string	false	"Filter by submission_status"
//	@Param			expose_id		query		string	false	"Filter by expose_id (SUB-XXXX)"
//	@Param			search_query	query		string	false	"FTS across title + message"
//	@Success		200				{object}	SubmissionResponses
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/me [get]
func (h *HTTPHandler) GetMySubmissions(w http.ResponseWriter, r *http.Request) {
	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid filter parameters"))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	submissions, totalCount, err := h.useCase.GetMySubmissions(r.Context(), userID, filter)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	resp := SubmissionResponses{
		Submissions: responses,
		Meta:        queryparam.NewPaginationMeta(filter.Page, filter.Limit, totalCount),
	}

	respond.JSON(w, http.StatusOK, resp)
}

// GetAvailableSubmissions retrieves a paginated list of pending submissions available for assignment.
//
//	@Summary		List available submissions
//	@Description	Retrieves a list of pending, unassigned submissions with pagination. Supports `anonymous` and `user` actors. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"Page number"
//	@Param			limit	query		int	false	"Items per page"
//	@Success		200		{object}	SubmissionResponses
//	@Failure		500		{object}	apierror.APIError
//	@Router			/submissions/available [get]
func (h *HTTPHandler) GetAvailableSubmissions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	offset := (page - 1) * limit

	submissions, totalCount, err := h.useCase.GetAvailable(r.Context(), limit, offset)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	resp := SubmissionResponses{
		Submissions: responses,
		Meta:        queryparam.NewPaginationMeta(int32(page), int32(limit), totalCount),
	}

	respond.JSON(w, http.StatusOK, resp)
}

// GetSubmission retrieves a submission by ID.
//
//	@Summary		Get submission
//	@Description	Retrieves a submission by its UUID. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id} [get]
func (h *HTTPHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	submission, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// GetSubmissionContext retrieves the context for a submission.
//
//	@Summary		Get submission context
//	@Description	Retrieves the context object for a submission by its UUID. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	SubmissionContext
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/context [get]
func (h *HTTPHandler) GetSubmissionContext(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	ctx, err := h.useCase.GetContext(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, ctx)
}

// GetSubmissionHistory retrieves the status history of a submission.
//
//	@Summary		Get submission history
//	@Description	Retrieves the status history list for a submission by its UUID. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{array}		sqlc.SubmissionStatusHistory
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/history [get]
func (h *HTTPHandler) GetSubmissionHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	history, err := h.useCase.ListStatusHistory(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, history)
}

// GetSubmissionAssignment retrieves the assignment details of a submission.
//
//	@Summary		Get submission assignment
//	@Description	Retrieves the assignment information for a submission by its UUID. Requires `user` actor and `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	sqlc.SubmissionAssignment
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/assignment [get]
func (h *HTTPHandler) GetSubmissionAssignment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	assignment, err := h.useCase.GetAssignment(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusOK, assignment)
}

// CreateSubmission creates a new submission.
//
//	@Summary		Create submission
//	@Description	Creates a new submission. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateSubmissionRequest	true	"Submission creation request"
//	@Success		201		{object}	SubmissionResponse
//	@Failure		400		{object}	apierror.APIError
//	@Failure		401		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/submissions [post]
func (h *HTTPHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	var req CreateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}
	req.RequesterID = userID

	submission, err := h.useCase.Create(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, apierror.NotFound(err.Error()))
			return
		}
		if errors.Is(err, ErrInvalidContext) {
			respond.Error(w, apierror.BadRequest(err.Error()))
			return
		}
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusCreated, h.buildSubmissionResponse(r.Context(), submission))
}

// UpdateSubmission updates an existing submission.
//
//	@Summary		Update submission
//	@Description	Updates an existing submission. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string					true	"Submission UUID"
//	@Param			request			body		UpdateSubmissionRequest	true	"Submission update request"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		409				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id} [patch]
func (h *HTTPHandler) UpdateSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	var req UpdateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	submission, err := h.useCase.Update(r.Context(), id, &req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// UpdateSubmissionContext updates the context of an existing submission.
//
//	@Summary		Update submission context
//	@Description	Updates the context map of an existing submission by its UUID. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string					true	"Submission UUID"
//	@Param			request			body		UpdateContextRequest	true	"Submission context update request"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/context [patch]
func (h *HTTPHandler) UpdateSubmissionContext(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	var req UpdateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	submission, err := h.useCase.UpdateContext(r.Context(), id, req.Context)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// ClaimSubmission claims an available submission for review.
//
//	@Summary		Claim submission
//	@Description	Claims an available unassigned submission for the authenticated user to act as approver. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		403				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		409				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/claim [post]
func (h *HTTPHandler) ClaimSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	submission, err := h.useCase.Claim(r.Context(), id, userID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// AssignSubmission assigns a submission to a specific user.
//
//	@Summary		Assign submission
//	@Description	Assigns an available submission to a specific user. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string					true	"Submission UUID"
//	@Param			request			body		AssignSubmissionRequest	true	"Assign submission request"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		403				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		409				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/assign [post]
func (h *HTTPHandler) AssignSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	var req AssignSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	submission, err := h.useCase.Assign(r.Context(), id, userID, req.AssignedTo)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// ApproveSubmissionWithReason approves a submission with a required reason.
//
//	@Summary		Approve submission
//	@Description	Marks an assigned submission as approved with an attached message. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string						true	"Submission UUID"
//	@Param			request			body		ApproveSubmissionRequest	true	"Approve submission request"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		403				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		409				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/approve [post]
func (h *HTTPHandler) ApproveSubmissionWithReason(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	var req ApproveSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	submission, err := h.useCase.ApproveWithReason(r.Context(), id, userID, req.Reason)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// RejectSubmissionWithReason rejects a submission with a required reason.
//
//	@Summary		Reject submission
//	@Description	Marks an assigned submission as rejected with an attached message. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string					true	"Submission UUID"
//	@Param			request			body		RejectSubmissionRequest	true	"Reject submission request"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		403				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		409				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/reject [post]
func (h *HTTPHandler) RejectSubmissionWithReason(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	var req RejectSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	// Get user ID from context (assuming middleware sets this)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	submission, err := h.useCase.RejectWithReason(r.Context(), id, userID, req.Reason)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// DeleteSubmission completely removes a submission.
//
//	@Summary		Delete submission
//	@Description	Completely removes a submission by its UUID. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path	string	true	"Submission UUID"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		404	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/submissions/{submission_id} [delete]
func (h *HTTPHandler) DeleteSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	err = h.useCase.Delete(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusNoContent, nil)
}

// BulkUpdateStatus updates the status of multiple submissions.
//
//	@Summary		Bulk update status
//	@Description	Updates the status of multiple submissions at once. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body	BulkUpdateStatusRequest	true	"Bulk update status request"
//	@Success		204
//	@Failure		400	{object}	apierror.APIError
//	@Failure		401	{object}	apierror.APIError
//	@Failure		404	{object}	apierror.APIError
//	@Failure		500	{object}	apierror.APIError
//	@Router			/submissions/bulk/status [patch]
func (h *HTTPHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.BadRequest("Invalid request body"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	err := h.useCase.BulkUpdateStatus(r.Context(), req.IDs, req.Status)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respond.Error(w, apierror.NotFound(err.Error()))
			return
		}
		if errors.Is(err, ErrInvalidTransition) {
			respond.Error(w, apierror.BadRequest(err.Error()))
			return
		}
		respond.Error(w, apierror.Internal())
		return
	}

	respond.JSON(w, http.StatusNoContent, nil)
}

// ArchiveSubmission archives a submission.
//
//	@Summary		Archive submission
//	@Description	Archives a submission by its UUID. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/archive [post]
func (h *HTTPHandler) ArchiveSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	submission, err := h.useCase.Archive(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// SoftDeleteSubmission soft-deletes a submission.
//
//	@Summary		Soft delete submission
//	@Description	Soft-deletes a submission by its UUID. Requires `user` actor and `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	apierror.APIError
//	@Failure		401				{object}	apierror.APIError
//	@Failure		404				{object}	apierror.APIError
//	@Failure		500				{object}	apierror.APIError
//	@Router			/submissions/{submission_id}/delete [patch]
func (h *HTTPHandler) SoftDeleteSubmission(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("Invalid submission ID"))
		return
	}

	submission, err := h.useCase.SoftDelete(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, h.buildSubmissionResponse(r.Context(), submission))
}

// buildSubmissionResponse creates a SubmissionResponse with status and optional history
func (h *HTTPHandler) buildSubmissionResponse(ctx context.Context, submission *sqlc.Submission) SubmissionResponse {
	response := SubmissionResponse{
		Submission: *submission,
	}

	// Optionally load status history (for single submission endpoints)
	history, err := h.useCase.ListStatusHistory(ctx, submission.ID)
	if err == nil && len(history) > 0 {
		response.StatusHistory = history
	}

	return response
}

// GetSubmissionsExceedingSLA lists submissions exceeding SLA.
//
//	@Summary		Get submissions exceeding SLA
//	@Description	Get list of submissions that have exceeded the SLA threshold. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			sla_hours	query		int	true	"SLA threshold in hours"
//	@Success		200			{object}	[]SubmissionResponse
//	@Router			/submissions/sla/exceeding [get]
func (h *HTTPHandler) GetSubmissionsExceedingSLA(w http.ResponseWriter, r *http.Request) {
	slaHoursStr := r.URL.Query().Get("sla_hours")
	if slaHoursStr == "" {
		respond.Error(w, apierror.BadRequest("sla_hours is required"))
		return
	}

	slaHours, err := strconv.Atoi(slaHoursStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid sla_hours"))
		return
	}

	submissions, err := h.useCase.GetSubmissionsExceedingSLA(r.Context(), slaHours)
	if err != nil {
		mapError(w, err)
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	respond.JSON(w, http.StatusOK, responses)
}

// CountSubmissionsExceedingSLA counts submissions exceeding SLA.
//
//	@Summary		Count submissions exceeding SLA
//	@Description	Get count of submissions that have exceeded the SLA threshold. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			sla_hours	query		int	true	"SLA threshold in hours"
//	@Success		200			{object}	map[string]int64
//	@Router			/submissions/sla/count-exceeding [get]
func (h *HTTPHandler) CountSubmissionsExceedingSLA(w http.ResponseWriter, r *http.Request) {
	slaHoursStr := r.URL.Query().Get("sla_hours")
	if slaHoursStr == "" {
		respond.Error(w, apierror.BadRequest("sla_hours is required"))
		return
	}

	slaHours, err := strconv.Atoi(slaHoursStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid sla_hours"))
		return
	}

	count, err := h.useCase.CountSubmissionsExceedingSLA(r.Context(), slaHours)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]int64{"count": count})
}

// GetSLAComplianceStats gets SLA compliance statistics.
//
//	@Summary		Get SLA compliance stats
//	@Description	Get SLA compliance statistics including compliant/non-compliant counts and rate. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			sla_hours	query		int	true	"SLA threshold in hours"
//	@Success		200			{object}	SLAComplianceStats
//	@Router			/submissions/sla/compliance-stats [get]
func (h *HTTPHandler) GetSLAComplianceStats(w http.ResponseWriter, r *http.Request) {
	slaHoursStr := r.URL.Query().Get("sla_hours")
	if slaHoursStr == "" {
		respond.Error(w, apierror.BadRequest("sla_hours is required"))
		return
	}

	slaHours, err := strconv.Atoi(slaHoursStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid sla_hours"))
		return
	}

	stats, err := h.useCase.GetSLAComplianceStats(r.Context(), slaHours)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, stats)
}

// GetAverageProcessingTime gets average processing time.
//
//	@Summary		Get average processing time
//	@Description	Get average processing time for approved submissions. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			days	query		int	true	"Number of days to calculate average"
//	@Success		200		{object}	map[string]float64
//	@Router			/submissions/sla/avg-processing-time [get]
func (h *HTTPHandler) GetAverageProcessingTime(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		respond.Error(w, apierror.BadRequest("days is required"))
		return
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid days"))
		return
	}

	avgTime, err := h.useCase.GetAverageProcessingTime(r.Context(), days)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]float64{"avg_processing_time_seconds": avgTime})
}

// GetSubmissionsBySLAStatus lists submissions by SLA status.
//
//	@Summary		Get submissions by SLA status
//	@Description	Get submissions filtered by SLA compliance status. Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			include_compliant	query		bool	true	"Include compliant (true) or non-compliant (false)"
//	@Param			sla_hours			query		int		true	"SLA threshold in hours"
//	@Param			page				query		int		false	"Page number"
//	@Param			limit				query		int		false	"Items per page"
//	@Success		200					{object}	[]SubmissionResponse
//	@Router			/submissions/sla/by-status [get]
func (h *HTTPHandler) GetSubmissionsBySLAStatus(w http.ResponseWriter, r *http.Request) {
	includeCompliantStr := r.URL.Query().Get("include_compliant")
	if includeCompliantStr == "" {
		respond.Error(w, apierror.BadRequest("include_compliant is required"))
		return
	}

	includeCompliant := includeCompliantStr == "true"

	slaHoursStr := r.URL.Query().Get("sla_hours")
	if slaHoursStr == "" {
		respond.Error(w, apierror.BadRequest("sla_hours is required"))
		return
	}

	slaHours, err := strconv.Atoi(slaHoursStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid sla_hours"))
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

	submissions, err := h.useCase.GetSubmissionsBySLAStatus(r.Context(), includeCompliant, slaHours, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	respond.JSON(w, http.StatusOK, responses)
}

// GetSubmissionPendingDuration gets pending duration for a submission.
//
//	@Summary		Get submission pending duration
//	@Description	Get how long a submission has been in pending status (in seconds). Requires `submission:read` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			submission_id	path		string	true	"Submission UUID"
//	@Success		200				{object}	map[string]int
//	@Router			/submissions/{submission_id}/pending-duration [get]
func (h *HTTPHandler) GetSubmissionPendingDuration(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "submission_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid submission_id"))
		return
	}

	duration, err := h.useCase.GetPendingDuration(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]int{"pending_seconds": duration})
}

// BulkUpdateSLABreach marks submissions as SLA breach.
//
//	@Summary		Bulk update SLA breach
//	@Description	Mark multiple submissions as SLA breach. Requires `submission:write` scope.
//	@Tags			submissions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		BulkUpdateSLABreachRequest	true	"Submission IDs to mark as breach"
//	@Success		200		{object}	[]SubmissionResponse
//	@Router			/submissions/sla/bulk-breach [post]
func (h *HTTPHandler) BulkUpdateSLABreach(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateSLABreachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	submissions, err := h.useCase.BulkUpdateSLABreach(r.Context(), req.IDs)
	if err != nil {
		mapError(w, err)
		return
	}

	responses := make([]SubmissionResponse, len(submissions))
	for i, s := range submissions {
		responses[i] = SubmissionResponse{Submission: *s}
	}

	respond.JSON(w, http.StatusOK, responses)
}

// BulkUpdateSLABreachRequest is the request payload for bulk SLA breach update.
type BulkUpdateSLABreachRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1"`
}

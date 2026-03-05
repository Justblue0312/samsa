package document

import (
	"encoding/json"
	"io"
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
	if err == ErrPermissionDenied {
		respond.Error(w, apierror.Forbidden())
		return
	}
	if err == ErrDocumentNotFound {
		respond.Error(w, apierror.NotFound("document not found"))
		return
	}
	respond.Error(w, apierror.Internal())
}

// Create creates a new document.
//
//	@Summary		Create document
//	@Description	Creates a new document. Requires `user actor`.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateDocumentRequest	true	"Document creation request"
//	@Success		201		{object}	DocumentResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/documents [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateDocument(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetByID retrieves a document by ID.
//
//	@Summary		Get document by ID
//	@Description	Retrieves a document by its UUID.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			document_id	path		string	true	"Document UUID"
//	@Success		200			{object}	DocumentResponse
//	@Failure		404			{object}	apierror.APIError
//	@Router			/documents/{document_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.GetDocument(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.NotFound("document not found"))
		return
	}

	respond.OK(w, resp)
}

// GetBySlug retrieves a document by slug.
//
//	@Summary		Get document by slug
//	@Description	Retrieves a document by its slug.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Param			slug	path		string	true	"Document slug"
//	@Success		200		{object}	DocumentResponse
//	@Failure		404		{object}	apierror.APIError
//	@Router			/documents/slug/{slug} [get]
func (h *HTTPHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	resp, err := h.u.GetDocumentBySlug(r.Context(), slug)
	if err != nil {
		respond.Error(w, apierror.NotFound("document not found"))
		return
	}

	respond.OK(w, resp)
}

// List retrieves documents.
//
//	@Summary		List documents
//	@Description	Retrieves documents based on filters.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Param			story_id	query	string	false	"Story UUID"
//	@Param			folder_id	query	string	false	"Folder UUID"
//	@Param			status		query	string	false	"Document status"
//	@Param			limit		query	int		false	"Limit"
//	@Param			offset		query	int		false	"Offset"
//	@Success		200			{array}	DocumentResponse
//	@Router			/documents [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	var params ListDocumentsParams
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListDocuments(r.Context(), params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing document.
//
//	@Summary		Update document
//	@Description	Updates document details. Requires `user actor` and ownership.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			document_id	path		string					true	"Document UUID"
//	@Param			request		body		UpdateDocumentRequest	true	"Document update request"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	var req UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateDocument(ctx, sub.User.ID, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete soft deletes a document.
//
//	@Summary		Delete document
//	@Description	Soft deletes a document. Requires `user actor` and ownership.
//	@Tags			documents
//	@Security		BearerAuth
//	@Param			document_id	path	string	true	"Document UUID"
//	@Success		204
//	@Router			/documents/{document_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	err = h.u.DeleteDocument(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// SubmitForReview submits a document for review.
//
//	@Summary		Submit document for review
//	@Description	Submits a document for review. Requires `user actor` and ownership.
//	@Tags			documents
//	@Security		BearerAuth
//	@Param			document_id	path		string	true	"Document UUID"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id}/submit [post]
func (h *HTTPHandler) SubmitForReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.SubmitForReview(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Approve approves a document.
//
//	@Summary		Approve document
//	@Description	Approves a document. Requires approval permissions.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			document_id	path		string				true	"Document UUID"
//	@Param			request		body		map[string]string	false	"Comments"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id}/approve [post]
func (h *HTTPHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	var req struct {
		Comments *string `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ApproveDocument(ctx, sub.User.ID, id, req.Comments)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Reject rejects a document.
//
//	@Summary		Reject document
//	@Description	Rejects a document. Requires approval permissions.
//	@Tags			documents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			document_id	path		string				true	"Document UUID"
//	@Param			request		body		map[string]string	false	"Comments"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id}/reject [post]
func (h *HTTPHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	var req struct {
		Comments *string `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.RejectDocument(ctx, sub.User.ID, id, req.Comments)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Archive archives a document.
//
//	@Summary		Archive document
//	@Description	Archives a document. Requires `user actor` and ownership.
//	@Tags			documents
//	@Security		BearerAuth
//	@Param			document_id	path		string	true	"Document UUID"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id}/archive [post]
func (h *HTTPHandler) Archive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.ArchiveDocument(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetVersionHistory retrieves version history for a document.
//
//	@Summary		Get document version history
//	@Description	Retrieves version history for a document.
//	@Tags			documents
//	@Produce		json
//	@Param			document_id	path	string	true	"Document UUID"
//	@Success		200			{array}	DocumentResponse
//	@Router			/documents/{document_id}/versions [get]
func (h *HTTPHandler) GetVersionHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.GetVersionHistory(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetStatusHistory retrieves status history for a document.
//
//	@Summary		Get document status history
//	@Description	Retrieves status history for a document.
//	@Tags			documents
//	@Produce		json
//	@Param			document_id	path	string	true	"Document UUID"
//	@Success		200			{array}	sqlc.DocumentStatusHistory
//	@Router			/documents/{document_id}/status-history [get]
func (h *HTTPHandler) GetStatusHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.GetStatusHistory(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// IncrementView increments the view count for a document.
//
//	@Summary		Increment document views
//	@Description	Increments the view count for a document.
//	@Tags			documents
//	@Produce		json
//	@Param			document_id	path		string	true	"Document UUID"
//	@Success		200			{object}	DocumentResponse
//	@Router			/documents/{document_id}/view [post]
func (h *HTTPHandler) IncrementView(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "document_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid document id"))
		return
	}

	resp, err := h.u.IncrementView(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

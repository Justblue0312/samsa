package document_folder

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	if err == ErrFolderNotFound {
		respond.Error(w, apierror.NotFound("folder not found"))
		return
	}
	if err == ErrMaxDepthExceeded {
		respond.Error(w, apierror.UnprocessableEntity("maximum folder depth exceeded (max 3 levels)"))
		return
	}
	if err == ErrFolderNotEmpty {
		respond.Error(w, apierror.UnprocessableEntity("folder is not empty"))
		return
	}
	respond.Error(w, apierror.Internal())
}

// Create creates a new document folder.
//
//	@Summary		Create document folder
//	@Description	Creates a new document folder. Requires `user actor`.
//	@Tags			document-folders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateDocumentFolderRequest	true	"Folder creation request"
//	@Success		201		{object}	DocumentFolderResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Failure		500		{object}	apierror.APIError
//	@Router			/document-folders [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	var req CreateDocumentFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	// Set owner ID from user context
	req.OwnerID = sub.User.ID

	resp, err := h.u.CreateFolder(ctx, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// GetByID retrieves a folder by ID.
//
//	@Summary		Get document folder by ID
//	@Description	Retrieves a document folder by its UUID.
//	@Tags			document-folders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			folder_id	path		string	true	"Folder UUID"
//	@Success		200			{object}	DocumentFolderResponse
//	@Failure		404			{object}	apierror.APIError
//	@Router			/document-folders/{folder_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	resp, err := h.u.GetFolder(r.Context(), id)
	if err != nil {
		respond.Error(w, apierror.NotFound("folder not found"))
		return
	}

	respond.OK(w, resp)
}

// List retrieves document folders.
//
//	@Summary		List document folders
//	@Description	Retrieves document folders based on filters.
//	@Tags			document-folders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	query	string	false	"Story UUID"
//	@Param			parent_id	query	string	false	"Parent Folder UUID"
//	@Param			limit		query	int		false	"Limit"
//	@Param			offset		query	int		false	"Offset"
//	@Success		200			{array}	DocumentFolderResponse
//	@Router			/document-folders [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	var params ListDocumentFoldersParams
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListFolders(r.Context(), params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing folder.
//
//	@Summary		Update document folder
//	@Description	Updates folder details. Requires `user actor` and ownership.
//	@Tags			document-folders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			folder_id	path		string						true	"Folder UUID"
//	@Param			request		body		UpdateDocumentFolderRequest	true	"Folder update request"
//	@Success		200			{object}	DocumentFolderResponse
//	@Router			/document-folders/{folder_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	var req UpdateDocumentFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateFolder(ctx, sub.User.ID, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete soft deletes a folder.
//
//	@Summary		Delete document folder
//	@Description	Deletes a folder. Requires `user actor` and ownership. Folder must be empty.
//	@Tags			document-folders
//	@Security		BearerAuth
//	@Param			folder_id	path	string	true	"Folder UUID"
//	@Success		204
//	@Failure		422	{object}	apierror.APIError
//	@Router			/document-folders/{folder_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	err = h.u.DeleteFolder(ctx, sub.User.ID, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

// Move moves a folder to a new parent.
//
//	@Summary		Move document folder
//	@Description	Moves a folder to a new parent. Requires `user actor` and ownership.
//	@Tags			document-folders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			folder_id	path		string						true	"Folder UUID"
//	@Param			request		body		MoveDocumentFolderRequest	true	"Move request"
//	@Success		200			{object}	DocumentFolderResponse
//	@Router			/document-folders/{folder_id}/move [post]
func (h *HTTPHandler) Move(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	var req MoveDocumentFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.MoveFolder(ctx, sub.User.ID, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetTree retrieves the folder tree.
//
//	@Summary		Get folder tree
//	@Description	Retrieves the folder tree starting from a folder.
//	@Tags			document-folders
//	@Produce		json
//	@Param			folder_id	path	string	true	"Folder UUID"
//	@Success		200			{array}	DocumentFolderResponse
//	@Router			/document-folders/{folder_id}/tree [get]
func (h *HTTPHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	resp, err := h.u.GetFolderTree(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetAncestors retrieves ancestor folders.
//
//	@Summary		Get folder ancestors
//	@Description	Retrieves ancestor folders of a folder.
//	@Tags			document-folders
//	@Produce		json
//	@Param			folder_id	path	string	true	"Folder UUID"
//	@Success		200			{array}	DocumentFolderResponse
//	@Router			/document-folders/{folder_id}/ancestors [get]
func (h *HTTPHandler) GetAncestors(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	resp, err := h.u.GetAncestors(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetDescendants retrieves descendant folders.
//
//	@Summary		Get folder descendants
//	@Description	Retrieves descendant folders of a folder.
//	@Tags			document-folders
//	@Produce		json
//	@Param			folder_id	path	string	true	"Folder UUID"
//	@Success		200			{array}	DocumentFolderResponse
//	@Router			/document-folders/{folder_id}/descendants [get]
func (h *HTTPHandler) GetDescendants(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "folder_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid folder id"))
		return
	}

	resp, err := h.u.GetDescendants(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Search searches for folders.
//
//	@Summary		Search document folders
//	@Description	Searches for document folders by name.
//	@Tags			document-folders
//	@Produce		json
//	@Param			q		query	string	true	"Search query"
//	@Param			limit	query	int		false	"Limit"
//	@Param			offset	query	int		false	"Offset"
//	@Success		200		{array}	DocumentFolderResponse
//	@Router			/document-folders/search [get]
func (h *HTTPHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil {
			limit = int32(parsed)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 32); err == nil {
			offset = int32(parsed)
		}
	}

	resp, err := h.u.SearchFolders(r.Context(), query, limit, offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

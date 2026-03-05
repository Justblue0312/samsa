package file

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/queryparam"
	"github.com/justblue/samsa/pkg/respond"
)

type HTTPHandler struct {
	u         UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func NewHTTPHandler(cfg *config.Config, v *validator.Validate, u UseCase) *HTTPHandler {
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
	if errors.Is(err, ErrAlreadyExists) {
		respond.Error(w, apierror.Conflict(err.Error()))
		return
	}
	respond.Error(w, apierror.Internal())
}

func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		respond.Error(w, apierror.BadRequest("failed to parse form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respond.Error(w, apierror.BadRequest("file is required"))
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	path := r.FormValue("path")
	if path == "" {
		respond.Error(w, apierror.BadRequest("path is required"))
		return
	}

	source := r.FormValue("source")
	if source == "" {
		source = "file"
	}

	req := &CreateRequest{
		Name:   name,
		Path:   path,
		Source: sqlc.FileUploadSource(source),
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	ownerID, err := uuid.Parse(r.Header.Get("X-Owner-ID"))
	if err != nil {
		ownerID = uuid.Nil
	}

	result, err := h.u.Create(r.Context(), ownerID, req, file)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, result)
}

func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := Validate(r)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	result, err := h.u.List(r.Context(), filter)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.GetByID(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respond.Error(w, apierror.BadRequest("failed to read request body"))
		return
	}
	defer r.Body.Close()

	var req UpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respond.Error(w, apierror.BadRequest("invalid JSON"))
		return
	}

	result, err := h.u.Update(r.Context(), id, &req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	err = h.u.Delete(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.GetDownloadURL(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// ShareFile marks a file as shared.
//
//	@Summary		Share file
//	@Description	Mark a file as shared (makes it publicly accessible). Requires `file:write` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"File UUID"
//	@Success		200	{object}	FileResponse
//	@Router			/files/{id}/share [post]
func (h *HTTPHandler) ShareFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.ShareFile(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// UnshareFile marks a file as private.
//
//	@Summary		Unshare file
//	@Description	Mark a file as private. Requires `file:write` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"File UUID"
//	@Success		200	{object}	FileResponse
//	@Router			/files/{id}/unshare [post]
func (h *HTTPHandler) UnshareFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.UnshareFile(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// ListSharedFiles lists all shared files.
//
//	@Summary		List shared files
//	@Description	List all publicly shared files. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"Page number"
//	@Param			limit	query		int	false	"Items per page"
//	@Success		200		{object}	ListResponse
//	@Router			/files/shared [get]
func (h *HTTPHandler) ListSharedFiles(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.ListSharedFiles(r.Context(), params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// GetFilesByOwnerAndType lists files by owner and MIME type.
//
//	@Summary		Get files by owner and type
//	@Description	List files filtered by owner ID and MIME type. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner_id	query		string	true	"Owner UUID"
//	@Param			mime_type	query		string	true	"MIME type (e.g., image/*)"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	ListResponse
//	@Router			/files/by-owner-type [get]
func (h *HTTPHandler) GetFilesByOwnerAndType(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		respond.Error(w, apierror.BadRequest("owner_id is required"))
		return
	}

	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid owner_id"))
		return
	}

	mimeType := r.URL.Query().Get("mime_type")
	if mimeType == "" {
		respond.Error(w, apierror.BadRequest("mime_type is required"))
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

	result, err := h.u.GetFilesByOwnerAndType(r.Context(), ownerID, mimeType, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// GetFilesByMimeType lists files by MIME type.
//
//	@Summary		Get files by MIME type
//	@Description	List files filtered by MIME type. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			mime_type	query		string	true	"MIME type (e.g., image/jpeg)"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	ListResponse
//	@Router			/files/by-mime-type [get]
func (h *HTTPHandler) GetFilesByMimeType(w http.ResponseWriter, r *http.Request) {
	mimeType := r.URL.Query().Get("mime_type")
	if mimeType == "" {
		respond.Error(w, apierror.BadRequest("mime_type is required"))
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

	result, err := h.u.GetFilesByMimeType(r.Context(), mimeType, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// CountFilesByMimeType counts files by MIME type.
//
//	@Summary		Count files by MIME type
//	@Description	Count total files with a specific MIME type. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			mime_type	query		string	true	"MIME type"
//	@Success		200			{object}	map[string]int64
//	@Router			/files/count-by-mime-type [get]
func (h *HTTPHandler) CountFilesByMimeType(w http.ResponseWriter, r *http.Request) {
	mimeType := r.URL.Query().Get("mime_type")
	if mimeType == "" {
		respond.Error(w, apierror.BadRequest("mime_type is required"))
		return
	}

	count, err := h.u.CountFilesByMimeType(r.Context(), mimeType)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]int64{"count": count})
}

// GetTotalSizeByOwner gets total storage size used by an owner.
//
//	@Summary		Get total size by owner
//	@Description	Get total storage size used by an owner. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner_id	query		string	true	"Owner UUID"
//	@Success		200			{object}	map[string]int64
//	@Router			/files/total-size [get]
func (h *HTTPHandler) GetTotalSizeByOwner(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		respond.Error(w, apierror.BadRequest("owner_id is required"))
		return
	}

	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid owner_id"))
		return
	}

	totalSize, err := h.u.GetTotalSizeByOwner(r.Context(), ownerID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]int64{"total_size_bytes": totalSize})
}

// SoftDeleteFile soft deletes a file.
//
//	@Summary		Soft delete file
//	@Description	Soft delete a file (can be restored). Requires `file:write` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"File UUID"
//	@Success		200	{object}	FileResponse
//	@Router			/files/{id}/trash [post]
func (h *HTTPHandler) SoftDeleteFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.SoftDeleteFile(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// RestoreFile restores a soft deleted file.
//
//	@Summary		Restore file
//	@Description	Restore a soft deleted file. Requires `file:write` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"File UUID"
//	@Success		200	{object}	FileResponse
//	@Router			/files/{id}/restore [post]
func (h *HTTPHandler) RestoreFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid file ID"))
		return
	}

	result, err := h.u.RestoreFile(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

// ListFilesWithFilters lists files with advanced filters.
//
//	@Summary		List files with filters
//	@Description	List files with advanced filtering options. Requires `file:read` scope.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			owner_id	query		string	false	"Filter by owner UUID"
//	@Param			mime_type	query		string	false	"Filter by MIME type"
//	@Param			reference	query		string	false	"Filter by reference (shared/private)"
//	@Param			is_archived	query		bool	false	"Filter by archived status"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Items per page"
//	@Success		200			{object}	ListResponse
//	@Router			/files/filter [get]
func (h *HTTPHandler) ListFilesWithFilters(w http.ResponseWriter, r *http.Request) {
	var ownerID, mimeType, reference *string
	var isArchived *bool

	if v := r.URL.Query().Get("owner_id"); v != "" {
		ownerID = &v
	}
	if v := r.URL.Query().Get("mime_type"); v != "" {
		mimeType = &v
	}
	if v := r.URL.Query().Get("reference"); v != "" {
		reference = &v
	}
	if v := r.URL.Query().Get("is_archived"); v != "" {
		val := v == "true"
		isArchived = &val
	}

	var params struct {
		Limit  int32 `json:"limit"`
		Offset int32 `json:"offset"`
	}
	if err := queryparam.DecodeRequest(&params, r.URL.RawQuery); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	result, err := h.u.ListFilesWithFilters(r.Context(), ownerID, mimeType, reference, isArchived, params.Limit, params.Offset)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, result)
}

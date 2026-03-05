package file

import "github.com/go-chi/chi/v5"

func (h *HTTPHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/download", h.GetDownloadURL)

	// File sharing endpoints
	r.Post("/{id}/share", h.ShareFile)
	r.Post("/{id}/unshare", h.UnshareFile)
	r.Get("/shared", h.ListSharedFiles)

	// File validation endpoints
	r.Get("/by-owner-type", h.GetFilesByOwnerAndType)
	r.Get("/by-mime-type", h.GetFilesByMimeType)
	r.Get("/count-by-mime-type", h.CountFilesByMimeType)
	r.Get("/total-size", h.GetTotalSizeByOwner)

	// Soft delete endpoints
	r.Post("/{id}/trash", h.SoftDeleteFile)
	r.Post("/{id}/restore", h.RestoreFile)

	// Filter endpoint
	r.Get("/filter", h.ListFilesWithFilters)
}

package comment

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/comments", func(r chi.Router) {
		r.Post("/", h.CreateComment)
		r.Get("/", h.GetComments)
		r.Get("/{comment_id}", h.GetComment)
		r.Patch("/{comment_id}", h.UpdateComment)
		r.Delete("/{comment_id}", h.DeleteComment)

		r.Post("/{comment_id}/report", h.Moderate("report"))
		r.Post("/{comment_id}/restore", h.Moderate("restore"))
		r.Post("/{comment_id}/pin", h.Moderate("pin"))
		r.Post("/{comment_id}/unpin", h.Moderate("unpin"))
		r.Post("/{comment_id}/resolve", h.Moderate("resolve"))
		r.Post("/{comment_id}/archive", h.Moderate("archive"))

		// Bulk moderation endpoints
		r.Post("/bulk/delete", h.BulkDeleteComments)
		r.Post("/bulk/archive", h.BulkArchiveComments)
		r.Post("/bulk/resolve", h.BulkResolveComments)
		r.Post("/bulk/pin", h.BulkPinComments)
		r.Post("/bulk/unpin", h.BulkUnpinComments)

		// Search and filter endpoints
		r.Get("/search", h.SearchComments)
		r.Get("/filter", h.ListCommentsWithFilters)
		r.Get("/batch", h.GetCommentsByIDs)
	})

}

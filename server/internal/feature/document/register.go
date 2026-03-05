package document

import (
	"github.com/go-chi/chi/v5"
)

func RegisterDocumentRoutes(router chi.Router, handler *HTTPHandler) {
	router.Route("/documents", func(r chi.Router) {
		r.Post("/", handler.Create)
		r.Get("/", handler.List)

		r.Route("/{document_id}", func(r chi.Router) {
			r.Get("/", handler.GetByID)
			r.Patch("/", handler.Update)
			r.Delete("/", handler.Delete)

			r.Post("/submit", handler.SubmitForReview)
			r.Post("/approve", handler.Approve)
			r.Post("/reject", handler.Reject)
			r.Post("/archive", handler.Archive)

			r.Get("/versions", handler.GetVersionHistory)
			r.Get("/status-history", handler.GetStatusHistory)

			r.Post("/view", handler.IncrementView)
		})

		r.Route("/slug/{slug}", func(r chi.Router) {
			r.Get("/", handler.GetBySlug)
		})
	})
}

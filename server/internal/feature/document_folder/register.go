package document_folder

import (
	"github.com/go-chi/chi/v5"
)

func RegisterDocumentFolderRoutes(router chi.Router, handler *HTTPHandler) {
	router.Route("/document-folders", func(r chi.Router) {
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/search", handler.Search)

		r.Route("/{folder_id}", func(r chi.Router) {
			r.Get("/", handler.GetByID)
			r.Patch("/", handler.Update)
			r.Delete("/", handler.Delete)

			r.Post("/move", handler.Move)
			r.Get("/tree", handler.GetTree)
			r.Get("/ancestors", handler.GetAncestors)
			r.Get("/descendants", handler.GetDescendants)
		})
	})
}

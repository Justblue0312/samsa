package chapter

import (
	"github.com/go-chi/chi/v5"
)

func RegisterChapterRoutes(router chi.Router, handler *HTTPHandler) {
	router.Route("/chapters", func(r chi.Router) {
		r.Post("/", handler.Create)
		r.Get("/", handler.List)

		r.Route("/{chapter_id}", func(r chi.Router) {
			r.Get("/", handler.GetByID)
			r.Patch("/", handler.Update)
			r.Delete("/", handler.Delete)

			r.Patch("/publish", handler.Publish)
			r.Patch("/unpublish", handler.Unpublish)
			r.Patch("/reorder", handler.Reorder)
			r.Post("/view", handler.IncrementView)
		})
	})
}

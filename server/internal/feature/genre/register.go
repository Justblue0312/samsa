package genre

import (
	"github.com/go-chi/chi/v5"
)

func RegisterGenreRoutes(router chi.Router, handler *HTTPHandler) {
	router.Route("/genres", func(r chi.Router) {
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Put("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})

	router.Route("/stories/{storyID}/genres", func(r chi.Router) {
		r.Get("/", handler.GetStoryGenres)
		r.Post("/", handler.AddToStory)
		r.Delete("/{genreID}", handler.RemoveFromStory)
	})
}

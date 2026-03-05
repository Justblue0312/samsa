package story_post

import "github.com/go-chi/chi/v5"

func RegisterStoryPostRoutes(router chi.Router, h *HTTPHandler) {
	router.Route("/story-posts", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{post_id}", h.GetByID)
		r.Patch("/{post_id}", h.Update)
		r.Delete("/{post_id}", h.Delete)
	})

	router.Route("/authors/{author_id}/posts", func(r chi.Router) {
		r.Get("/", h.ListByAuthor)
	})

	router.Route("/stories/{story_id}/posts", func(r chi.Router) {
		r.Get("/", h.ListByStory)
	})
}

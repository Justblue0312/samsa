package story

import "github.com/go-chi/chi/v5"

func RegisterStoryRoutes(router chi.Router, h *HTTPHandler) {
	router.Route("/stories", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)

		r.Route("/{story_id}", func(r chi.Router) {
			r.Get("/", h.GetByID)
			r.Patch("/", h.Update)
			r.Delete("/", h.Delete)

			r.Patch("/publish", h.Publish)
			r.Patch("/archive", h.Archive)

			r.Route("/vote", func(r chi.Router) {
				r.Post("/", h.Vote)
				r.Delete("/", h.DeleteVote)
				r.Get("/stats", h.GetVoteStats)
			})
		})
	})
}

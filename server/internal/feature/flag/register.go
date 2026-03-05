package flag

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

// RegisterHTTPEndpoints registers the flag HTTP endpoints
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/admin/flags", func(r chi.Router) {
		// All flag endpoints require moderator authentication
		r.Use(middleware.RequireActor(subject.UserActor))
		r.Use(middleware.RequireScopes(subject.WebWriteScope))

		// List flags - GET /admin/flags
		r.Get("/", h.List)

		// Create flag - POST /admin/flags
		r.Post("/", h.Create)

		r.Route("/{flag_id}", func(r chi.Router) {
			// Get flag by ID - GET /admin/flags/{flag_id}
			r.Get("/", h.GetByID)

			// Update flag - PATCH /admin/flags/{flag_id}
			r.Patch("/", h.Update)

			// Delete flag - DELETE /admin/flags/{flag_id}
			r.Delete("/", h.Delete)
		})
	})

	// Routes nested under stories
	r.Route("/stories", func(r chi.Router) {
		// List flags for a story - GET /stories/{story_id}/flags
		// Requires moderator authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/flags", h.ListByStory)
	})
}

package tag

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

// RegisterHTTPEndpoints registers the tag HTTP endpoints
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/tags", func(r chi.Router) {
		// Create tag - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/", h.CreateTag)

		// List tags - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/", h.ListTags)

		// Search tags - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/search", h.SearchTags)

		// Get tags by IDs (batch) - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/batch", h.GetTagsByIDs)

		// Get tag by ID - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{tag_id}", h.GetTag)

		// Update tag - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Patch("/{tag_id}", h.UpdateTag)

		// Delete tag - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Delete("/{tag_id}", h.DeleteTag)
	})

	// Routes for entities
	r.Route("/entities", func(r chi.Router) {
		// Get tags by entity - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{entity_type}/{entity_id}/tags", h.GetTagsByEntity)

		// Get tag count by entity - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{entity_type}/{entity_id}/tags/count", h.GetTagCount)
	})

	// Routes for owners
	r.Route("/owners", func(r chi.Router) {
		// Get tags by owner - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{owner_id}/tags", h.GetTagsByOwner)
	})
}

package story_vote

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

// RegisterHTTPEndpoints registers the story vote HTTP endpoints
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/story-votes", func(r chi.Router) {
		// Create/update vote - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/", h.CreateVote)

		// Get vote by ID - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{vote_id}", h.GetVote)

		// List user votes - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/users/{user_id}", h.ListUserVotes)
	})

	// Routes nested under stories
	r.Route("/stories", func(r chi.Router) {
		// Get user's vote for a story - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/my-vote", h.GetUserVote)

		// Delete user's vote for a story - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Delete("/{story_id}/vote", h.DeleteUserVote)

		// Get vote statistics - public endpoint
		r.Get("/{story_id}/vote-stats", h.GetVoteStats)

		// List votes for a story - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/votes", h.ListStoryVotes)
	})
}

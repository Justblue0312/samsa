package story_status_history

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

// RegisterHTTPEndpoints registers the story status history HTTP endpoints
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/stories", func(r chi.Router) {
		// Get story status history - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/status-history", h.GetStoryStatusHistory)
	})

	// Get specific status history entry - requires authentication and read scope
	r.With(middleware.RequireActor(subject.UserActor)).
		With(middleware.RequireScopes(subject.WebReadScope)).
		Get("/status-history/{history_id}", h.GetStatusHistoryEntry)
}

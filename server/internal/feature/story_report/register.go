package story_report

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

// RegisterHTTPEndpoints registers the story report HTTP endpoints
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/story-reports", func(r chi.Router) {
		// Create report - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/", h.CreateReport)

		// List reports - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/", h.ListReports)

		// List pending reports - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/pending", h.ListPendingReports)

		// Get pending count - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/pending/count", h.GetPendingReportCount)

		// Get report by ID - requires authentication and read scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{report_id}", h.GetReport)

		// Update report - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Patch("/{report_id}", h.UpdateReport)

		// Delete report - requires authentication and write scope
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Delete("/{report_id}", h.DeleteReport)

		// Resolve report - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/{report_id}/resolve", h.ResolveReport)

		// Reject report - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/{report_id}/reject", h.RejectReport)

		// Archive report - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebWriteScope)).
			Post("/{report_id}/archive", h.ArchiveReport)
	})

	// Routes nested under stories
	r.Route("/stories", func(r chi.Router) {
		// List reports for a story - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/reports", h.ListStoryReports)

		// Get report count for a story - moderator only
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.WebReadScope)).
			Get("/{story_id}/reports/count", h.GetReportCount)
	})
}

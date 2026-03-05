package submission

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

func RegisterHTTPEndpoint(r chi.Router, h *HTTPHandler) {
	r.Route("/submissions", func(r chi.Router) {
		// Public routes (no auth required)
		r.With(middleware.RequireActor(subject.UserActor, subject.AnonymousActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/available", h.GetAvailableSubmissions)

		// Authenticated routes
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/", h.GetSubmissions)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/me", h.GetMySubmissions)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/{submission_id}", h.GetSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/{submission_id}/context", h.GetSubmissionContext)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/{submission_id}/history", h.GetSubmissionHistory)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/{submission_id}/assignment", h.GetSubmissionAssignment)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/{submission_id}/pending-duration", h.GetSubmissionPendingDuration)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/", h.CreateSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Patch("/{submission_id}", h.UpdateSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Patch("/{submission_id}/context", h.UpdateSubmissionContext)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/{submission_id}/claim", h.ClaimSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/{submission_id}/assign", h.AssignSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/{submission_id}/approve", h.ApproveSubmissionWithReason)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/{submission_id}/reject", h.RejectSubmissionWithReason)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/{submission_id}/archive", h.ArchiveSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Patch("/{submission_id}/delete", h.SoftDeleteSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Delete("/{submission_id}", h.DeleteSubmission)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Patch("/bulk/status", h.BulkUpdateStatus)

		// SLA tracking endpoints
		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/sla/exceeding", h.GetSubmissionsExceedingSLA)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/sla/count-exceeding", h.CountSubmissionsExceedingSLA)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/sla/compliance-stats", h.GetSLAComplianceStats)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/sla/avg-processing-time", h.GetAverageProcessingTime)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
			Get("/sla/by-status", h.GetSubmissionsBySLAStatus)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.SubmissionWriteScope, subject.WebWriteScope)).
			Post("/sla/bulk-breach", h.BulkUpdateSLABreach)
	})
}

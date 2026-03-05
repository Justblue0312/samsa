package notification

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoint(r chi.Router, h *HTTPHandler) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/unread", h.GetUnread)
		r.Get("/{notification_id}", h.GetByID)
		r.Patch("/{notification_id}/read", h.MarkAsRead)
		r.Patch("/{notification_id}/unread", h.MarkAsUnread)
		r.Post("/read-all", h.MarkAllAsRead)
		r.Delete("/{notification_id}", h.Delete)
	})
}

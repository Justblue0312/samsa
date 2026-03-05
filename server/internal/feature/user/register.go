package user

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Get("/me", h.Me)
	r.Get("/me/scopes", h.MeScopes)
	r.Delete("/me/oauth-accounts/{provider}", h.DisconnectProvider)

	r.Get("/{user_id}/online", h.IsOnline)
}

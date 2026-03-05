package google

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoint(r chi.Router, h *HTTPHandler) {
	r.Route("/google", func(r chi.Router) {
		r.Get("/authorize", h.Authorize)
		r.Get("/callback", h.Callback)
	})
}

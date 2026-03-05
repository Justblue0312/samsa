package github

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoint(r chi.Router, h *HTTPHandler) {
	r.Route("/github", func(r chi.Router) {
		r.Get("/authorize", h.Authorize)
		r.Get("/callback", h.Callback)
	})
}

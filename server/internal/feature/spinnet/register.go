package spinnet

import (
	"github.com/go-chi/chi/v5"
)

// RegisterHTTPEndpoints registers all spinnet HTTP endpoints.
func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/spinnets", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)

		r.Get("/syntax/{syntax}", h.GetBySyntax)

		r.Route("/{spinnet_id}", func(r chi.Router) {
			r.Get("/", h.GetByID)
			r.Put("/", h.Update)
			r.Delete("/", h.Delete)
		})
	})
}

// RegisterSpinnetRoutes is an alias for RegisterHTTPEndpoints for backward compatibility.
func RegisterSpinnetRoutes(router chi.Router) {
	// This function is deprecated. Use RegisterHTTPEndpoints instead.
	// Kept for backward compatibility.
	_ = router
}

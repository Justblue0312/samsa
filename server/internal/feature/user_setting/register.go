package user_setting

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoint(r *chi.Mux, h *HTTPHandler) {
	r.Get("/me/settings", h.GetMeSettings)
	r.Patch("/me/settings", h.UpdateMeSettings)
	r.Delete("/me/settings/reset", h.ResetMeSettings)
}

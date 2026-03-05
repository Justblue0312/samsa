package auth

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoint(r chi.Router, h *HTTPHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/register", h.Register)
		r.Post("/verification-email", h.SendVerificationEmail)
		r.Post("/verification-email/{code}", h.ConfirmVerificationEmail)
		r.Post("/password/change", h.ChangePassword)
		r.Post("/password/forgot", h.ForgotPassword)
		r.Post("/password/reset/{code}", h.ResetPassword)
	})
}

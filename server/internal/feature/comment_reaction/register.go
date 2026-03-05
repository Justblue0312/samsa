package commentreaction

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("comments/{comment_id}/reactions", func(r chi.Router) {
		r.Get("/{comment_reaction_id}", h.GetReaction)
		r.Get("/", h.GetReactions)
		r.Post("/", h.React)
		r.Delete("/", h.Unreact)
	})
}

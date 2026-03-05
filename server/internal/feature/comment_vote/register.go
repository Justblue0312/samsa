package commentvote

import "github.com/go-chi/chi/v5"

func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("comments/{comment_id}/votes", func(r chi.Router) {
		r.Get("/{comment_vote_id}", h.GetVote)
		r.Get("/", h.GetVotes)
		r.Post("/", h.Vote)
		r.Delete("/", h.Unvote)
	})
}

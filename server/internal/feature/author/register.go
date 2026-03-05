package author

import (
	"github.com/go-chi/chi/v5"
	"github.com/justblue/samsa/internal/transport/http/middleware"
	"github.com/justblue/samsa/pkg/subject"
)

func RegisterHTTPEndpoints(r chi.Router, h *HTTPHandler) {
	r.Route("/authors", func(r chi.Router) {
		r.With(middleware.RequireActor(subject.UserActor, subject.AnonymousActor)).
			With(middleware.RequireScopes(subject.AuthorReadScope, subject.WebReadScope)).
			Get("/", h.ListAuthors)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.AuthorReadScope, subject.WebReadScope)).
			Get("/me", h.GetMyAuthor)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.AuthorReadScope, subject.WebReadScope)).
			Get("/slug/{slug}", h.GetAuthorBySlug)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.AuthorReadScope, subject.WebReadScope)).
			Get("/{author_id}", h.GetAuthor)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.AuthorWriteScope, subject.WebWriteScope)).
			Post("/", h.CreateAuthor)

		r.With(middleware.RequireActor(subject.UserActor)).
			With(middleware.RequireScopes(subject.AuthorWriteScope, subject.WebWriteScope)).
			Patch("/{author_id}", h.UpdateAuthor)

		r.With(middleware.RequireActor(subject.ModeratorActor)).
			With(middleware.RequireScopes(subject.AuthorWriteScope, subject.WebWriteScope)).
			Delete("/{author_id}", h.DeleteAuthor)

		r.With(middleware.RequireActor(subject.ModeratorActor)).
			With(middleware.RequireScopes(subject.AuthorWriteScope, subject.WebWriteScope)).
			Patch("/{author_id}/recommend", h.SetRecommended)
	})
}

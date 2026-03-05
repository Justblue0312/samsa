package middleware

import (
	"context"
	"net/http"
	"slices"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/session"
	"github.com/justblue/samsa/pkg/subject"
)

// AuthSubject is a middleware that authenticates and authorizes subjects.
// It uses the session cookie to authenticate the subject and the scopes to authorize the subject.
func AuthSubject(cfg *config.Config, sessionRepo session.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle ignored paths
			if slices.Contains(cfg.IgnoredPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Handle authentication and authorization
			ctx := r.Context()
			sub := subject.NewAnonymous()

			cookie, err := r.Cookie(cfg.UserSessionCookieName)
			if err == nil && cookie != nil {
				sess, user, err := sessionRepo.GetByToken(ctx, cookie.Value, false)
				if err == nil && sess != nil && user != nil {
					scopes := sess.Scopes
					sub = subject.New(user, subject.StrToScope(scopes), sess)
				}
			}

			ctx = context.WithValue(ctx, common.AuthSubjectContextKey, sub)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireScopes is a middleware that requires the subject to have certain scopes.
// It uses the scopes to authorize the subject.
func RequireScopes(scopes ...subject.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sub, err := common.GetUserSubject(ctx)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Note: User must have AT LEAST ONE scope
			hasAnyScope := slices.ContainsFunc(scopes, sub.HasScope)
			if !hasAnyScope {
				http.Error(w, "Not Permitted", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireActor is a middleware that requires the subject to have certain actors.
// It uses the actors to authorize the subject.
func RequireActor(actors ...subject.Actor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sub, err := common.GetAuthSubject(ctx)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Note: User must have AT LEAST ONE actor
			hasAnyActor := slices.ContainsFunc(actors, func(a subject.Actor) bool { return sub.GetActor() == a })
			if !hasAnyActor {
				http.Error(w, "Not Permitted", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

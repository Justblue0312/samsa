package subject

import "github.com/justblue/samsa/gen/sqlc"

type Actor string

var (
	AnonymousActor Actor = "anonymous"
	UserActor      Actor = "user"
	ModeratorActor Actor = "moderator"
)

// AuthSubject represents an authenticated subject with user information and scopes.
type AuthSubject struct {
	User    *sqlc.User
	Scopes  []Scope
	Session *sqlc.Session

	setOfScopes map[Scope]struct{}
}

// New creates a new AuthSubject instance.
func New(user *sqlc.User, scopes []Scope, session *sqlc.Session) *AuthSubject {
	setOfScopes := make(map[Scope]struct{}, len(scopes))
	for _, scope := range scopes {
		setOfScopes[scope] = struct{}{}
	}

	return &AuthSubject{
		User:        user,
		Scopes:      scopes,
		Session:     session,
		setOfScopes: setOfScopes,
	}
}

// NewAnonymous creates a new anonymous AuthSubject instance.
// Note: Kind of bad pratice but it's work.
func NewAnonymous() *AuthSubject {
	return &AuthSubject{
		User:        nil,
		Scopes:      []Scope{},
		Session:     nil,
		setOfScopes: map[Scope]struct{}{},
	}
}

func (s *AuthSubject) IsAnonymous() bool         { return s.User == nil }
func (s *AuthSubject) IsUser() bool              { return s.User != nil && len(s.Scopes) > 0 }
func (s *AuthSubject) IsAdmin() bool             { return s.IsUser() && s.User.IsAdmin }
func (s *AuthSubject) IsStaff() bool             { return s.IsUser() && s.User.IsStaff }
func (s *AuthSubject) IsAuthor() bool            { return s.IsUser() && s.User.IsAuthor }
func (s *AuthSubject) HasScope(scope Scope) bool { return s.setOfScopes[scope] == struct{}{} }
func (s *AuthSubject) IsModerator() bool         { return s.IsUser() && (s.User.IsStaff || s.User.IsAdmin) }
func (s *AuthSubject) IsAnonymousOrUser() bool   { return s.IsAnonymous() || s.IsUser() }
func (s *AuthSubject) GetActor() Actor {
	if s.IsAnonymous() {
		return AnonymousActor
	}
	if s.IsUser() {
		return UserActor
	}
	if s.IsModerator() {
		return ModeratorActor
	}
	return AnonymousActor
}

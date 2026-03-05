package common

import (
	"context"
	"errors"

	"github.com/justblue/samsa/pkg/subject"
)

type ContentKey string

var (
	AuthSubjectContextKey ContentKey = "auth-subject"

	ErrNoSubject = errors.New("no subject found in context")
	ErrNotUser   = errors.New("subject is not a user")
)

func GetAuthSubject(ctx context.Context) (*subject.AuthSubject, error) {
	sub, ok := ctx.Value(AuthSubjectContextKey).(*subject.AuthSubject)
	if !ok {
		return nil, ErrNoSubject
	}
	return sub, nil
}

func GetUserSubject(ctx context.Context) (*subject.AuthSubject, error) {
	sub, ok := ctx.Value(AuthSubjectContextKey).(*subject.AuthSubject)
	if !ok {
		return nil, ErrNoSubject
	}
	if !sub.IsUser() {
		return nil, ErrNotUser
	}
	return sub, nil
}

package tag

import "errors"

var (
	ErrNotFound      = errors.New("tag not found")
	ErrAlreadyExists = errors.New("tag already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrNotOwner      = errors.New("user is not the tag owner")
)

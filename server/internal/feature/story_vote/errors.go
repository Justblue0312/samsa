package story_vote

import "errors"

var (
	ErrNotFound          = errors.New("vote not found")
	ErrAlreadyExists     = errors.New("vote already exists")
	ErrInvalidRating     = errors.New("invalid rating: must be between 1 and 5")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidTransition = errors.New("invalid status transition")
)

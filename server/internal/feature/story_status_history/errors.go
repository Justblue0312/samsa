package story_status_history

import "errors"

var (
	ErrNotFound     = errors.New("status history entry not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

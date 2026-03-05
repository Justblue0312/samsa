package story_report

import "errors"

var (
	ErrNotFound          = errors.New("report not found")
	ErrAlreadyExists     = errors.New("report already exists")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidStatus     = errors.New("invalid report status")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrNotModerator      = errors.New("user is not a moderator")
	ErrNotReporter       = errors.New("user is not the reporter")
)

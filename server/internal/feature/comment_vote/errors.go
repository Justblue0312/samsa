package commentvote

import (
	"errors"
)

var (
	ErrNotFound      = errors.New("comment vote not found")
	ErrDuplicateVote = errors.New("duplicate vote")
)

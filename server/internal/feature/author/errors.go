package author

import "errors"

var (
	ErrAuthorNotFound    = errors.New("author not found")
	ErrSlugTaken         = errors.New("author slug already taken")
	ErrAlreadyExists     = errors.New("author already exists for this user")
	ErrStageNameRequired = errors.New("author stage name is required")
	ErrIDRequired        = errors.New("author ID is required")
)

package apierror

import "net/http"

// APIError is the standard error response body sent to clients.
// It implements the error interface so it can travel through middleware chains.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// WithMessage allows chaining a new message onto an existing error, e.g. to add context.
func (e *APIError) WithMessage(message string) *APIError {
	e.Message = message
	return e
}

// WithDetails allows chaining additional details onto an existing error, e.g. to include validation errors.
func (e *APIError) WithDetails(details any) *APIError {
	e.Details = details
	return e
}

// HTTPStatus maps the error code to the appropriate HTTP status code.
// Centralised here so the mapping is never duplicated across handlers.
func (e *APIError) HTTPStatus() int {
	switch e.Code {
	case "BAD_REQUEST":
		return http.StatusBadRequest
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "NOT_FOUND":
		return http.StatusNotFound
	case "CONFLICT":
		return http.StatusConflict
	case "UNPROCESSABLE_ENTITY":
		return http.StatusUnprocessableEntity
	case "TOO_MANY_REQUESTS":
		return http.StatusTooManyRequests
	case "VALIDATION_ERROR":
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// ── Constructors ──────────────────────────────────────────────────────────────
// One per HTTP error class. Message is always passed by the caller —
// never hardcoded inside the constructor (except for generic errors like
// Forbidden and Unauthorized where the message is always the same).

func BadRequest(msg string) *APIError {
	return &APIError{Code: "BAD_REQUEST", Message: msg}
}

func Unauthorized() *APIError {
	return &APIError{Code: "UNAUTHORIZED", Message: "authentication required"}
}

func Forbidden() *APIError {
	return &APIError{Code: "FORBIDDEN", Message: "forbidden"}
}

func NotFound(msg string) *APIError {
	return &APIError{Code: "NOT_FOUND", Message: msg}
}

func Conflict(msg string) *APIError {
	return &APIError{Code: "CONFLICT", Message: msg}
}

func UnprocessableEntity(msg string) *APIError {
	return &APIError{Code: "UNPROCESSABLE_ENTITY", Message: msg}
}

func TooManyRequests() *APIError {
	return &APIError{Code: "TOO_MANY_REQUESTS", Message: "rate limit exceeded"}
}

func Internal() *APIError {
	return &APIError{Code: "INTERNAL_ERROR", Message: "an internal error occurred"}
}

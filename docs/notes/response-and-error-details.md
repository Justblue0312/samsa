# Go Response & Error Packages Best Practices

## Table of Contents

1. [Package Structure](#1-package-structure)
2. [Package Responsibilities](#2-package-responsibilities)
3. [Dependency Direction](#3-dependency-direction)
4. [`pkg/apierror`](#4-pkgapierror)
5. [`pkg/respond`](#5-pkgrespond)
6. [Using Both in a Handler](#6-using-both-in-a-handler)
7. [Domain Error Mapping](#7-domain-error-mapping)
8. [Middleware Integration](#8-middleware-integration)
9. [Rules Cheatsheet](#9-rules-cheatsheet)

---

## 1. Package Structure

Split `respond` and `apierror` into two separate packages. They have different jobs and different reasons to change.

```
internal/pkg/
  apierror/
    apierror.go       ← error shape, codes, constructors, HTTP status mapping
  respond/
    respond.go        ← writes responses to http.ResponseWriter
    decode.go         ← decodes and validates JSON request bodies
```

**Why not combine them?**

If combined, the package must import `net/http` for writing AND define error shapes — it becomes a grab-bag with two reasons to change. Splitting means:

- `apierror` can be used in tests, middleware, and domain layers without pulling in HTTP writing logic
- `respond` can evolve its encoding strategy (e.g. add msgpack support) without touching error definitions
- Each package has a single, clear job

---

## 2. Package Responsibilities

| Package                     | Job                            | Knows About                          | Does NOT Know About                  |
| --------------------------- | ------------------------------ | ------------------------------------ | ------------------------------------ |
| `pkg/apierror`              | Define error shapes and codes  | HTTP status codes (for mapping only) | `http.ResponseWriter`, domain models |
| `pkg/respond`               | Write responses to the wire    | `http.ResponseWriter`, `apierror`    | Domain models, feature packages      |
| `feature/*/handler_http.go` | Map domain errors to responses | Domain errors, `respond`, `apierror` | Raw JSON encoding                    |

---

## 3. Dependency Direction

```
feature/handler_http.go
  │
  ├──▶ pkg/respond      (writes to wire)
  │       │
  │       └──▶ pkg/apierror   (error shape + status mapping)
  │
  └──▶ pkg/apierror     (constructors used directly)
```

`apierror` has zero internal dependencies — it is the foundation.
`respond` depends only on `apierror`.
Handlers depend on both but never write raw JSON themselves.

---

## 4. `pkg/apierror`

Pure data — no `http.ResponseWriter`, no domain knowledge.

```go
// pkg/apierror/apierror.go
package apierror

import "net/http"

// APIError is the standard error response body sent to clients.
// It implements the error interface so it can travel through middleware chains.
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
    return e.Message
}

// HTTPStatus maps the error code to the appropriate HTTP status code.
// Centralised here so the mapping is never duplicated across handlers.
func (e *APIError) HTTPStatus() int {
    switch e.Code {
    case "BAD_REQUEST":           return http.StatusBadRequest
    case "UNAUTHORIZED":          return http.StatusUnauthorized
    case "FORBIDDEN":             return http.StatusForbidden
    case "NOT_FOUND":             return http.StatusNotFound
    case "CONFLICT":              return http.StatusConflict
    case "UNPROCESSABLE_ENTITY":  return http.StatusUnprocessableEntity
    case "TOO_MANY_REQUESTS":     return http.StatusTooManyRequests
    default:                      return http.StatusInternalServerError
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
```

### Design Notes

**`APIError` implements `error`** so it can be returned from middleware or passed through chi's middleware chain without type assertions everywhere.

**`HTTPStatus()` lives on the struct** — not in `respond`. This keeps the code↔status mapping in one place. If you add a new error code, you update one switch statement, not every handler.

**Constructors return `*APIError`** (pointer) so callers can pass it to `respond.Error()` directly without wrapping.

---

## 5. `pkg/respond`

Knows about `http.ResponseWriter`. All JSON encoding happens here — never in handlers.

```go
// pkg/respond/respond.go
package respond

import (
    "encoding/json"
    "log/slog"
    "net/http"

    "myapp/internal/pkg/apierror"
)

// JSON writes any value as a JSON response with the given status code.
// This is the foundation — all other helpers call this.
func JSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        // Encoding failure after WriteHeader — we can't change the status anymore.
        // Log it and move on.
        slog.Error("respond: failed to encode response", "error", err)
    }
}

// OK writes a 200 JSON response.
func OK(w http.ResponseWriter, data any) {
    JSON(w, http.StatusOK, data)
}

// Created writes a 201 JSON response.
func Created(w http.ResponseWriter, data any) {
    JSON(w, http.StatusCreated, data)
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
    w.WriteHeader(http.StatusNoContent)
}

// Error writes an apierror.APIError as a JSON response.
// The HTTP status code is derived from the error's own HTTPStatus() method —
// no status code duplication in callers.
func Error(w http.ResponseWriter, err *apierror.APIError) {
    JSON(w, err.HTTPStatus(), err)
}
```

```go
// pkg/respond/decode.go
package respond

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "strings"
)

const maxBodyBytes = 1 << 20 // 1 MB

// Decode reads the JSON request body into dst and returns a plain error.
// The caller decides how to respond — Decode itself never writes to the wire.
//
// Handles:
//   - body size limit (1 MB)
//   - unknown fields (returns error instead of silently ignoring)
//   - multiple JSON objects in one body (returns error)
//   - empty body
func Decode(r *http.Request, dst any) error {
    r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)

    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    if err := dec.Decode(dst); err != nil {
        return translateDecodeError(err)
    }

    // reject requests with more than one JSON object in the body
    if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
        return fmt.Errorf("request body must contain a single JSON object")
    }

    return nil
}

func translateDecodeError(err error) error {
    var syntaxErr *json.SyntaxError
    var unmarshalErr *json.UnmarshalTypeError

    switch {
    case errors.As(err, &syntaxErr):
        return fmt.Errorf("request body contains malformed JSON at position %d", syntaxErr.Offset)
    case errors.Is(err, io.ErrUnexpectedEOF):
        return fmt.Errorf("request body contains malformed JSON")
    case errors.As(err, &unmarshalErr):
        return fmt.Errorf("field '%s' must be of type %s", unmarshalErr.Field, unmarshalErr.Type)
    case errors.Is(err, io.EOF):
        return fmt.Errorf("request body must not be empty")
    case strings.HasPrefix(err.Error(), "json: unknown field"):
        field := strings.TrimPrefix(err.Error(), "json: unknown field ")
        return fmt.Errorf("unknown field %s", field)
    default:
        return fmt.Errorf("failed to decode request body: %w", err)
    }
}
```

### Design Notes

**`Decode` returns `error`, not `*apierror.APIError`** — it has no knowledge of how you want to respond. The handler calls `apierror.BadRequest(err.Error())` after checking the error. This keeps `decode.go` free of any HTTP writing concern.

**`Error(w, err)` takes `*apierror.APIError`, not a plain `error`** — this is intentional. It forces the handler to explicitly construct a typed API error rather than accidentally leaking an internal error message. You can never accidentally call `respond.Error(w, someInternalErr)`.

**`json.NewEncoder(w).Encode(data)` adds a trailing newline** — this is standard for JSON APIs and makes `curl` output readable without extra flags.

---

## 6. Using Both in a Handler

```go
// feature/user/handler_http.go
package user

import (
    "errors"
    "log/slog"
    "net/http"

    "myapp/internal/pkg/apierror"
    "myapp/internal/pkg/respond"
)

func (h *HTTPHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 1. Decode and validate request body
    var req createUserRequest
    if err := respond.Decode(r, &req); err != nil {
        respond.Error(w, apierror.BadRequest(err.Error()))
        return
    }

    // 2. RBAC check
    if roleFromCtx(r.Context()) != string(RoleAdmin) {
        respond.Error(w, apierror.Forbidden())
        return
    }

    // 3. Call usecase
    user, err := h.usecase.CreateUser(r.Context(), CreateUserInput{
        Name:  req.Name,
        Email: req.Email,
        Role:  Role(req.Role),
    })
    if err != nil {
        respondDomainError(w, r, err)
        return
    }

    // 4. Write success response
    respond.Created(w, toResponse(user))
}

func (h *HTTPHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    user, err := h.usecase.GetUser(r.Context(), id)
    if err != nil {
        respondDomainError(w, r, err)
        return
    }

    respond.OK(w, toResponse(user))
}

func (h *HTTPHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := h.usecase.DeleteUser(r.Context(), id); err != nil {
        respondDomainError(w, r, err)
        return
    }

    respond.NoContent(w)
}
```

---

## 7. Domain Error Mapping

Each feature defines its own `respondDomainError` function in `handler_http.go`. This function is the **only place** where domain errors are mapped to `apierror` constructors.

```go
// feature/user/handler_http.go (continued)

// respondDomainError maps feature-level domain errors → apierror → wire.
// This function is feature-specific — it lives here, not in pkg/respond.
// errors.As is always checked before errors.Is — typed errors are more specific.
func respondDomainError(w http.ResponseWriter, r *http.Request, err error) {
    // typed errors first — errors.As unwraps and gives access to fields
    var valErr *ValidationError
    if errors.As(err, &valErr) {
        respond.Error(w, apierror.BadRequest(valErr.Message))
        return
    }

    var notFound *NotFoundError
    if errors.As(err, &notFound) {
        respond.Error(w, apierror.NotFound(notFound.Error()))
        return
    }

    // sentinel errors — errors.Is handles wrapped sentinels correctly
    switch {
    case errors.Is(err, ErrEmailTaken):
        respond.Error(w, apierror.Conflict("email already taken"))
    case errors.Is(err, ErrUnauthorized):
        respond.Error(w, apierror.Forbidden())
    case errors.Is(err, ErrHasActiveSession):
        respond.Error(w, apierror.Conflict("user has an active session"))
    default:
        // unexpected — log internally, return generic message externally
        slog.Error("unhandled domain error",
            "error", err,
            "method", r.Method,
            "path", r.URL.Path,
        )
        respond.Error(w, apierror.Internal())
    }
}
```

### Why `respondDomainError` Is Per Feature, Not in `pkg/respond`

`respondDomainError` must import domain error types (`*ValidationError`, `ErrEmailTaken`, etc.) from the `user` package. If it lived in `pkg/respond`, then `pkg/respond` would import feature packages — violating the dependency rule that `pkg/` has no knowledge of domain.

Each feature has its own `respondDomainError`. That's not duplication — it's the right boundary.

---

## 8. Middleware Integration

Chi middleware can also use `respond` and `apierror` directly since `APIError` implements the `error` interface.

### Auth Middleware

```go
// internal/middleware/auth.go
package middleware

import (
    "net/http"

    "myapp/internal/pkg/apierror"
    "myapp/internal/pkg/respond"
)

type SessionValidator interface {
    ValidateToken(ctx context.Context, token string) (*Subject, error)
}

func Auth(sv SessionValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                respond.Error(w, apierror.Unauthorized())
                return
            }

            subject, err := sv.ValidateToken(r.Context(), token)
            if err != nil {
                respond.Error(w, apierror.Unauthorized())
                return
            }

            next.ServeHTTP(w, r.WithContext(withSubject(r.Context(), subject)))
        })
    }
}
```

### Rate Limit Middleware

```go
// internal/middleware/ratelimit.go
package middleware

import (
    "net/http"

    "myapp/internal/pkg/apierror"
    "myapp/internal/pkg/respond"
)

func RateLimit(rl RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !rl.Allow(r.Context()) {
                // apierror.TooManyRequests has no message param — message is always the same
                respond.Error(w, apierror.TooManyRequests())
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Recovery Middleware

```go
// internal/middleware/recovery.go
package middleware

import (
    "log/slog"
    "net/http"

    "myapp/internal/pkg/apierror"
    "myapp/internal/pkg/respond"
)

func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                slog.Error("panic recovered",
                    "recover", rec,
                    "method", r.Method,
                    "path", r.URL.Path,
                )
                respond.Error(w, apierror.Internal())
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

---

## 9. Rules Cheatsheet

### `pkg/apierror`

```
□ APIError implements error interface — can travel through middleware chains
□ HTTPStatus() lives on the struct — mapping is never duplicated in handlers
□ Constructors return *APIError — never a plain struct value
□ Generic errors (Forbidden, Unauthorized) have fixed messages — no msg param
□ Specific errors (BadRequest, NotFound) accept msg — caller provides detail
□ Never import feature packages or domain models
□ Never import http.ResponseWriter — pure data only
```

### `pkg/respond`

```
□ All JSON encoding happens here — handlers never call json.Marshal directly
□ respond.Error takes *apierror.APIError — never a plain error interface
□ respond.Decode returns plain error — caller maps to apierror, never Decode itself
□ Decode enforces: body size limit, DisallowUnknownFields, single JSON object
□ translateDecodeError gives human-readable messages for all JSON parse failures
□ Never import feature packages or domain models
□ Log encoding failures inside JSON() — they cannot change the status after WriteHeader
```

### Handlers

```
□ Never call json.Marshal or json.NewEncoder directly — always use respond.*
□ Never construct http status codes directly — always use respond.* helpers
□ Bind/decode errors → apierror.BadRequest — always inline, before any usecase call
□ RBAC errors → apierror.Forbidden — always inline, before any usecase call
□ Domain errors → respondDomainError — centralised per feature, not per handler method
□ errors.As checked before errors.Is — typed errors are more specific
□ Default case in respondDomainError always logs + returns apierror.Internal()
□ Never expose internal error details in any response — apierror.Internal() only
□ respondDomainError lives in handler_http.go — never in pkg/respond
```

### Middleware

```
□ Use respond.Error + apierror constructors — same pattern as handlers
□ Never write raw JSON in middleware
□ Recovery middleware always returns apierror.Internal() — never exposes panic detail
```

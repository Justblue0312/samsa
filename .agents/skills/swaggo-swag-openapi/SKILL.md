---
name: swaggo-swag-openapi
description: Guidelines for writing OpenAPI documentation using swaggo/swag in this Go project. Covers API info annotations, handler annotations, authentication documentation, parameters, and generation commands. Use when adding or updating API documentation for HTTP handlers.
---

# Swaggo/Swag OpenAPI Documentation

Guidelines for documenting APIs using swaggo/swag in this Samsa project.

---

## Overview

This project uses [swaggo/swag](https://github.com/swaggo/swag) to generate OpenAPI (Swagger) documentation from Go comments.

- **Generate command**: `task swagger`
- **Output**: `gen/swagger/` (swagger.yaml, swagger.json, docs.go)
- **Main entry**: `bootstrap/cli.go` - contains global API annotations

---

## API Info Annotations

Place in `bootstrap/cli.go` (usually near the `GetServeCmd` function):

```go
//	@title			Samsa
//	@version		0.1.0
//	@description	Samsa is a simple backend for a writing platform.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	https://github.com/justblue/samsa
//	@contact.email	trao0312@gmail.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8000
//	@BasePath	/api/v1

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
```

---

## Handler Annotations

Place directly above each HTTP handler function in `internal/feature/*/http_handler.go`.

### Required Annotations

| Annotation     | Description                                    |
| -------------- | ---------------------------------------------- |
| `@Summary`     | Brief one-line description                     |
| `@Description` | Detailed description (include actors & scopes) |
| `@Tags`        | API category (e.g., `authors`, `users`)        |
| `@Accept`      | Input content type (usually `json`)            |
| `@Produce`     | Output content type (usually `json`)           |
| `@Security`    | Security scheme (usually `BearerAuth`)         |
| `@Success`     | Success response code and type                 |
| `@Router`      | HTTP method and path pattern                   |

### Optional Annotations

| Annotation | Description                             |
| ---------- | --------------------------------------- |
| `@Param`   | Path, query, header, or body parameters |
| `@Header`  | Header parameters                       |
| `@Failure` | Error response codes and types          |

---

## Authentication Documentation

### Actors

Document the required actor in `@Description`:

- `anonymous` - Unauthenticated users
- `user` - Authenticated regular users
- `moderator` - Moderator users

### Scopes

Document the required scope in `@Description`:

- Format: `<scope>:read` or `<scope>:write`
- Examples: `author:read`, `author:write`, `user:read`, `story:write`

Note:

- Add new line of `@Description` to store actor and scope requirements.
- Add backtick symbols (`) for actor and scope references.

### Example Description

```go
// @Description Retrieves the author profile for the authenticated user.
// @Description Requires `user actor` and `author:read scope`.
```

---

## Parameters

### Path Parameters

```go
// @Param author_id path string true "Author UUID"
```

### Query Parameters

```go
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param order_by query string false "Sort field (created_at, updated_at)"
// @Param user_id query uuid.UUID false "Filter by user ID"
// @Param is_recommended query bool false "Filter by recommended status"
// @Param search_query query string false "Search query"
```

### Body Parameters

```go
// @Param request body CreateAuthorRequest true "Author creation request"
```

### Parameter Types

| Go Type     | Swagger Type               |
| ----------- | -------------------------- |
| `int`       | integer                    |
| `int32`     | integer                    |
| `int64`     | integer                    |
| `float32`   | number                     |
| `float64`   | number                     |
| `bool`      | boolean                    |
| `string`    | string                     |
| `uuid.UUID` | string (format: uuid)      |
| `time.Time` | string (format: date-time) |
| `[]byte`    | string (format: byte)      |

---

Note:

- If the API method have a various imput, for example

```go
// @Param request body CreateFooAuthorRequest true "Author foo creation request"
// @Param request body CreateBarAuthorRequest true "Author bar creation request"
```

We have to add both request bodies to the API method.

## Response Types

### Success Responses

```go
// @Success 200 {object} AuthorReposonse
// @Success 201 {object} AuthorReposonse
// @Success 204
```

### Failure Responses

```go
// @Failure 400 {object} apierror.APIError
// @Failure 401 {object} apierror.APIError
// @Failure 404 {object} apierror.APIError
// @Failure 500 {object} apierror.APIError
```

---

## Complete Example

```go
// CreateAuthor creates a new author profile.
// @Summary Create author
// @Description Creates a new author profile for the authenticated user. Requires user actor and author:write scope.
// @Tags authors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAuthorRequest true "Author creation request"
// @Success 201 {object} AuthorReposonse
// @Router /authors [post]
func (h *HTTPHandler) CreateAuthor(w http.ResponseWriter, r *http.Request) {
    // handler implementation
}
```

---

## Route Patterns

### Standard CRUD

| Operation | Method | Path              |
| --------- | ------ | ----------------- |
| List      | GET    | `/resources`      |
| Get by ID | GET    | `/resources/{id}` |
| Create    | POST   | `/resources`      |
| Update    | PATCH  | `/resources/{id}` |
| Delete    | DELETE | `/resources/{id}` |

### Custom Endpoints

```go
// GetAuthorBySlug
// @Router /authors/slug/{slug} [get]

// SetRecommended (moderator action)
// @Router /authors/{author_id}/recommend [patch]
```

---

## Generating Documentation

After adding or modifying swag comments:

```bash
# Using task (recommended)
task swagger

# Or directly with swag
swag init -o gen/swagger -g bootstrap/cli.go
swag fmt
```

This generates:

- `gen/swagger/docs.go` - Go documentation
- `gen/swagger/swagger.json` - OpenAPI 2.0 JSON
- `gen/swagger/swagger.yaml` - OpenAPI 2.0 YAML

---

## Common Patterns

### Listing with Filters

```go
// ListAuthors retrieves a paginated list of authors.
// @Summary List authors
// @Description Retrieves a list of authors with pagination and optional filters. Supports anonymous and authenticated users. Required scopes: author:read.
// @Tags authors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param order_by query string false "Sort field"
// @Param user_id query uuid.UUID false "Filter by user ID"
// @Param is_recommended query bool false "Filter by recommended"
// @Param search_query query string false "Search query"
// @Success 200 {object} AuthorResponses
// @Router /authors [get]
func (h *HTTPHandler) ListAuthors(w http.ResponseWriter, r *http.Request) {
```

### Soft Delete

```go
// DeleteAuthor soft deletes an author profile.
// @Summary Delete author
// @Description Soft deletes an author profile by their UUID. Requires moderator actor and author:write scope.
// @Tags authors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param author_id path string true "Author UUID"
// @Success 204
// @Router /authors/{author_id} [delete]
```

### Admin/Moderator Actions

```go
// SetRecommended sets the recommended status for an author.
// @Summary Set author recommended status
// @Description Sets the recommended status for an author. Requires moderator actor and author:read scope.
// @Tags authors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param author_id path string true "Author UUID"
// @Param request body SetRecommendedRequest true "Recommended status request"
// @Success 200 {object} AuthorReposonse
// @Router /authors/{author_id}/recommend [patch]
```

---

## Quick Reference

| Annotation     | Purpose                               |
| -------------- | ------------------------------------- |
| `@Summary`     | Brief title                           |
| `@Description` | Detailed info (include actor + scope) |
| `@Tags`        | API grouping                          |
| `@Accept`      | Request body type                     |
| `@Produce`     | Response body type                    |
| `@Security`    | Authentication                        |
| `@Param`       | Parameters (path, query, body)        |
| `@Success`     | Success response                      |
| `@Failure`     | Error response                        |
| `@Router`      | Route path and method                 |

**Run after changes:**

```bash
task swagger
```

---

## See Also

- **go-documentation** - Go documentation best practices
- **go-style-core** - Core Go style guidelines

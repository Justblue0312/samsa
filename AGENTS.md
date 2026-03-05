# Samsa Project - AGENTS.md

## Project Overview

**Samsa** is a backend platform for a writing/storytelling application built with Go. The name appears to reference Franz Kafka's "Metamorphosis" protagonist, fitting for a writing platform.

### Core Technologies

- **Language**: Go 1.25.7+
- **HTTP Framework**: chi/v5 (router), gorilla/websocket (real-time)
- **Database**: PostgreSQL with pgx/v5 driver
- **ORM**: sqlc (SQL-to-Go code generation)
- **Cache**: Redis (go-redis/v9)
- **Task Queue**: Hibiken/asynq (background jobs)
- **Validation**: go-playground/validator/v10
- **Testing**: testify, golang/mock
- **API Docs**: swaggo/swag (OpenAPI/Swagger)
- **Observability**: OpenTelemetry (traces, metrics)
- **Storage**: AWS S3 SDK (compatible with RustFS)

### Architecture

**Feature-based modular architecture** following clean architecture principles:

```
internal/feature/<feature_name>/
├── errors.go           # Sentinel errors (ErrNotFound, etc.)
├── models.go           # Request/Response DTOs
├── filter.go           # Query filters & pagination
├── repository.go       # Data access layer
├── usecase.go          # Business logic layer
├── http_handler.go     # HTTP handlers
├── register.go         # Route registration
├── status.go           # Status constants (if applicable)
├── tasks.go            # Background tasks (if applicable)
├── notifier.go         # WebSocket notifications (if applicable)
└── mocks/              # Generated mocks
```

**Request Flow**: `HTTP Handler → UseCase → Repository → Database`

### Key Features

The platform includes:

- **User Management**: Authentication, OAuth (Google, GitHub), sessions
- **Content**: Stories, chapters, documents, document folders
- **Social**: Comments, votes, reactions, flags
- **Submissions**: Review workflow with statuses (pending, claimed, assigned, approved, rejected, timeouted, archived)
- **Authors & Genres**: Content categorization
- **Files**: S3-based file storage
- **Notifications**: Real-time WebSocket notifications
- **Tags**: Entity tagging system

## Building and Running

### Prerequisites

- Go 1.25.7+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (for dependencies)
- Taskfile (optional but recommended)

### Setup

```bash
cd /home/justblue/Projects/samsa/server

# Copy environment template
cp .env.template .env
# Edit .env with your configuration

# Start dependencies (PostgreSQL, Redis, etc.)
task up

# Run migrations
task migrate:up

# Generate code (sqlc, mocks, swagger)
task generate

# Build
task build

# Run
task run
```

### Key Commands

```bash
# Build & Run
go build -ldflags="-s -w" -o bin/samsa cmd/samsa/*
go run cmd/samsa/main.go

# Test
go test ./... -v
go test -v -run ^TestName$ ./path/to/...   # exact match
go test -v -run TestName ./path/to/...     # pattern match

# Generate
go generate ./...              # Run all //go:generate directives
task sqlc                      # Generate SQLC types
task swagger                   # Generate OpenAPI docs

# Database
task migrate:up                # Apply migrations
task migrate:down              # Rollback last migration
task migrate:reset             # Rollback all migrations
task migrate:create NAME=foo   # Create new migration

# Lint & Format
golangci-lint run
gofmt -w . && goimports -w .

# Taskfile shortcuts
task build  task test  task lint  task run
```

### Environment Configuration

Key environment variables (see `.env.template`):

```bash
# Server
SAMSA_MODE=development          # development, production, testing
SAMSA_HTTP_PORT=8000
SAMSA_WS_PORT=8082

# Database
SAMSA_POSTGRES_HOST=localhost
SAMSA_POSTGRES_PORT=5432
SAMSA_POSTGRES_USER=samsa
SAMSA_POSTGRES_PWD=samsa
SAMSA_POSTGRES_DATABASE=samsa
SAMSA_POSTGRES_TEST_DATABASE=samsa_test

# Redis
SAMSA_REDIS_HOST=localhost
SAMSA_REDIS_PORT=6379

# OAuth (optional)
SAMSA_OAUTH2_GOOGLE_CLIENT_ID=...
SAMSA_OAUTH2_GITHUB_CLIENT_ID=...

# S3 Storage
SAMSA_AWS_ACCESS_KEY_ID=...
SAMSA_AWS_SECRET_ACCESS_KEY=...
SAMSA_AWS_S3_ENDPOINT_URL=http://rustfs:9000
```

## Development Conventions

### Code Style

**Imports** (enforced by golangci-lint):

```go
import (
    "context"    // stdlib first
    "fmt"        // stdlib
    "github.com/google/uuid"          // third-party
    "github.com/justblue/samsa/config" // internal
)
```

**Naming**:

- Packages: lowercase, single word (`file`, `user_setting`)
- Interfaces: PascalCase + `er` suffix (`Reader`, `UseCase`)
- Structs: PascalCase (`HTTPHandler`, `usecase`)
- Private fields: camelCase (`repo`, `cfg`)
- Errors: `Err` prefix (`ErrNotFound`)

**Struct Layout**:

```go
type HTTPHandler struct {
    u         UseCase
    cfg       *config.Config
    validator *validator.Validate
}
```

### Error Handling

```go
// Sentinel errors in errors.go
var (
    ErrNotFound    = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
)

// Wrap errors with context
fmt.Errorf("repo.Create: %w", err)

// Check errors
if errors.Is(err, ErrNotFound) { ... }
```

### HTTP Handlers

Handlers should only:

1. Authenticate/authorize
2. Parse parameters
3. Validate request body
4. Call use case
5. Respond

```go
func mapError(w http.ResponseWriter, err error) {
    if errors.Is(err, ErrNotFound) {
        respond.Error(w, apierror.NotFound(err.Error()))
        return
    }
    if errors.Is(err, ErrUnauthorized) {
        respond.Error(w, apierror.Unauthorized(err.Error()))
        return
    }
    respond.Error(w, apierror.Internal())
}
```

### Pagination

```go
respond.OK(w, ListResponse{
    Items: items,
    Meta:  queryparam.NewPaginationMeta(params.Page, params.Limit, total),
})
```

### Authorization

**Scopes**: Defined in `pkg/subject/scope.go`

```go
const (
    WebReadScope  Scope = "web:read"
    WebWriteScope Scope = "web:write"

    SubmissionReadScope  Scope = "submission:read"
    SubmissionWriteScope Scope = "submission:write"
    // ... etc
)
```

**Middleware Usage** (in `register.go`):

```go
r.With(middleware.RequireActor(subject.UserActor)).
  With(middleware.RequireScopes(subject.SubmissionReadScope, subject.WebReadScope)).
  Get("/", h.GetSubmissions)
```

### Validation

Use `validator.Validate` for request validation:

```go
type CreateSubmissionRequest struct {
    RequesterID uuid.UUID `json:"requester_id" validate:"required"`
    Title       string    `json:"title" validate:"required"`
    Tags        []string  `json:"tags" validate:"max=10"`
}

// In handler:
if err := h.validator.Struct(req); err != nil {
    // Handle validation error
}
```

### Testing

```go
// Use testify for assertions
assert.Equal(t, expected, actual)
require.NoError(t, err)

// Use gomock for mocks (generate with go generate)
//go:generate mockgen -source=usecase.go -destination=mocks/mock_usecase.go -package=mocks
```

### Database

**Migrations**: Located in `db/migrations/`, managed by goose
**SQLC**: Queries in `db/queries/`, schema in `db/migrations/`, generated code in `gen/sqlc/`

**Submission Statuses** (example workflow):

```
pending → claimed → approved → archived
pending → assigned → rejected → archived
pending → timeouted → archived
```

### WebSocket Notifications

Real-time notifications via `notification.Notifier`:

```go
notifier.NotifyNew(ctx, userID, notificationType, payload)
```

### Background Tasks

Asynq tasks for background processing:

```go
// tasks.go - Auto-timeout runs daily at 2 AM
// Cron: "0 2 * * *"
// Timeout threshold: 30 days inactivity
```

## Project Structure

```
/home/justblue/Projects/samsa/
├── server/                 # Go backend
│   ├── bootstrap/          # Application bootstrap
│   ├── cmd/                # Main entry point
│   ├── config/             # Configuration
│   ├── db/                 # SQLC config, migrations, queries
│   ├── gen/                # Generated code (sqlc, swagger)
│   ├── internal/           # Private application code
│   │   ├── common/         # Shared utilities
│   │   ├── feature/        # Feature modules
│   │   ├── infras/         # Infrastructure (postgres, redis, s3)
│   │   ├── testkit/        # Test utilities
│   │   └── transport/      # HTTP/gRPC middleware
│   ├── pkg/                # Public packages
│   │   └── subject/        # Authorization scopes
│   └── tools/              # Code generation tools
├── client/                 # Frontend (if exists)
├── docs/                   # Documentation
│   ├── notes/
│   ├── plans/
│   └── todos/
└── packages/               # Additional packages
    └── crawlbot/
```

## Key Files

- `server/Taskfile.yaml` - Task runner configuration
- `server/db/sqlc.yaml` - SQLC code generation config
- `server/.golangci.yaml` - Linter configuration
- `server/bootstrap/cli.go` - Application entry point
- `server/internal/feature/*/register.go` - Route registration per feature

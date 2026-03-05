# Samsa

A backend platform for a writing and storytelling application. Samsa provides a complete infrastructure for content creation, submission management, and community engagement.

## Overview

Samsa is designed to support a writing community platform with features including:

- **User Management** - Authentication, OAuth (Google, GitHub), sessions, and user settings
- **Content Management** - Stories, chapters, documents, and document folders
- **Submission Workflow** - Review system with statuses (pending, claimed, assigned, approved, rejected, timeout, archived)
- **Social Features** - Comments, votes, reactions, and flags
- **Content Organization** - Authors, genres, and tags
- **File Storage** - S3-compatible storage via RustFS
- **Real-time Notifications** - WebSocket-based notification system
- **Web Crawling** - Automated content discovery via Crawlbot

## Tech Stack

### Backend (Server)

- **Language**: Go 1.25.7+
- **HTTP Framework**: chi/v5 router with gorilla/websocket for real-time features
- **Database**: PostgreSQL 18 with pgx/v5 driver
- **ORM**: sqlc (SQL-to-Go code generation)
- **Cache**: DragonFlyDB / Redis
- **Task Queue**: Hibiken/asynq for background jobs
- **Validation**: go-playground/validator/v10
- **API Documentation**: swaggo/swag (OpenAPI/Swagger)
- **Observability**: OpenTelemetry (traces, metrics, logs)
- **Storage**: AWS S3 SDK (compatible with RustFS)

### Additional Packages

- **Crawlbot** (Python) - Web crawler for content discovery
- **Samsakit** (Python) - Supporting utilities and tools

### Infrastructure

- **Containerization**: Docker & Docker Compose
- **Monitoring**: Grafana, Prometheus, Jaeger, Loki
- **Log Collection**: Promtail
- **Object Storage**: RustFS

## Quick Start

### Prerequisites

- Go 1.25.7+
- Python 3.11+ (for Crawlbot and Samsakit)
- Docker & Docker Compose
- Taskfile (recommended)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd samsa/server

# Copy environment template
cp .env.template .env
# Edit .env with your configuration

# Start all dependencies (PostgreSQL, Redis, observability stack, RustFS)
task up

# Run database migrations
task migrate:up

# Generate code (sqlc types, mocks, swagger docs)
task generate

# Build the server
task build

# Run the server
task run
```

The server will start on `http://localhost:8000` (HTTP API) and `ws://localhost:8082` (WebSocket).

### Accessing Services

| Service | URL | Description |
|---------|-----|-------------|
| API Server | http://localhost:8000 | Main REST API |
| Swagger Docs | http://localhost:8000/swagger/index.html | API documentation |
| Grafana | http://localhost:3000 | Metrics & dashboards |
| Jaeger | http://localhost:17686 | Distributed tracing |
| Prometheus | http://localhost:9091 | Metrics storage |
| RustFS Console | http://localhost:9001 | Object storage UI |
| RustFS API | http://localhost:9000 | S3-compatible API |

## Architecture

Samsa follows a **feature-based modular architecture** with clean architecture principles:

```
internal/feature/<feature_name>/
├── errors.go           # Sentinel errors
├── models.go           # Request/Response DTOs
├── filter.go           # Query filters & pagination
├── repository.go       # Data access layer
├── usecase.go          # Business logic layer
├── http_handler.go     # HTTP handlers
├── register.go         # Route registration
└── tasks.go            # Background tasks (optional)
```

### Request Flow

```
HTTP Request → Middleware → HTTP Handler → UseCase → Repository → Database
                                              ↓
                                         Background Tasks (Async)
                                              ↓
                                         WebSocket Notifications
```

### Key Design Patterns

- **Repository Pattern**: Data access abstracted through interfaces
- **UseCase Pattern**: Business logic isolated from transport layer
- **Dependency Injection**: Components receive dependencies via constructor
- **Scope-based Authorization**: Fine-grained access control via middleware

## Development

### Key Commands

```bash
# Build & Run
go build -ldflags="-s -w" -o bin/samsa cmd/samsa/*
go run cmd/samsa/main.go

# Testing
go test ./... -v
go test -v -run ^TestName$ ./path/to/...   # exact match
go test -v -run TestName ./path/to/...     # pattern match

# Code Generation
go generate ./...              # Run all //go:generate directives
task sqlc                      # Generate SQLC types
task swagger                   # Generate OpenAPI docs
task mocks                     # Generate test mocks

# Database
task migrate:up                # Apply migrations
task migrate:down              # Rollback last migration
task migrate:reset             # Rollback all migrations
task migrate:create NAME=foo   # Create new migration

# Linting & Formatting
golangci-lint run
gofmt -w . && goimports -w .

# Docker
task up                        # Start all containers
task down                      # Stop all containers
task logs                      # View logs
```

### Environment Configuration

Key environment variables (see `server/.env.template` for full list):

```bash
# Server
SAMSA_MODE=development          # development, production, testing
SAMSA_HTTP_PORT=8000
SAMSA_GRPC_PORT=8001
SAMSA_WS_PORT=8082

# Database
SAMSA_POSTGRES_HOST=localhost
SAMSA_POSTGRES_PORT=5432
SAMSA_POSTGRES_USER=samsa
SAMSA_POSTGRES_PWD=samsa
SAMSA_POSTGRES_DATABASE=samsa

# Redis
SAMSA_REDIS_HOST=localhost
SAMSA_REDIS_PORT=6379

# OAuth (optional)
SAMSA_OAUTH2_GOOGLE_CLIENT_ID=...
SAMSA_OAUTH2_GITHUB_CLIENT_ID=...

# S3 Storage
SAMSA_AWS_ACCESS_KEY_ID=...
SAMSA_AWS_SECRET_ACCESS_KEY=...
SAMSA_AWS_S3_ENDPOINT_URL=http://localhost:9000
```

### Crawlbot (Python)

Crawlbot is a web crawler for content discovery. Located in `packages/crawlbot/`:

```bash
cd packages/crawlbot

# Install dependencies (using uv)
uv sync

# Run the crawler
uv run python src/main.py
```

### Samsakit (Python)

Supporting utilities and tools. Located in `packages/samsakit/`:

```bash
cd packages/samsakit

# Install dependencies
uv sync

# Use the toolkit
uv run python src/main.py
```

## Project Structure

```
samsa/
├── server/                 # Go backend (main application)
│   ├── bootstrap/          # Application bootstrap & CLI
│   ├── cmd/                # Main entry point
│   ├── config/             # Configuration management
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
│   ├── proto/              # Protocol buffer definitions
│   └── deploy/             # Docker & infrastructure configs
├── client/                 # Frontend application (TBD)
├── packages/               # Additional packages
│   ├── crawlbot/           # Web crawler (Python)
│   └── samsakit/           # Toolkit utilities (Python)
├── docs/                   # Documentation
│   ├── notes/              # Development notes
│   ├── plans/              # Design documents
│   └── todos/              # Task tracking
└── data/                   # Sample data & fixtures
```

## Submission Workflow

The submission review system follows this workflow:

```
                    ┌─────────────┐
                    │   pending   │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
   ┌──────────┐     ┌──────────┐     ┌─────────────┐
   │  claimed │     │ assigned │     │  timeouted  │
   └────┬─────┘     └────┬─────┘     └──────┬──────┘
        │                │                  │
        ▼                ▼                  │
   ┌──────────┐     ┌──────────┐            │
   │ approved │     │ rejected │            │
   └────┬─────┘     └────┬─────┘            │
        │                │                  │
        └────────────────┼──────────────────┘
                         ▼
                   ┌──────────┐
                   │ archived │
                   └──────────┘
```

Submissions can be:
- **Pending**: Awaiting review
- **Claimed**: Reviewer has claimed the submission
- **Assigned**: Assigned to a specific reviewer
- **Approved**: Accepted for publication
- **Rejected**: Declined
- **Timeouted**: No activity for 30 days
- **Archived**: Final state for all submissions

## Observability

Samsa includes a complete observability stack:

- **Metrics**: Prometheus collects metrics from the application
- **Tracing**: Jaeger provides distributed tracing
- **Logging**: Loki aggregates logs with Promtail as the collector
- **Dashboards**: Grafana provides visualization

All services are pre-configured in `docker-compose.yaml` and start with `task up`.

## Testing

```bash
# Run all tests
go test ./... -v

# Run specific test
go test -v -run ^TestName$ ./internal/feature/...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Generate mocks for testing
go generate ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes (`git commit -m 'Add some feature'`)
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

### Code Style

- Follow Go best practices and effective Go guidelines
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before committing
- Write tests for new features
- Keep functions small and focused
- Use meaningful variable and function names

## License

See the [LICENSE](server/LICENSE) file for details.

---

**Note**: This project is named in reference to Franz Kafka's "The Metamorphosis" protagonist, Gregor Samsa - fitting for a platform dedicated to writing and storytelling.

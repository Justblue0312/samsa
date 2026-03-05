# Go Server Architecture — Complete Reference

## Table of Contents

1. [Problems We Are Solving](#1-problems-we-are-solving)
2. [Full Directory Structure](#2-full-directory-structure)
3. [Layer Responsibilities](#3-layer-responsibilities)
4. [Cycle Import Prevention](#4-cycle-import-prevention)
5. [Feature Package — Complete Anatomy](#5-feature-package--complete-anatomy)
6. [Shared Packages (`pkg/`)](#6-shared-packages-pkg)
7. [Moderator / Admin APIs](#7-moderator--admin-apis)
8. [Asynq Task Workers](#8-asynq-task-workers)
9. [Transport Layer](#9-transport-layer)
10. [Bootstrap Layer](#10-bootstrap-layer)
11. [sqlc and the Repository Layer](#11-sqlc-and-the-repository-layer)
12. [Migration Steps](#12-migration-steps)
13. [Rules Cheatsheet](#13-rules-cheatsheet)

---

## 1. Problems We Are Solving

| Problem                                                   | Current State                                     | Impact                                    |
| --------------------------------------------------------- | ------------------------------------------------- | ----------------------------------------- |
| Domain inside transport                                   | `api/http/domain/`                                | HTTP layer owns business models           |
| Features split by transport                               | `api/http/v1/user` + `api/grpc/v2/user`           | One feature spans multiple folders        |
| Infrastructure called service                             | `service/postgres`, `service/redis`               | Confusing naming, unclear boundaries      |
| Direct cross-feature imports                              | `user` imports `project` imports `user`           | Import cycle compile error                |
| No clear async task ownership                             | `tasks.go` scattered loosely                      | Hard to trace what dispatches what        |
| Graceful shutdown, migration, config loading in `main.go` | Bloated entrypoint                                | Hard to test startup logic                |
| Shared utils with no home                                 | `respond`, `apierror`, `jwt`, `pwd` spread around | Imported inconsistently                   |
| Generated code inside `internal/`                         | `internal/gen/sqlc`                               | Build artifacts mixed with business logic |

---

## 2. Full Directory Structure

```
server
├── cmd
│   └── main.go                      ← 10 lines max, just calls bootstrap
│
├── db
│   └── migrations/
│
├── gen                               ← ALL generated code lives outside internal
│   ├── sqlc/
│   └── grpc/
│
├── proto
│   └── foo/api/v2/
│
├── scripts/
│
└── internal
    ├── feature                       ← every business feature, fully self-contained
    │   ├── auth
    │   │   ├── model.go
    │   │   ├── repository.go         ← interface + local cross-feature interfaces
    │   │   ├── repository_impl.go    ← sqlc wrapper + struct mapping
    │   │   ├── usecase.go            ← interface + Input structs
    │   │   ├── usecase_impl.go       ← business logic
    │   │   ├── handler_http.go       ← public HTTP: Request/Response structs, RBAC, mapping
    │   │   ├── handler_http_admin.go ← admin HTTP handlers (if needed)
    │   │   ├── handler_grpc.go       ← gRPC handler + proto mapping
    │   │   ├── task.go               ← task constants, payload structs, enqueue helpers
    │   │   ├── task_handler.go       ← asynq job handler logic
    │   │   └── routes.go             ← route registration for HTTP
    │   │
    │   ├── user
    │   │   └── (same structure as auth)
    │   │
    │   └── project
    │       └── (same structure)
    │
    ├── pkg                           ← shared utilities, zero domain knowledge
    │   ├── respond/                  ← HTTP response helpers
    │   ├── apierror/                 ← typed API error definitions
    │   ├── validate/                 ← validation helpers
    │   ├── utils/                    ← generic helpers (pagination, ptr, etc.)
    │   └── security
    │       ├── jwt/
    │       └── pwd/
    │
    ├── infra                         ← renamed from service — pure infrastructure clients
    │   ├── postgres
    │   │   └── client.go             ← *sql.DB or *pgxpool setup only
    │   ├── redis
    │   │   └── client.go
    │   └── logger
    │       └── logger.go
    │
    ├── middleware                     ← shared middleware, applies to HTTP and gRPC
    │   ├── auth.go
    │   └── rbac.go
    │
    ├── transport                      ← thin server wiring, no business logic
    │   ├── http
    │   │   └── server.go             ← gin setup, global middleware, route registration
    │   ├── grpc
    │   │   ├── server.go
    │   │   └── interceptor/
    │   │       └── auth.go
    │   ├── moderator                  ← admin-only HTTP surface (Option B, if needed)
    │   │   └── server.go
    │   └── worker
    │       └── server.go             ← asynq.ServeMux registration
    │
    ├── bootstrap
    │   ├── wire.go                   ← DI root: constructs everything, imports all features
    │   ├── server.go                 ← starts all servers + graceful shutdown
    │   ├── config.go                 ← loads + validates env config
    │   ├── migrate.go                ← runs DB migrations before servers start
    │   └── health.go                 ← readiness/liveness probe setup
    │
    └── settings
        └── config.go                 ← Config struct definition
```

---

## 3. Layer Responsibilities

### The Dependency Rule

```
feature/*           imports → pkg/, infra/, gen/, settings/
pkg/*               imports → external libs only
infra/*             imports → external libs only
transport/*         imports → feature/* (handlers only), infra/
bootstrap/*         imports → everything (this is the only place allowed to)
```

Nothing imports `bootstrap/`. Features never import other features directly.

---

### What Each File Does

#### `feature/<n>/model.go`

- Domain structs with no `json`, `db`, or `binding` tags
- Domain-level constants and enums
- Domain errors (`ErrNotFound`, `ErrEmailTaken`)
- Pure Go — no framework imports

```go
package user

type User struct {
    ID    string
    Name  string
    Email string
    Role  Role
}

type Role string
const (
    RoleAdmin  Role = "admin"
    RoleMember Role = "member"
)

var (
    ErrNotFound   = errors.New("user not found")
    ErrEmailTaken = errors.New("email already taken")
)
```

---

#### `feature/<n>/repository.go`

- Own `Repository` interface
- **Local cross-feature interfaces** — the primary tool for breaking import cycles
- Minimal projection structs for cross-feature data (only fields actually needed)

```go
package user

type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Save(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}

// Cross-feature: user needs to check auth sessions but must NOT import /auth
// Define only the method slice needed — auth.Repository satisfies this implicitly
type sessionChecker interface {
    HasActiveSession(ctx context.Context, userID string) (bool, error)
}
```

---

#### `feature/<n>/repository_impl.go`

- Implements `Repository` interface
- Only job: wrap sqlc queries and map generated types → domain models
- No business logic whatsoever

```go
package user

type repository struct{ q *sqlc.Queries }

func NewRepository(q *sqlc.Queries) Repository { return &repository{q: q} }

func (r *repository) FindByID(ctx context.Context, id string) (*User, error) {
    row, err := r.q.GetUser(ctx, id)
    if err == sql.ErrNoRows { return nil, ErrNotFound }
    if err != nil { return nil, err }
    return &User{ID: row.ID, Name: row.Name, Email: row.Email}, nil
}
```

---

#### `feature/<n>/usecase.go`

- `Usecase` interface
- **Input structs** — business-layer inputs, no `json` or `binding` tags
- These are NOT HTTP request models. The handler maps HTTP → Input.

```go
package user

type CreateUserInput struct {
    Name  string
    Email string
    Role  Role
}

type Usecase interface {
    GetUser(ctx context.Context, id string) (*User, error)
    CreateUser(ctx context.Context, input CreateUserInput) (*User, error)
    DeleteUser(ctx context.Context, id string) error
}
```

---

#### `feature/<n>/usecase_impl.go`

- Implements `Usecase` interface
- All business logic lives here
- Receives cross-feature dependencies via local interfaces — never imports another feature

```go
package user

type usecase struct {
    repo           Repository
    sessionChecker sessionChecker  // local interface, injected — no /auth import needed
    asynqClient    *asynq.Client
}

func NewUsecase(repo Repository, sc sessionChecker, client *asynq.Client) Usecase {
    return &usecase{repo: repo, sessionChecker: sc, asynqClient: client}
}

func (u *usecase) DeleteUser(ctx context.Context, id string) error {
    active, err := u.sessionChecker.HasActiveSession(ctx, id)
    if err != nil { return err }
    if active { return ErrUnauthorized }
    return u.repo.Delete(ctx, id)
}
```

---

#### `feature/<n>/handler_http.go`

- HTTP Request and Response structs with `json`/`binding` tags (only file that has these)
- Bind, validate, RBAC check — then map `Request` → `Input` → call usecase → map result → `Response`
- No business logic

```go
package user

type createUserRequest struct {
    Name  string `json:"name"  binding:"required,min=2"`
    Email string `json:"email" binding:"required,email"`
    Role  string `json:"role"  binding:"required,oneof=admin member"`
}

type userResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type HTTPHandler struct{ usecase Usecase }

func NewHTTPHandler(uc Usecase) *HTTPHandler { return &HTTPHandler{usecase: uc} }

func (h *HTTPHandler) CreateUser(c *gin.Context) {
    var req createUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, apierror.BadRequest(err))
        return
    }
    if c.GetString("role") != string(RoleAdmin) {
        c.JSON(http.StatusForbidden, apierror.Forbidden())
        return
    }
    user, err := h.usecase.CreateUser(c.Request.Context(), CreateUserInput{
        Name: req.Name, Email: req.Email, Role: Role(req.Role),
    })
    if err != nil {
        switch err {
        case ErrEmailTaken: c.JSON(http.StatusConflict, apierror.Conflict(err))
        default: c.JSON(http.StatusInternalServerError, apierror.Internal())
        }
        return
    }
    c.JSON(http.StatusCreated, toResponse(user))
}

func toResponse(u *User) userResponse {
    return userResponse{ID: u.ID, Name: u.Name, Email: u.Email}
}
```

---

#### `feature/<n>/handler_grpc.go`

- Same usecase, different transport
- Maps proto request → usecase Input, maps domain model → proto response

```go
package user

type GRPCHandler struct {
    pb.UnimplementedUserServiceServer
    usecase Usecase
}

func NewGRPCHandler(uc Usecase) *GRPCHandler { return &GRPCHandler{usecase: uc} }

func (h *GRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.usecase.GetUser(ctx, req.Id)
    if err != nil {
        switch err {
        case ErrNotFound: return nil, status.Error(codes.NotFound, err.Error())
        default: return nil, status.Error(codes.Internal, "internal error")
        }
    }
    return &pb.GetUserResponse{Id: user.ID, Name: user.Name, Email: user.Email}, nil
}
```

---

#### `feature/<n>/task.go`

- Task name constants — single source of truth
- Payload structs (no `json` tags needed beyond encoding)
- Enqueue helper functions — keeps `asynq.NewTask` calls out of business logic

```go
package user

const (
    TaskSendWelcomeEmail   = "user:send_welcome_email"
    TaskSyncToAnalytics    = "user:sync_to_analytics"
)

type SendWelcomeEmailPayload struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}

func NewSendWelcomeEmailTask(p SendWelcomeEmailPayload) (*asynq.Task, error) {
    b, err := json.Marshal(p)
    if err != nil { return nil, err }
    return asynq.NewTask(TaskSendWelcomeEmail, b), nil
}
```

---

#### `feature/<n>/task_handler.go`

- Handles incoming asynq jobs
- Same pattern as HTTP handler: decode payload → call usecase → done

```go
package user

type TaskHandler struct{ usecase Usecase }

func NewTaskHandler(uc Usecase) *TaskHandler { return &TaskHandler{usecase: uc} }

func (h *TaskHandler) HandleSendWelcomeEmail(ctx context.Context, t *asynq.Task) error {
    var p SendWelcomeEmailPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil { return err }
    return h.usecase.SendWelcomeEmail(ctx, p.UserID, p.Email)
}
```

---

#### `feature/<n>/routes.go`

- Route group setup only
- Attaches middleware to route groups
- Calls handler methods

```go
package user

func RegisterRoutes(r *gin.RouterGroup, h *HTTPHandler, adminH *HTTPAdminHandler) {
    users := r.Group("/users").Use(middleware.Auth())
    users.GET("/:id", h.GetUser)
    users.POST("",    h.CreateUser)

    admin := r.Group("/admin/users").Use(middleware.Auth(), middleware.RequireRole("admin"))
    admin.GET("",           adminH.ListUsers)
    admin.DELETE("/:id",    adminH.DeleteUser)
}
```

---

## 4. Cycle Import Prevention

### The Root Cause

```
user/usecase_impl.go  imports  /project
project/usecase_impl.go  imports  /user
→ compile error: import cycle not allowed
```

### The Fix: Local Interfaces

Go's implicit interface satisfaction means **you never need to import a package just to use one of its methods**.
Define the minimal interface you need, locally, where you need it. The concrete type satisfies it automatically.

```
❌ WRONG
// user/usecase_impl.go
import "myapp/internal/feature/project"   ← direct import causes cycle

✅ CORRECT
// user/repository.go — define locally, import nothing
type projectReader interface {
    FindByOwnerID(ctx context.Context, ownerID string) ([]*OwnedProject, error)
}
type OwnedProject struct{ ID, Name string }  // only fields user needs

// user/usecase_impl.go — inject via constructor
type usecase struct {
    repo         Repository
    projectRepo  projectReader   // satisfied implicitly by project.Repository
}
```

### Wiring Without Cycles — in `bootstrap/wire.go`

```go
userRepo    := user.NewRepository(q)
projectRepo := project.NewRepository(q)

// user.NewUsecase accepts a projectReader interface
// project.Repository satisfies it — Go checks this at compile time, no import needed in /user
userUsecase    := user.NewUsecase(userRepo, projectRepo)
projectUsecase := project.NewUsecase(projectRepo, userRepo)
```

### Dependency Direction Summary

```
feature/user     →  pkg/, infra/, gen/sqlc, gen/grpc, settings/
feature/project  →  pkg/, infra/, gen/sqlc, gen/grpc, settings/
transport/*      →  feature/* (handler constructors only)
bootstrap/       →  everything
```

`bootstrap` is the only node that points to all features. Since nothing imports `bootstrap`, there is no way for a cycle to form.

---

## 5. Feature Package — Complete Anatomy

```
feature/user/
  model.go              domain structs, enums, domain errors    — no tags
  repository.go         Repository interface                     — no tags
                        local cross-feature interfaces           — no tags
  repository_impl.go    sqlc wrapper + struct mapping            — db tags ok
  usecase.go            Usecase interface + Input structs        — no tags
  usecase_impl.go       business logic implementation            — no tags
  handler_http.go       Request/Response structs, HTTP handler   — json + binding tags
  handler_http_admin.go admin-only HTTP handlers                 — json + binding tags
  handler_grpc.go       gRPC handler + proto mapping             — no tags
  task.go               task constants, payloads, enqueue helpers
  task_handler.go       asynq job handler
  routes.go             HTTP route registration
```

### Model Ownership Rule

A model belongs in a feature package when **it has no meaning without its parent**.

```
user_setting   → feature/user/model.go     ✅ (no meaning without User)
user_address   → feature/user/model.go     ✅
project_member → feature/project/model.go  ✅
notification   → feature/notification/     ✅ (user, project, billing all fire these)
audit_log      → feature/auditlog/         ✅ (cross-cutting, no single owner)
```

---

## 6. Shared Packages (`pkg/`)

Lives at `internal/pkg/`. Rule: **if it has zero knowledge of your domain models, it goes here**.

```
internal/pkg/
  respond/          HTTP response helpers
  apierror/         typed API errors with codes and messages
  validate/         validation wrappers
  utils/            pagination, pointer helpers, generic tools
  security/
    jwt/            token generation + parsing
    pwd/            password hashing + comparison
  subject/          context key helpers (current user, request ID, etc.)
```

```go
// ✅ belongs in pkg/ — no domain knowledge
// pkg/respond/respond.go
func JSON(c *gin.Context, status int, data any) {
    c.JSON(status, data)
}
func Error(c *gin.Context, err *apierror.APIError) {
    c.JSON(err.HTTPStatus, err)
}

// ✅ belongs in pkg/
// pkg/apierror/apierror.go
type APIError struct {
    Code       string `json:"code"`
    Message    string `json:"message"`
    HTTPStatus int    `json:"-"`
}
func NotFound(msg string) *APIError { ... }
func BadRequest(err error) *APIError { ... }

// ❌ does NOT belong in pkg/ — knows about User
func RespondWithUser(c *gin.Context, u *user.User) { ... }
// this belongs in feature/user/handler_http.go as toResponse()
```

---

## 7. Moderator / Admin APIs

Admin/staff APIs are not a separate feature — they are the same feature with elevated permissions and sometimes a different response shape.

### Option A — Extra Handler File (Recommended)

Use when admin endpoints share most logic with the public usecase.

```
feature/user/
  handler_http.go        ← public endpoints
  handler_http_admin.go  ← admin endpoints, same usecase
```

```go
// handler_http_admin.go
type HTTPAdminHandler struct{ usecase Usecase }

func NewHTTPAdminHandler(uc Usecase) *HTTPAdminHandler { ... }

func (h *HTTPAdminHandler) ListUsers(c *gin.Context) {
    // admin can see all users, more fields, pagination
    users, err := h.usecase.ListAll(c.Request.Context(), ...)
    ...
}
```

Routes attach different middleware for the admin group:

```go
// routes.go
func RegisterRoutes(r *gin.RouterGroup, h *HTTPHandler, adminH *HTTPAdminHandler) {
    r.Group("/users").Use(middleware.Auth()).
        GET("/:id", h.GetUser)

    r.Group("/admin/users").
        Use(middleware.Auth(), middleware.RequireRole("admin", "staff")).
        GET("",           adminH.ListUsers).
        DELETE("/:id",    adminH.DeleteUser).
        PUT("/:id/role",  adminH.UpdateRole)
}
```

### Option B — Separate Transport (For Large Admin Surfaces)

Use only when the admin API is a completely different application (separate port, separate auth scheme, fundamentally different flows).

```
transport/
  http/server.go          ← public API
  moderator/server.go     ← admin-only HTTP surface, its own router + middleware stack
```

`transport/moderator/server.go` imports feature handlers just like `transport/http/server.go` does — no new feature packages needed.

---

## 8. Asynq Task Workers

Each feature owns its async tasks entirely. The worker transport just wires them up.

### Feature Owns Three Concerns

**`task.go`** — task names, payload structs, enqueue helpers

```go
package user

const TaskSendWelcomeEmail = "user:send_welcome_email"

type SendWelcomeEmailPayload struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}

func NewSendWelcomeEmailTask(p SendWelcomeEmailPayload) (*asynq.Task, error) {
    b, _ := json.Marshal(p)
    return asynq.NewTask(TaskSendWelcomeEmail, b), nil
}
```

**`task_handler.go`** — decode payload, call usecase

```go
package user

type TaskHandler struct{ usecase Usecase }

func NewTaskHandler(uc Usecase) *TaskHandler { return &TaskHandler{usecase: uc} }

func (h *TaskHandler) HandleSendWelcomeEmail(ctx context.Context, t *asynq.Task) error {
    var p SendWelcomeEmailPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil { return err }
    return h.usecase.SendWelcomeEmail(ctx, p.UserID, p.Email)
}
```

**Dispatching from usecase** — enqueue after business logic completes

```go
// usecase_impl.go
func (u *usecase) CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // ... create user

    task, _ := NewSendWelcomeEmailTask(SendWelcomeEmailPayload{
        UserID: user.ID, Email: user.Email,
    })
    u.asynqClient.Enqueue(task)

    return user, nil
}
```

### Worker Transport — Registration Only

```go
// transport/worker/server.go
package worker

func New(
    userTask *user.TaskHandler,
    authTask *auth.TaskHandler,
) *asynq.ServeMux {
    mux := asynq.NewServeMux()

    mux.HandleFunc(user.TaskSendWelcomeEmail, userTask.HandleSendWelcomeEmail)
    mux.HandleFunc(user.TaskSyncToAnalytics,  userTask.HandleSyncUser)
    mux.HandleFunc(auth.TaskRevokeExpiredTokens, authTask.HandleRevokeExpired)

    return mux
}
```

---

## 9. Transport Layer

Every transport file does one thing: **receive constructed handlers and wire them into a server**. No business logic, no SQL, no domain models.

```go
// transport/http/server.go
package http

type Server struct{ engine *gin.Engine }

func New(
    cfg *settings.Config,
    userHandler   *user.HTTPHandler,
    userAdminH    *user.HTTPAdminHandler,
    authHandler   *auth.HTTPHandler,
) *Server {
    r := gin.New()
    r.Use(gin.Recovery(), middleware.Logger())

    api := r.Group("/api/v1")
    user.RegisterRoutes(api, userHandler, userAdminH)
    auth.RegisterRoutes(api, authHandler)

    return &Server{engine: r}
}

func (s *Server) Start(addr string) error { return s.engine.Run(addr) }
func (s *Server) Shutdown() { /* graceful */ }
```

```go
// transport/grpc/server.go
package grpc

func New(
    cfg *settings.Config,
    userGRPC *user.GRPCHandler,
) *grpc.Server {
    s := grpc.NewServer(
        grpc.ChainUnaryInterceptor(interceptor.Auth()),
    )
    pb.RegisterUserServiceServer(s, userGRPC)
    return s
}
```

---

## 10. Bootstrap Layer

Everything about **starting and stopping the application**. Nothing else.

### `bootstrap/config.go` — load and validate config

```go
package bootstrap

func LoadConfig() (*settings.Config, error) {
    var cfg settings.Config
    if err := envconfig.Process("APP", &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### `bootstrap/wire.go` — DI root, constructs the entire object graph

```go
package bootstrap

func Init(cfg *settings.Config) (*App, error) {
    // infra
    db, _ := sql.Open("postgres", cfg.DSN)
    q      := sqlc.New(db)
    rdb   := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})

    // repositories
    authRepo := auth.NewRepository(q)
    userRepo := user.NewRepository(q)

    // usecases — cross-feature deps injected as local interfaces, no cycles
    authUsecase := auth.NewUsecase(authRepo, userRepo)
    userUsecase := user.NewUsecase(userRepo, authRepo, asynqClient)

    // HTTP handlers
    authHTTP  := auth.NewHTTPHandler(authUsecase)
    userHTTP  := user.NewHTTPHandler(userUsecase)
    userAdmin := user.NewHTTPAdminHandler(userUsecase)

    // gRPC handlers
    userGRPC := user.NewGRPCHandler(userUsecase)

    // task handlers
    userTask := user.NewTaskHandler(userUsecase)
    authTask := auth.NewTaskHandler(authUsecase)

    // servers
    httpServer   := transportHTTP.New(cfg, userHTTP, userAdmin, authHTTP)
    grpcServer   := transportGRPC.New(cfg, userGRPC)
    workerServer := transportWorker.New(userTask, authTask)

    return &App{
        HTTP:   httpServer,
        GRPC:   grpcServer,
        Worker: workerServer,
    }, nil
}
```

### `bootstrap/server.go` — starts all servers, handles graceful shutdown

```go
package bootstrap

func Run(app *App) error {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error { return app.HTTP.Start(":8080") })
    g.Go(func() error { return app.GRPC.Start(":9090") })
    g.Go(func() error { return app.Worker.Start() })

    g.Go(func() error {
        <-ctx.Done()
        app.HTTP.Shutdown()
        app.GRPC.Stop()
        app.Worker.Stop()
        return nil
    })

    return g.Wait()
}
```

### `bootstrap/migrate.go` — run migrations before servers start

```go
package bootstrap

func RunMigrations(db *sql.DB) error {
    m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
    if err != nil { return err }
    if err := m.Up(); err != nil && err != migrate.ErrNoChange { return err }
    return nil
}
```

### `bootstrap/health.go` — readiness and liveness probes

```go
package bootstrap

func RegisterHealthRoutes(r *gin.Engine, db *sql.DB) {
    r.GET("/healthz",  handleLiveness)
    r.GET("/readyz",   handleReadiness(db))
}
```

### `cmd/main.go` — trivially small

```go
package main

func main() {
    cfg, err := bootstrap.LoadConfig()
    if err != nil { log.Fatal("config:", err) }

    if err := bootstrap.RunMigrations(db); err != nil { log.Fatal("migrate:", err) }

    app, err := bootstrap.Init(cfg)
    if err != nil { log.Fatal("init:", err) }

    if err := bootstrap.Run(app); err != nil { log.Fatal("run:", err) }
}
```

---

## 11. sqlc and the Repository Layer

**Keep the repository layer even when using sqlc.** It stays thin but earns its place.

### Why Not Remove It

Calling sqlc directly from usecases couples business logic to generated types:

```go
// ❌ sqlc leaking into usecase
func (u *usecase) GetUser(ctx context.Context, id string) (*User, error) {
    row, err := u.q.GetUser(ctx, id)  // now usecase knows about sqlc internals
    // ...
}
```

This means: mocking requires mocking sqlc, adding a cache layer requires touching usecases, switching storage requires rewriting usecases.

### What the Repo Layer Actually Does With sqlc

```go
// ✅ repository_impl.go — its only job is mapping sqlc ↔ domain
func (r *repository) FindByID(ctx context.Context, id string) (*User, error) {
    row, err := r.q.GetUser(ctx, id)
    if err == sql.ErrNoRows { return nil, ErrNotFound }
    if err != nil { return nil, err }
    return &User{                      // sqlc type → domain model
        ID:    row.ID,
        Name:  row.Name,
        Email: row.Email,
    }, nil
}
```

It is thin, but it buys you:

| Benefit             | How                                                          |
| ------------------- | ------------------------------------------------------------ |
| Testable usecases   | Mock the `Repository` interface — no sqlc knowledge needed   |
| Transparent caching | Wrap the repo with a Redis decorator, usecase doesn't change |
| Storage flexibility | Swap sqlc for pgx or another ORM — usecase is untouched      |
| Clean domain model  | sqlc types with `db` tags never escape to business logic     |

---

## 12. Migration Steps

Each step compiles and runs independently. Never do a big-bang rewrite.

### Step 1 — Create skeleton directories

```bash
mkdir -p internal/feature/auth internal/feature/user
mkdir -p internal/infra/postgres internal/infra/redis internal/infra/logger
mkdir -p internal/transport/http internal/transport/grpc
mkdir -p internal/transport/worker internal/transport/moderator
mkdir -p internal/pkg/respond internal/pkg/apierror
mkdir -p internal/pkg/security/jwt internal/pkg/security/pwd
mkdir -p internal/bootstrap
mkdir -p gen/sqlc gen/grpc
```

### Step 2 — Move infrastructure (`service/` → `infra/`)

```bash
mv internal/service/postgres  internal/infra/postgres
mv internal/service/redis     internal/infra/redis
mv internal/service/logger    internal/infra/logger
```

```bash
grep -rl "internal/service/" . | xargs sed -i 's|internal/service/|internal/infra/|g'
```

✅ Compile.

### Step 3 — Move generated code outside `internal/`

```bash
mv internal/gen/sqlc  gen/sqlc
mv internal/gen/grpc  gen/grpc
```

Update sqlc.yaml and buf.gen.yaml output paths, regenerate:

```bash
sqlc generate && buf generate
```

```bash
grep -rl "internal/gen/" . | xargs sed -i 's|internal/gen/|gen/|g'
```

✅ Compile.

### Step 4 — Move shared packages to `pkg/`

```bash
mv internal/api/http/respond    internal/pkg/respond
mv internal/api/http/apierror   internal/pkg/apierror
# repeat for validate, utils, security/jwt, security/pwd, subject
```

```bash
grep -rl "api/http/respond" . | xargs sed -i 's|api/http/respond|pkg/respond|g'
```

✅ Compile.

### Step 5 — Extract domain models from `api/http/domain/`

Move each type to the feature it belongs to:

```
internal/api/http/domain/user.go  →  internal/feature/user/model.go
internal/api/http/domain/auth.go  →  internal/feature/auth/model.go
```

Update all imports pointing to the old domain path.

✅ Compile.

### Step 6 — Migrate one feature fully (start with `auth`)

Inside `internal/feature/auth/`:

- Create `repository.go` with the interface
- Create `repository_impl.go` wrapping sqlc
- Create `usecase.go` with the interface and Input structs
- Move business logic into `usecase_impl.go`
- Copy HTTP handler to `handler_http.go`, strip domain-level types back to model.go
- Copy gRPC handler to `handler_grpc.go`
- Create `task.go` and `task_handler.go`
- Replace any cross-feature direct imports with local interface definitions

✅ Compile.

### Step 7 — Migrate remaining features

Repeat Step 6 for `user`, `project`, etc. One at a time, compile after each.

### Step 8 — Create transport layer

Move server setup out of wherever it lives now:

```
gin initialization     →  transport/http/server.go
gRPC server setup      →  transport/grpc/server.go
asynq server + mux     →  transport/worker/server.go
moderator routes       →  transport/moderator/server.go (if Option B)
```

### Step 9 — Create `bootstrap/wire.go`

Move all DI wiring from `main.go` or existing bootstrap into `wire.go`.
Create `server.go` with the errgroup graceful shutdown.

### Step 10 — Clean up old structure

```bash
rm -rf internal/api/http/domain
rm -rf internal/api/http/v1
rm -rf internal/api/grpc/v2
rm -rf internal/service
rm -rf internal/gen
```

---

## 13. Rules Cheatsheet

```
□ No feature package imports another feature package directly
□ Cross-feature deps are always local interfaces in repository.go
□ model.go has no json, db, or binding tags
□ Input structs (usecase.go) have no tags — they are business inputs not HTTP models
□ Request/Response structs with json/binding tags only exist in handler_http.go
□ repository_impl.go does struct mapping only — no business logic
□ usecase_impl.go has no HTTP, gin, grpc, or asynq decode logic
□ task_handler.go only decodes payload and calls usecase
□ bootstrap/wire.go is the only file that imports multiple feature packages
□ transport/ files call constructors only — no business logic
□ pkg/ packages have zero imports from feature/
□ infra/ packages have zero imports from feature/ or pkg/domain types
□ gen/ lives outside internal/
□ cmd/main.go is under 20 lines
```

### Where Does X Live — Quick Reference

| X                             | File                              | Has tags      |
| ----------------------------- | --------------------------------- | ------------- |
| `User{}` struct               | `feature/user/model.go`           | none          |
| `UserSetting{}` struct        | `feature/user/model.go`           | none          |
| `ErrNotFound`                 | `feature/user/model.go`           | —             |
| `Repository` interface        | `feature/user/repository.go`      | none          |
| Cross-feature local interface | `feature/user/repository.go`      | none          |
| `Usecase` interface           | `feature/user/usecase.go`         | none          |
| `CreateUserInput{}`           | `feature/user/usecase.go`         | none          |
| SQL queries                   | `feature/user/repository_impl.go` | db            |
| Business logic                | `feature/user/usecase_impl.go`    | none          |
| `CreateUserRequest{}`         | `feature/user/handler_http.go`    | json, binding |
| `UserResponse{}`              | `feature/user/handler_http.go`    | json          |
| RBAC checks                   | `feature/user/handler_http.go`    | —             |
| Proto ↔ domain mapping        | `feature/user/handler_grpc.go`    | —             |
| Task name constants           | `feature/user/task.go`            | —             |
| Asynq job logic               | `feature/user/task_handler.go`    | —             |
| Route registration            | `feature/user/routes.go`          | —             |
| Server setup                  | `transport/*/server.go`           | —             |
| All DI wiring                 | `bootstrap/wire.go`               | —             |
| Graceful shutdown             | `bootstrap/server.go`             | —             |
| JWT helpers                   | `pkg/security/jwt/`               | —             |
| Password hashing              | `pkg/security/pwd/`               | —             |
| HTTP response helpers         | `pkg/respond/`                    | —             |
| Typed API errors              | `pkg/apierror/`                   | —             |

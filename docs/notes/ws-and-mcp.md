# WebSocket & MCP Transport Best Practices

## Table of Contents

1. [Updated Transport Structure](#1-updated-transport-structure)
2. [WebSocket — Library Choice](#2-websocket--library-choice)
3. [WebSocket — Architecture](#3-websocket--architecture)
4. [WebSocket — Hub & Room](#4-websocket--hub--room)
5. [WebSocket — Redis Pub/Sub for Multi-Instance](#5-websocket--redis-pubsub-for-multi-instance)
6. [WebSocket — Feature Integration](#6-websocket--feature-integration)
7. [WebSocket — Transport Server](#7-websocket--transport-server)
8. [MCP — Library Choice](#8-mcp--library-choice)
9. [MCP — Route Detection with Chi + Swaggo](#9-mcp--route-detection-with-chi--swaggo)
10. [MCP — Transport Server](#10-mcp--transport-server)
11. [MCP — Feature Integration](#11-mcp--feature-integration)
12. [Rules Cheatsheet](#12-rules-cheatsheet)

---

## 1. Updated Transport Structure

```
internal/
  transport/
    http/
      server.go
    grpc/
      server.go
      interceptor/
    worker/
      server.go
    ws/                         ← WebSocket
      server.go                 ← upgrades HTTP → WS, manages Hub lifecycle
      hub.go                    ← central connection registry + broadcast engine
      room.go                   ← scoped broadcast (per-entity grouping)
      client.go                 ← single connection lifecycle (read/write pumps)
    mcp/                        ← Model Context Protocol
      server.go                 ← mcp server setup, tool registration
      tools.go                  ← tool definitions grouped by feature

  infra/
    postgres/
    redis/
      client.go                 ← shared *redis.Client
      pubsub.go                 ← pub/sub helpers used by ws Hub
```

`transport/ws/client.go` is NOT the same as `infra/redis/client.go`.

- `transport/ws/client.go` — represents a single WebSocket connection (a browser tab)
- `infra/redis/client.go` — the Redis infrastructure connection

---

## 2. WebSocket — Library Choice

Use **`gorilla/websocket`** for the WebSocket protocol layer.

```bash
go get github.com/gorilla/websocket
go get github.com/redis/go-redis/v9   # already in your infra
```

**Why gorilla/websocket:**

- Most battle-tested Go WebSocket library
- Fine-grained control over upgrader config (buffer sizes, origin checks)
- Explicit read/write separation — maps cleanly to the pump pattern
- Widely used, excellent documentation

**Why not `nhooyr.io/websocket`:**

- More ergonomic API but less control over low-level behaviour
- Gorilla is the de-facto standard for production Go servers with custom hub logic

**For Redis pub/sub across multiple server instances:**

- Use `go-redis` channels — the same client you already have in `infra/redis/`
- No extra library needed

---

## 3. WebSocket — Architecture

```
Browser Tab
    │  WS connection
    ▼
transport/ws/Client      ← one per connection: read pump + write pump goroutines
    │
    ▼
transport/ws/Room        ← optional grouping (e.g. projectID, chatID)
    │
    ▼
transport/ws/Hub         ← central registry: register/unregister clients, broadcast
    │
    ├── local broadcast  ← sends to clients on THIS server instance
    │
    └── Redis pub/sub    ← publishes to OTHER server instances
```

### Why Server/Client Structure

Yes — always use the server/client pattern (also called Hub/Client). Never manage connections directly in handlers.

**Server (Hub)** at `transport/ws/`:

- Owns the global connection registry
- Knows which clients are connected to which rooms
- Receives broadcast requests and fans out to the right clients
- Subscribes to Redis for cross-instance messages

**Client** at `transport/ws/client.go`:

- Wraps a single `*websocket.Conn`
- Owns two goroutines: `readPump` and `writePump`
- Sends incoming messages to the Hub
- Receives outgoing messages from the Hub via a buffered channel

This separation means your feature usecases never touch WebSocket connections directly. They just publish an event — the transport layer handles delivery.

---

## 4. WebSocket — Hub & Room

### `transport/ws/client.go`

```go
// transport/ws/client.go
package ws

import (
    "log/slog"
    "time"

    "github.com/gorilla/websocket"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 512
)

// Client represents a single WebSocket connection.
type Client struct {
    hub    *Hub
    conn   *websocket.Conn
    send   chan []byte  // buffered channel of outbound messages
    UserID string
    RoomID string       // empty string means global (no room)
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, roomID string) *Client {
    return &Client{
        hub:    hub,
        conn:   conn,
        send:   make(chan []byte, 256),
        UserID: userID,
        RoomID: roomID,
    }
}

// Start registers the client and launches both pumps.
// Call this after upgrading the HTTP connection.
func (c *Client) Start() {
    c.hub.register <- c
    go c.writePump()
    go c.readPump()
}

// readPump reads messages from the WebSocket connection.
// One goroutine per client — blocks until connection closes.
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, msg, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure,
            ) {
                slog.Error("ws: unexpected close", "error", err, "userID", c.UserID)
            }
            break
        }
        // forward inbound message to hub for routing
        c.hub.inbound <- InboundMessage{Client: c, Data: msg}
    }
}

// writePump writes messages from the send channel to the WebSocket connection.
// One goroutine per client — blocks until send channel closes or connection drops.
func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case msg, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                slog.Error("ws: write error", "error", err, "userID", c.UserID)
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### `transport/ws/room.go`

```go
// transport/ws/room.go
package ws

// Room is a scoped group of clients (e.g. per projectID, per chatID).
// The Hub manages rooms — rooms themselves are just client sets.
type Room struct {
    ID      string
    clients map[*Client]struct{}
}

func newRoom(id string) *Room {
    return &Room{
        ID:      id,
        clients: make(map[*Client]struct{}),
    }
}

func (r *Room) add(c *Client) {
    r.clients[c] = struct{}{}
}

func (r *Room) remove(c *Client) {
    delete(r.clients, c)
}

func (r *Room) isEmpty() bool {
    return len(r.clients) == 0
}
```

### `transport/ws/hub.go`

```go
// transport/ws/hub.go
package ws

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/redis/go-redis/v9"
)

// OutboundMessage is what the hub broadcasts to clients.
type OutboundMessage struct {
    RoomID  string // empty = broadcast to all connected clients
    UserID  string // empty = broadcast to all in room; set to target a specific user
    Payload []byte
}

// InboundMessage is a message received from a client.
type InboundMessage struct {
    Client *Client
    Data   []byte
}

// Hub is the central connection registry and broadcast engine.
// There is ONE hub per server process.
type Hub struct {
    // client lifecycle
    register   chan *Client
    unregister chan *Client

    // rooms: roomID → Room
    rooms map[string]*Room

    // global clients (no room)
    globals map[*Client]struct{}

    // inbound messages from clients
    inbound chan InboundMessage

    // outbound messages from features/usecases
    Broadcast chan OutboundMessage

    // redis for cross-instance broadcasting
    rdb       *redis.Client
    redisChan string
}

func NewHub(rdb *redis.Client) *Hub {
    return &Hub{
        register:   make(chan *Client),
        unregister: make(chan *Client),
        rooms:      make(map[string]*Room),
        globals:    make(map[*Client]struct{}),
        inbound:    make(chan InboundMessage, 256),
        Broadcast:  make(chan OutboundMessage, 256),
        rdb:        rdb,
        redisChan:  "ws:broadcast",
    }
}

// Run starts the hub event loop. Call in a goroutine.
func (h *Hub) Run(ctx context.Context) {
    // subscribe to redis for cross-instance messages
    sub := h.rdb.Subscribe(ctx, h.redisChan)
    defer sub.Close()

    redisMsgs := sub.Channel()

    for {
        select {
        case <-ctx.Done():
            return

        case client := <-h.register:
            h.addClient(client)

        case client := <-h.unregister:
            h.removeClient(client)

        case msg := <-h.Broadcast:
            // publish to redis so other instances also broadcast
            h.publishToRedis(ctx, msg)
            // also deliver locally
            h.deliverLocally(msg)

        case redisMsg := <-redisMsgs:
            // message from another instance — deliver locally only
            var msg OutboundMessage
            if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
                slog.Error("ws: failed to unmarshal redis message", "error", err)
                continue
            }
            h.deliverLocally(msg)

        case inbound := <-h.inbound:
            // handle client → server messages if needed
            // most apps only need server → client; ignore or log inbound
            slog.Debug("ws: inbound message", "userID", inbound.Client.UserID)
        }
    }
}

func (h *Hub) addClient(c *Client) {
    if c.RoomID == "" {
        h.globals[c] = struct{}{}
        return
    }
    room, ok := h.rooms[c.RoomID]
    if !ok {
        room = newRoom(c.RoomID)
        h.rooms[c.RoomID] = room
    }
    room.add(c)
}

func (h *Hub) removeClient(c *Client) {
    close(c.send)
    if c.RoomID == "" {
        delete(h.globals, c)
        return
    }
    if room, ok := h.rooms[c.RoomID]; ok {
        room.remove(c)
        if room.isEmpty() {
            delete(h.rooms, c.RoomID)
        }
    }
}

func (h *Hub) deliverLocally(msg OutboundMessage) {
    var targets []*Client

    if msg.RoomID == "" {
        // broadcast to all global clients
        for c := range h.globals {
            targets = append(targets, c)
        }
    } else if room, ok := h.rooms[msg.RoomID]; ok {
        for c := range room.clients {
            targets = append(targets, c)
        }
    }

    for _, c := range targets {
        // if UserID is set, only send to that specific user
        if msg.UserID != "" && c.UserID != msg.UserID {
            continue
        }
        select {
        case c.send <- msg.Payload:
        default:
            // client send buffer full — unregister it
            h.removeClient(c)
        }
    }
}

func (h *Hub) publishToRedis(ctx context.Context, msg OutboundMessage) {
    b, err := json.Marshal(msg)
    if err != nil {
        slog.Error("ws: failed to marshal broadcast message", "error", err)
        return
    }
    if err := h.rdb.Publish(ctx, h.redisChan, b).Err(); err != nil {
        slog.Error("ws: failed to publish to redis", "error", err)
    }
}
```

---

## 5. WebSocket — Redis Pub/Sub for Multi-Instance

When you run multiple server instances (horizontal scaling), a broadcast from instance A must reach clients connected to instance B. Redis pub/sub solves this.

```
Instance A                        Instance B
  Hub.Broadcast ──▶ Redis ──▶ Hub subscription
  hub.deliverLocally()             hub.deliverLocally()
  (clients on A get it)            (clients on B get it)
```

The Hub already handles this in `Run()`:

- On `Broadcast` channel: publish to Redis AND deliver locally
- On Redis subscription channel: deliver locally only (don't re-publish or you'll loop)

The `infra/redis/pubsub.go` helper wraps the channel name conventions:

```go
// infra/redis/pubsub.go
package redis

const (
    ChanWSBroadcast = "ws:broadcast"
    ChanWSRoom      = "ws:room:%s"    // format with roomID
)
```

---

## 6. WebSocket — Feature Integration

Features never import the `transport/ws` package. They communicate via a **`Publisher` interface** defined locally — the same local interface pattern used everywhere else.

### The Publisher Interface — Defined in the Feature

```go
// feature/notification/usecase.go
package notification

import "context"

// wsPublisher is the local interface — feature never imports transport/ws
type wsPublisher interface {
    Publish(ctx context.Context, roomID, userID string, payload []byte) error
}

type Usecase interface {
    SendToUser(ctx context.Context, userID string, msg Message) error
    SendToRoom(ctx context.Context, roomID string, msg Message) error
    BroadcastAll(ctx context.Context, msg Message) error
}
```

### The Usecase Implementation

```go
// feature/notification/usecase_impl.go
package notification

import (
    "context"
    "encoding/json"
    "fmt"
)

type usecase struct {
    repo      Repository
    publisher wsPublisher // injected from bootstrap — satisfied by Hub
}

func NewUsecase(repo Repository, pub wsPublisher) Usecase {
    return &usecase{repo: repo, publisher: pub}
}

func (u *usecase) SendToUser(ctx context.Context, userID string, msg Message) error {
    b, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("usecase.SendToUser marshal: %w", err)
    }
    // empty roomID = global; userID scopes it to one user
    return u.publisher.Publish(ctx, "", userID, b)
}

func (u *usecase) SendToRoom(ctx context.Context, roomID string, msg Message) error {
    b, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("usecase.SendToRoom marshal: %w", err)
    }
    return u.publisher.Publish(ctx, roomID, "", b)
}
```

### Hub Satisfies the Interface

The `Hub.Broadcast` channel needs a thin adapter to satisfy the `wsPublisher` interface:

```go
// transport/ws/publisher.go
package ws

import (
    "context"
)

// Publisher wraps Hub and implements any feature's wsPublisher interface.
// This is what bootstrap/wire.go passes into feature usecases.
type Publisher struct {
    hub *Hub
}

func NewPublisher(hub *Hub) *Publisher {
    return &Publisher{hub: hub}
}

func (p *Publisher) Publish(ctx context.Context, roomID, userID string, payload []byte) error {
    select {
    case p.hub.Broadcast <- OutboundMessage{
        RoomID:  roomID,
        UserID:  userID,
        Payload: payload,
    }:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### Wiring in `bootstrap/wire.go`

```go
// bootstrap/wire.go
rdb := redis.New(cfg.RedisAddr)

wsHub       := ws.NewHub(rdb)
wsPublisher := ws.NewPublisher(wsHub)

// inject publisher into any feature that needs real-time delivery
notifUsecase := notification.NewUsecase(notifRepo, wsPublisher)
userUsecase  := user.NewUsecase(userRepo, authRepo, wsPublisher, asynqClient)

wsServer := transportWS.New(cfg, wsHub, notifHTTP)
```

### Usage From Other Features

Any feature usecase that needs to push a real-time event just calls its own usecase:

```go
// feature/project/usecase_impl.go

// after a member is added to a project, notify them in real time
func (u *usecase) AddMember(ctx context.Context, input AddMemberInput) error {
    // ... business logic ...

    // dispatch notification — project usecase calls notification usecase
    // notification usecase handles the ws publish
    _ = u.notifier.SendToUser(ctx, input.UserID, notification.Message{
        Type:    "project.member_added",
        Payload: map[string]string{"project_id": input.ProjectID},
    })

    return nil
}
```

---

## 7. WebSocket — Transport Server

```go
// transport/ws/server.go
package ws

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/gorilla/websocket"

    "myapp/internal/settings"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // in production: check r.Header.Get("Origin") against cfg.CorsOrigin
        return true
    },
}

type Server struct {
    hub *Hub
    cfg *settings.Config
}

func New(cfg *settings.Config, hub *Hub) *Server {
    return &Server{hub: hub, cfg: cfg}
}

func (s *Server) RegisterRoutes(r chi.Router) {
    r.Get("/ws", s.handleUpgrade)
    r.Get("/ws/room/{roomID}", s.handleRoomUpgrade)
}

func (s *Server) handleUpgrade(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    userID := userIDFromCtx(r.Context()) // set by auth middleware
    client := NewClient(s.hub, conn, userID, "")
    client.Start()
}

func (s *Server) handleRoomUpgrade(w http.ResponseWriter, r *http.Request) {
    roomID := chi.URLParam(r, "roomID")
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    userID := userIDFromCtx(r.Context())
    client := NewClient(s.hub, conn, userID, roomID)
    client.Start()
}

// Start runs the hub event loop. Called from bootstrap/server.go in an errgroup.
func (s *Server) Start(ctx context.Context) error {
    s.hub.Run(ctx)
    return nil
}

func (s *Server) Stop() {
    // hub stops when ctx is cancelled — no explicit stop needed
}
```

In `bootstrap/server.go`:

```go
g.Go(func() error { return app.WS.Start(ctx) })
```

---

## 8. MCP — Library Choice

Use **`mark3labs/mcp-go`**.

```bash
go get github.com/mark3labs/mcp-go
```

**Why `mark3labs/mcp-go`:**

- Most complete Go implementation of the MCP spec
- Supports both `stdio` and `SSE` (HTTP) transport — SSE pairs naturally with chi
- Tools, resources, and prompts all supported
- Active maintenance, closest to the reference implementation

**Transport mode for your stack:** Use **SSE transport** (Server-Sent Events over HTTP). This integrates with chi naturally — MCP sits on a `/mcp` route group the same way your REST API does.

---

## 9. MCP — Route Detection with Chi + Swaggo

The key question: how does MCP know which tool to call for which action?

**The answer: you map explicitly.** MCP tools are not auto-detected from chi routes. Instead, each MCP tool is a named function that calls a feature usecase directly — the same usecase your HTTP handlers use.

However, **Swaggo's generated OpenAPI spec can drive the tool descriptions** so you never write the same documentation twice.

### The Strategy

```
Swaggo annotation   →  generates openapi.json
openapi.json        →  read by mcp/tools.go at startup
mcp/tools.go        →  registers MCP tools with descriptions from swagger
Tool execution      →  calls feature usecase directly
```

This means:

- Tool names and descriptions come from your swagger doc — single source of truth
- Tool logic calls the same usecases as your HTTP handlers — no duplication
- Adding a new HTTP endpoint + swagger annotation automatically enriches MCP

### `transport/mcp/server.go`

```go
// transport/mcp/server.go
package mcp

import (
    "context"
    "net/http"

    "github.com/mark3labs/mcp-go/server"
    "github.com/mark3labs/mcp-go/mcp"

    "myapp/internal/settings"
)

type Handlers struct {
    User    *UserTools
    Project *ProjectTools
    // add new feature tools here
}

type Server struct {
    mcp *server.MCPServer
    sse *server.SSEServer
    cfg *settings.Config
}

func New(cfg *settings.Config, h Handlers) *Server {
    s := server.NewMCPServer(
        "MyApp MCP",
        "1.0.0",
        server.WithToolCapabilities(true),
        server.WithResourceCapabilities(true, true),
    )

    // register all tools from all features
    h.User.Register(s)
    h.Project.Register(s)

    sse := server.NewSSEServer(s,
        server.WithBaseURL(cfg.MCPBaseURL),   // e.g. "https://api.example.com"
        server.WithBasePath("/mcp"),
    )

    return &Server{mcp: s, sse: sse, cfg: cfg}
}

// Handler returns an http.Handler to mount on chi.
// Mount at /mcp in transport/http/server.go.
func (s *Server) Handler() http.Handler {
    return s.sse
}

func (s *Server) Start(ctx context.Context) error {
    return s.sse.Start(ctx, s.cfg.MCPAddr) // standalone if needed
}
```

Mount in `transport/http/server.go`:

```go
// transport/http/server.go
func registerRoutes(r *chi.Mux, mw Middlewares, h Handlers, wsServer *ws.Server, mcpServer *mcp.Server) {
    // ... existing routes ...

    // MCP — mount the SSE handler
    r.Mount("/mcp", mcpServer.Handler())

    // WebSocket
    wsServer.RegisterRoutes(r)
}
```

---

## 10. MCP — Transport Server

### `transport/mcp/tools.go` — Tool Definitions Per Feature

Each feature gets its own `*Tools` struct in the `mcp` transport package. Tools call feature usecases directly.

```go
// transport/mcp/user_tools.go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"

    "myapp/internal/feature/user"
)

type UserTools struct {
    usecase user.Usecase
}

func NewUserTools(uc user.Usecase) *UserTools {
    return &UserTools{usecase: uc}
}

// Register adds all user-related tools to the MCP server.
func (t *UserTools) Register(s *server.MCPServer) {
    s.AddTool(t.getUser(), t.handleGetUser)
    s.AddTool(t.createUser(), t.handleCreateUser)
    s.AddTool(t.listUsers(), t.handleListUsers)
}

// Tool definitions — description pulled from swagger annotations where possible

func (t *UserTools) getUser() mcp.Tool {
    return mcp.NewTool("get_user",
        mcp.WithDescription("Retrieve a user by their ID"),
        mcp.WithString("id",
            mcp.Required(),
            mcp.Description("The user's unique identifier"),
        ),
    )
}

func (t *UserTools) handleGetUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    id, err := req.Params.Arguments.StringParam("id")
    if err != nil {
        return mcp.NewToolResultError("missing required parameter: id"), nil
    }

    u, err := t.usecase.GetUser(ctx, id)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("failed to get user: %s", err.Error())), nil
    }

    b, _ := json.Marshal(map[string]any{
        "id":    u.ID,
        "name":  u.Name,
        "email": u.Email,
        "role":  u.Role,
    })
    return mcp.NewToolResultText(string(b)), nil
}

func (t *UserTools) createUser() mcp.Tool {
    return mcp.NewTool("create_user",
        mcp.WithDescription("Create a new user account"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Full name of the user"),
        ),
        mcp.WithString("email",
            mcp.Required(),
            mcp.Description("Email address — must be unique"),
        ),
        mcp.WithString("role",
            mcp.Description("User role: admin or member (default: member)"),
        ),
    )
}

func (t *UserTools) handleCreateUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name,  _ := req.Params.Arguments.StringParam("name")
    email, _ := req.Params.Arguments.StringParam("email")
    role,  _ := req.Params.Arguments.StringParam("role")

    if role == "" {
        role = string(user.RoleMember)
    }

    u, err := t.usecase.CreateUser(ctx, user.CreateUserInput{
        Name:  name,
        Email: email,
        Role:  user.Role(role),
    })
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    b, _ := json.Marshal(map[string]any{"id": u.ID, "name": u.Name})
    return mcp.NewToolResultText(string(b)), nil
}

func (t *UserTools) listUsers() mcp.Tool {
    return mcp.NewTool("list_users",
        mcp.WithDescription("List all users with optional role filter"),
        mcp.WithString("role",
            mcp.Description("Filter by role: admin or member"),
        ),
    )
}

func (t *UserTools) handleListUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    role, _ := req.Params.Arguments.StringParam("role")

    users, err := t.usecase.ListUsers(ctx, user.ListUsersInput{Role: user.Role(role)})
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    b, _ := json.Marshal(users)
    return mcp.NewToolResultText(string(b)), nil
}
```

---

## 11. MCP — Feature Integration

MCP tools call feature usecases directly — the same usecases HTTP handlers use. No new layer needed.

```
HTTP Handler  ──┐
                ├──▶ feature/user.Usecase  ──▶ feature/user.Repository
MCP Tool      ──┘
```

The usecase does not know or care whether it was called from HTTP or MCP. This is the correct boundary.

### Wiring in `bootstrap/wire.go`

```go
// bootstrap/wire.go

// usecases (already constructed for HTTP/gRPC)
userUsecase    := user.NewUsecase(userRepo, authRepo, wsPublisher, asynqClient)
projectUsecase := project.NewUsecase(projectRepo, userRepo)

// MCP tools — reuse the same usecases
mcpHandlers := transportMCP.Handlers{
    User:    transportMCP.NewUserTools(userUsecase),
    Project: transportMCP.NewProjectTools(projectUsecase),
}

mcpServer := transportMCP.New(cfg, mcpHandlers)

// HTTP server gets the mcp handler mounted at /mcp
httpServer := transportHTTP.New(cfg, mw, httpHandlers, wsServer, mcpServer)
```

### Swaggo Annotations → MCP Description Consistency

To keep swagger and MCP descriptions in sync, define description constants:

```go
// feature/user/docs.go  (or top of usecase.go)
package user

// Description constants — used by both swaggo annotations and MCP tool definitions
const (
    DocGetUser    = "Retrieve a user by their ID"
    DocCreateUser = "Create a new user account"
    DocListUsers  = "List all users with optional role filter"
)
```

```go
// In MCP tool definition:
mcp.WithDescription(user.DocGetUser)

// In HTTP handler swaggo annotation:
// @Summary      Retrieve a user by their ID  ← same string
```

This is lightweight but effective — if the description changes, it changes in one place.

---

## 12. Rules Cheatsheet

### WebSocket

```
□ Use gorilla/websocket — most battle-tested, fine-grained control
□ Always use Hub/Client pattern — never manage connections directly in handlers
□ One Hub per server process — lives at transport/ws/hub.go
□ One Client per connection — lives at transport/ws/client.go
□ Client has exactly two goroutines: readPump and writePump
□ Hub.Broadcast publishes to Redis AND delivers locally
□ Redis subscription delivers locally only — never re-publish to avoid loops
□ Features define a local wsPublisher interface — never import transport/ws
□ transport/ws/publisher.go wraps Hub and satisfies the interface
□ bootstrap/wire.go creates Hub, wraps it in Publisher, injects into usecases
□ Rooms are optional groupings managed by Hub — use for scoped broadcasts
□ Unresponsive clients (full send buffer) are unregistered immediately
□ WS routes registered via server.RegisterRoutes(r) in transport/http/server.go
□ Hub.Run(ctx) is started in bootstrap/server.go errgroup
```

### MCP

```
□ Use mark3labs/mcp-go — most complete Go MCP implementation
□ Use SSE transport — integrates with chi, mount at /mcp
□ MCP tools call feature usecases directly — same usecases HTTP handlers use
□ One *Tools struct per feature in transport/mcp/
□ Tools.Register(s) called in transport/mcp/server.go New() constructor
□ Tool handlers return mcp.NewToolResultError for domain errors — never panic
□ Tool handlers return mcp.NewToolResultText with JSON for success
□ Define description constants in feature package — shared by swagger and MCP
□ MCP server wired in bootstrap/wire.go with existing usecases — no duplication
□ Mount mcpServer.Handler() in transport/http/server.go at /mcp
□ MCP auth: protect /mcp routes with the same Auth middleware as HTTP routes
□ Never auto-detect routes from chi — always register tools explicitly
```

### General Transport Rules

```
□ Transport packages import feature handler/tool constructors only
□ Transport packages never contain business logic
□ bootstrap/wire.go is the only place that imports all transport + feature packages
□ Each transport has Start(ctx) + Stop()/Shutdown() — consistent interface for App
□ App struct in bootstrap/app.go holds one field per transport server
□ bootstrap/server.go runs all servers in errgroup, handles signal shutdown
```

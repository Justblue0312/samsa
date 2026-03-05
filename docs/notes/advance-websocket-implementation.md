# Advanced WebSocket Implementation

## Table of Contents

1. [Design Principles for Scalable WS Features](#1-design-principles-for-scalable-ws-features)
2. [Complete Directory Structure](#2-complete-directory-structure)
3. [Message Envelope — The Foundation](#3-message-envelope--the-foundation)
4. [Feature Handler Registry — The Expansion Point](#4-feature-handler-registry--the-expansion-point)
5. [Core Infrastructure](#5-core-infrastructure)
6. [Complete Hub Implementation](#6-complete-hub-implementation)
7. [Complete Client Implementation](#7-complete-client-implementation)
8. [Feature 1 — Presence (Online/Offline)](#8-feature-1--presence-onlineoffline)
9. [Feature 2 — Notifications](#9-feature-2--notifications)
10. [Feature 3 — Typing Indicators](#10-feature-3--typing-indicators)
11. [Feature 4 — Live Cursor / Collaborative](#11-feature-4--live-cursor--collaborative)
12. [Publisher Interface — Feature Integration](#12-publisher-interface--feature-integration)
13. [Transport Server](#13-transport-server)
14. [Bootstrap Wiring](#14-bootstrap-wiring)
15. [Testing WebSocket Features](#15-testing-websocket-features)
16. [Adding a New Feature — Step-by-Step Checklist](#16-adding-a-new-feature--step-by-step-checklist)
17. [Rules Cheatsheet](#17-rules-cheatsheet)

---

## 1. Design Principles for Scalable WS Features

The problem with naive hub implementations is that the hub accumulates every feature's logic over time:

```go
// ❌ what happens without a plan — hub becomes a god object
func (h *Hub) handleInbound(msg InboundMessage) {
    switch msg.Type {
    case "ping":           h.handlePresencePing(msg)
    case "typing_start":   h.handleTypingStart(msg)
    case "cursor_move":    h.handleCursorMove(msg)
    case "chat_message":   h.handleChat(msg)
    // ... grows forever
    }
}
```

The correct approach is a **feature handler registry**. The hub is a router, not an implementor. Each feature registers its own handler for its own message types. The hub never needs to change when a new feature is added.

```
New WS Feature checklist:
  1. Define message types (constants in feature package)
  2. Implement a FeatureHandler in transport/ws/handlers/
  3. Register it in transport/ws/server.go
  4. Done — hub.go never changes
```

---

## 2. Complete Directory Structure

```
transport/ws/
  server.go           ← upgrader, route registration, hub lifecycle
  hub.go              ← router only: register/unregister/route messages
  client.go           ← single connection: readPump + writePump
  room.go             ← scoped client grouping
  publisher.go        ← outbound adapter for feature usecases
  registry.go         ← FeatureHandler interface + handler registry
  message.go          ← Envelope, message type constants, helpers
  handlers/
    presence.go       ← handles presence ping/pong, online state
    notification.go   ← handles notification delivery
    typing.go         ← handles typing indicators
    cursor.go         ← handles live cursor / collaborative state

infra/redis/
  presence.go         ← presence counter store (INCR/DECR/Lua)
  pubsub.go           ← cross-instance broadcast helpers
```

---

## 3. Message Envelope — The Foundation

Every WebSocket message — inbound and outbound — uses the same JSON envelope. The `type` field is the routing key. This is the contract between server and every client.

```go
// transport/ws/message.go
package ws

import (
    "encoding/json"
    "fmt"
    "time"
)

// Envelope is the standard wrapper for every WS message in both directions.
// Inbound:  client → server
// Outbound: server → client
type Envelope struct {
    Type      string          `json:"type"`
    RequestID string          `json:"request_id,omitempty"` // client sets this for ack
    Payload   json.RawMessage `json:"payload,omitempty"`
    Timestamp time.Time       `json:"ts"`
    Error     *EnvelopeError  `json:"error,omitempty"` // set only on error responses
}

type EnvelopeError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// ── Message type constants ────────────────────────────────────────────────────
// Convention: "<feature>.<action>"
// Inbound types (client → server) use imperative form: "presence.ping"
// Outbound types (server → client) use past tense:    "presence.online"

const (
    // system
    TypeSystemError = "system.error"
    TypeSystemAck   = "system.ack"

    // presence (feature 1)
    TypePresencePing    = "presence.ping"     // client → server
    TypePresenceOnline  = "presence.online"   // server → client
    TypePresenceOffline = "presence.offline"  // server → client

    // notifications (feature 2)
    TypeNotificationNew  = "notification.new"   // server → client
    TypeNotificationRead = "notification.read"  // client → server (mark read)

    // typing (feature 3)
    TypeTypingStart = "typing.start" // client → server
    TypeTypingStop  = "typing.stop"  // client → server
    TypeTypingState = "typing.state" // server → client (broadcast to room)

    // cursor (feature 4)
    TypeCursorMove  = "cursor.move"  // client → server
    TypeCursorState = "cursor.state" // server → client (broadcast to room)
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// NewOutbound builds a server→client envelope.
func NewOutbound(msgType string, payload any) (*Envelope, error) {
    b, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("ws: marshal outbound payload: %w", err)
    }
    return &Envelope{
        Type:      msgType,
        Payload:   b,
        Timestamp: time.Now(),
    }, nil
}

// NewErrorOutbound builds an error envelope to send to a specific client.
func NewErrorOutbound(requestID, code, msg string) *Envelope {
    return &Envelope{
        Type:      TypeSystemError,
        RequestID: requestID,
        Timestamp: time.Now(),
        Error:     &EnvelopeError{Code: code, Message: msg},
    }
}

// MarshalEnvelope serialises an envelope to bytes for the wire.
func MarshalEnvelope(e *Envelope) ([]byte, error) {
    return json.Marshal(e)
}

// ParseEnvelope deserialises bytes from the wire into an envelope.
func ParseEnvelope(data []byte) (*Envelope, error) {
    var e Envelope
    if err := json.Unmarshal(data, &e); err != nil {
        return nil, fmt.Errorf("ws: parse envelope: %w", err)
    }
    return &e, nil
}
```

---

## 4. Feature Handler Registry — The Expansion Point

This is what makes the hub infinitely extensible. The hub never changes when you add features.

```go
// transport/ws/registry.go
package ws

import (
    "context"
    "log/slog"
    "sync"
)

// FeatureHandler handles all inbound messages for a specific feature.
// Each feature in transport/ws/handlers/ implements this interface.
type FeatureHandler interface {
    // Types returns the list of message types this handler owns.
    // e.g. []string{"presence.ping", "presence.status"}
    Types() []string

    // Handle is called by the hub for every inbound message whose type
    // matches one of the types returned by Types().
    Handle(ctx context.Context, client *Client, env *Envelope)

    // OnConnect is called when a client first connects.
    // Use for initialising per-client state (e.g. send initial presence snapshot).
    OnConnect(ctx context.Context, client *Client)

    // OnDisconnect is called when a client disconnects.
    // Use for cleanup (e.g. decrement presence counter, broadcast offline).
    OnDisconnect(ctx context.Context, client *Client)
}

// Registry holds all registered feature handlers and dispatches inbound messages.
type Registry struct {
    mu       sync.RWMutex
    handlers map[string]FeatureHandler // msgType → handler
    all      []FeatureHandler          // for OnConnect/OnDisconnect iteration
}

func NewRegistry() *Registry {
    return &Registry{
        handlers: make(map[string]FeatureHandler),
    }
}

// Register adds a feature handler. Called once at startup from server.go.
func (r *Registry) Register(h FeatureHandler) {
    r.mu.Lock()
    defer r.mu.Unlock()

    for _, t := range h.Types() {
        if existing, ok := r.handlers[t]; ok {
            // two handlers claiming the same type is a programmer error
            panic("ws: duplicate handler for type " + t + " (already registered by " +
                fmt.Sprintf("%T", existing) + ")")
        }
        r.handlers[t] = h
    }
    r.all = append(r.all, h)
}

// Dispatch routes an inbound message to the correct feature handler.
func (r *Registry) Dispatch(ctx context.Context, client *Client, env *Envelope) {
    r.mu.RLock()
    h, ok := r.handlers[env.Type]
    r.mu.RUnlock()

    if !ok {
        slog.Warn("ws: no handler for message type", "type", env.Type, "userID", client.UserID)
        // send error back to the specific client
        client.SendError(env.RequestID, "UNKNOWN_TYPE",
            "no handler registered for type: "+env.Type)
        return
    }

    h.Handle(ctx, client, env)
}

// OnConnect notifies all registered handlers that a client connected.
func (r *Registry) OnConnect(ctx context.Context, client *Client) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, h := range r.all {
        h.OnConnect(ctx, client)
    }
}

// OnDisconnect notifies all registered handlers that a client disconnected.
func (r *Registry) OnDisconnect(ctx context.Context, client *Client) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, h := range r.all {
        h.OnDisconnect(ctx, client)
    }
}
```

---

## 5. Core Infrastructure

### `infra/redis/presence.go`

```go
// infra/redis/presence.go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

const PresenceTTL = 90 * time.Second

type PresenceStore struct {
    rdb *redis.Client
}

func NewPresenceStore(rdb *redis.Client) *PresenceStore {
    return &PresenceStore{rdb: rdb}
}

func presenceKey(userID string) string {
    return fmt.Sprintf("ws:presence:%s", userID)
}

// Connect atomically increments the connection counter and sets TTL.
func (p *PresenceStore) Connect(ctx context.Context, userID string) error {
    key := presenceKey(userID)
    pipe := p.rdb.Pipeline()
    pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, PresenceTTL)
    _, err := pipe.Exec(ctx)
    return err
}

// Disconnect atomically decrements counter and deletes key when it reaches 0.
// Returns the remaining connection count.
func (p *PresenceStore) Disconnect(ctx context.Context, userID string) (int64, error) {
    // Lua script: DECR and DEL in a single atomic operation
    // Prevents the race between DECR and a subsequent DEL
    script := redis.NewScript(`
        local count = redis.call('DECR', KEYS[1])
        if count <= 0 then
            redis.call('DEL', KEYS[1])
            return 0
        end
        return count
    `)
    result, err := script.Run(ctx, p.rdb, []string{presenceKey(userID)}).Int64()
    if err != nil && err != redis.Nil {
        return 0, err
    }
    return result, nil
}

// Refresh extends TTL on every successful pong.
// EXPIRE returns 0 if key does not exist — safe to call regardless.
func (p *PresenceStore) Refresh(ctx context.Context, userID string) error {
    return p.rdb.Expire(ctx, presenceKey(userID), PresenceTTL).Err()
}

// IsOnline returns true if the user has at least one active connection.
func (p *PresenceStore) IsOnline(ctx context.Context, userID string) (bool, error) {
    count, err := p.rdb.Get(ctx, presenceKey(userID)).Int64()
    if err == redis.Nil {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// OnlineBatch returns online status for a list of userIDs in one round-trip.
func (p *PresenceStore) OnlineBatch(ctx context.Context, userIDs []string) (map[string]bool, error) {
    if len(userIDs) == 0 {
        return map[string]bool{}, nil
    }

    keys := make([]string, len(userIDs))
    for i, id := range userIDs {
        keys[i] = presenceKey(id)
    }

    vals, err := p.rdb.MGet(ctx, keys...).Result()
    if err != nil {
        return nil, err
    }

    result := make(map[string]bool, len(userIDs))
    for i, v := range vals {
        result[userIDs[i]] = v != nil
    }
    return result, nil
}
```

### `infra/redis/pubsub.go`

```go
// infra/redis/pubsub.go
package redis

// Channel name conventions for cross-instance WS broadcasting.
const (
    ChanWSBroadcast = "ws:broadcast"        // global broadcast
    ChanWSRoom      = "ws:room:%s"          // per-room broadcast, format with roomID
    ChanWSUser      = "ws:user:%s"          // per-user, format with userID
)
```

---

## 6. Complete Hub Implementation

The hub is a **pure router**. It never implements feature logic.

```go
// transport/ws/hub.go
package ws

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    "github.com/redis/go-redis/v9"
)

// OutboundMessage is what the hub fans out to connected clients.
type OutboundMessage struct {
    RoomID  string // empty = not scoped to a room
    UserID  string // empty = all clients in room; set to target one user
    Global  bool   // true = send to all connected clients regardless of room
    Payload []byte // pre-marshalled Envelope
}

// OfflineSchedule tells the hub to check and emit offline after a delay.
type OfflineSchedule struct {
    UserID string
    After  time.Duration
}

type Hub struct {
    register        chan *Client
    unregister      chan *Client
    inbound         chan InboundMessage
    Broadcast       chan OutboundMessage
    scheduleOffline chan OfflineSchedule

    rooms   map[string]*Room
    globals map[*Client]struct{}

    registry *Registry
    rdb      *redis.Client
}

func NewHub(registry *Registry, rdb *redis.Client) *Hub {
    return &Hub{
        register:        make(chan *Client),
        unregister:      make(chan *Client),
        inbound:         make(chan InboundMessage, 512),
        Broadcast:       make(chan OutboundMessage, 512),
        scheduleOffline: make(chan OfflineSchedule, 64),
        rooms:           make(map[string]*Room),
        globals:         make(map[*Client]struct{}),
        registry:        registry,
        rdb:             rdb,
    }
}

// Run is the single event loop for the hub. One goroutine, no locks needed.
// Started in bootstrap/server.go via errgroup.
func (h *Hub) Run(ctx context.Context) {
    sub := h.rdb.Subscribe(ctx, "ws:broadcast")
    defer sub.Close()

    redisMsgs := sub.Channel()

    for {
        select {
        case <-ctx.Done():
            return

        case client := <-h.register:
            h.addClient(client)
            h.registry.OnConnect(ctx, client) // notify all feature handlers

        case client := <-h.unregister:
            h.registry.OnDisconnect(ctx, client) // notify all feature handlers first
            h.removeClient(client)

        case msg := <-h.inbound:
            env, err := ParseEnvelope(msg.Data)
            if err != nil {
                slog.Warn("ws: malformed inbound message",
                    "userID", msg.Client.UserID, "error", err)
                continue
            }
            // dispatch to the registered feature handler — hub does nothing else
            h.registry.Dispatch(ctx, msg.Client, env)

        case msg := <-h.Broadcast:
            h.publishToRedis(ctx, msg) // cross-instance
            h.deliverLocally(msg)      // same instance

        case redisMsg := <-redisMsgs:
            var msg OutboundMessage
            if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
                slog.Error("ws: failed to parse redis message", "error", err)
                continue
            }
            h.deliverLocally(msg) // local only — never re-publish

        case event := <-h.scheduleOffline:
            go h.handleOfflineSchedule(ctx, event)
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
    if _, ok := h.globals[c]; ok {
        delete(h.globals, c)
    }
    if c.RoomID != "" {
        if room, ok := h.rooms[c.RoomID]; ok {
            room.remove(c)
            if room.isEmpty() {
                delete(h.rooms, c.RoomID)
            }
        }
    }
    close(c.send) // signal writePump to exit
}

func (h *Hub) deliverLocally(msg OutboundMessage) {
    var targets []*Client

    switch {
    case msg.Global:
        for c := range h.globals {
            targets = append(targets, c)
        }
        for _, room := range h.rooms {
            for c := range room.clients {
                targets = append(targets, c)
            }
        }
    case msg.RoomID != "":
        if room, ok := h.rooms[msg.RoomID]; ok {
            for c := range room.clients {
                targets = append(targets, c)
            }
        }
    default:
        // UserID-targeted with no room — search globals
        for c := range h.globals {
            targets = append(targets, c)
        }
    }

    for _, c := range targets {
        if msg.UserID != "" && c.UserID != msg.UserID {
            continue
        }
        select {
        case c.send <- msg.Payload:
        default:
            // buffer full — client is too slow, drop and unregister
            slog.Warn("ws: client send buffer full, dropping", "userID", c.UserID)
            h.removeClient(c)
        }
    }
}

func (h *Hub) publishToRedis(ctx context.Context, msg OutboundMessage) {
    b, err := json.Marshal(msg)
    if err != nil {
        slog.Error("ws: marshal broadcast for redis", "error", err)
        return
    }
    if err := h.rdb.Publish(ctx, "ws:broadcast", b).Err(); err != nil {
        slog.Error("ws: publish to redis", "error", err)
    }
}

// handleOfflineSchedule waits then checks if user is truly offline.
// The 3s delay absorbs browser tab refresh / network blip reconnects.
func (h *Hub) handleOfflineSchedule(ctx context.Context, event OfflineSchedule) {
    timer := time.NewTimer(event.After)
    defer timer.Stop()

    select {
    case <-ctx.Done():
        return
    case <-timer.C:
        // re-check online status — if reconnected during delay, abort
        h.registry.Dispatch(ctx, nil, &Envelope{
            Type:    "_internal.check_offline",
            Payload: mustMarshal(map[string]string{"user_id": event.UserID}),
        })
    }
}

func mustMarshal(v any) []byte {
    b, _ := json.Marshal(v)
    return b
}
```

---

## 7. Complete Client Implementation

```go
// transport/ws/client.go
package ws

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    "github.com/gorilla/websocket"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10 // 54s
    maxMessageSize = 4096
)

// InboundMessage is a raw message received from a client before parsing.
type InboundMessage struct {
    Client *Client
    Data   []byte
}

type Client struct {
    hub    *Hub
    conn   *websocket.Conn
    send   chan []byte

    UserID string
    RoomID string
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

// Start registers the client with the hub and launches pumps.
func (c *Client) Start() {
    c.hub.register <- c
    go c.writePump()
    go c.readPump()
}

// Send queues a pre-marshalled message for delivery. Non-blocking.
// Returns false if the buffer is full.
func (c *Client) Send(data []byte) bool {
    select {
    case c.send <- data:
        return true
    default:
        return false
    }
}

// SendEnvelope marshals and queues an envelope.
func (c *Client) SendEnvelope(env *Envelope) error {
    b, err := MarshalEnvelope(env)
    if err != nil {
        return err
    }
    if !c.Send(b) {
        return fmt.Errorf("ws: send buffer full for user %s", c.UserID)
    }
    return nil
}

// SendError sends a typed error back to this specific client.
func (c *Client) SendError(requestID, code, msg string) {
    env := NewErrorOutbound(requestID, code, msg)
    b, _ := MarshalEnvelope(env)
    c.Send(b)
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(appData string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        // hub registry handles TTL refresh via OnPong if needed
        return nil
    })

    for {
        _, data, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure,
            ) {
                slog.Warn("ws: unexpected close", "userID", c.UserID, "error", err)
            }
            return
        }
        c.hub.inbound <- InboundMessage{Client: c, Data: data}
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case data, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
                slog.Warn("ws: write error", "userID", c.UserID, "error", err)
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

---

## 8. Feature 1 — Presence (Online/Offline)

```go
// transport/ws/handlers/presence.go
package handlers

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    "myapp/internal/transport/ws"
)

// PresenceStore is the local interface — handler never imports infra/redis
type PresenceStore interface {
    Connect(ctx context.Context, userID string) error
    Disconnect(ctx context.Context, userID string) (remaining int64, err error)
    Refresh(ctx context.Context, userID string) error
    IsOnline(ctx context.Context, userID string) (bool, error)
}

type PresenceHandler struct {
    store PresenceStore
    hub   *ws.Hub
}

func NewPresenceHandler(store PresenceStore, hub *ws.Hub) *PresenceHandler {
    return &PresenceHandler{store: store, hub: hub}
}

func (h *PresenceHandler) Types() []string {
    return []string{
        ws.TypePresencePing,
        "_internal.check_offline", // internal event from hub's delayed offline check
    }
}

// OnConnect — called by registry when a new client registers.
// Increments counter and broadcasts online to relevant parties.
func (h *PresenceHandler) OnConnect(ctx context.Context, client *ws.Client) {
    if err := h.store.Connect(ctx, client.UserID); err != nil {
        slog.Error("presence: connect store error",
            "userID", client.UserID, "error", err)
        return
    }

    env, _ := ws.NewOutbound(ws.TypePresenceOnline, map[string]string{
        "user_id": client.UserID,
    })
    b, _ := ws.MarshalEnvelope(env)

    // broadcast to all so other users see them come online
    h.hub.Broadcast <- ws.OutboundMessage{Global: true, Payload: b}
}

// OnDisconnect — called by registry when a client unregisters.
// Decrements counter. Schedules offline notification with 3s delay
// to absorb page refresh / tab switch reconnects.
func (h *PresenceHandler) OnDisconnect(ctx context.Context, client *ws.Client) {
    remaining, err := h.store.Disconnect(ctx, client.UserID)
    if err != nil {
        slog.Error("presence: disconnect store error",
            "userID", client.UserID, "error", err)
        return
    }

    if remaining <= 0 {
        // last connection — schedule offline with delay
        h.hub.ScheduleOffline(client.UserID, 3*time.Second)
    }
}

// Handle routes inbound presence messages.
func (h *PresenceHandler) Handle(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    switch env.Type {
    case ws.TypePresencePing:
        h.handlePing(ctx, client, env)
    case "_internal.check_offline":
        h.handleInternalCheckOffline(ctx, env)
    }
}

func (h *PresenceHandler) handlePing(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    // extend TTL — client is still alive
    if err := h.store.Refresh(ctx, client.UserID); err != nil {
        slog.Error("presence: refresh error",
            "userID", client.UserID, "error", err)
    }

    // ack back to the client that sent the ping
    ack, _ := ws.NewOutbound(ws.TypeSystemAck, map[string]string{
        "request_id": env.RequestID,
    })
    _ = client.SendEnvelope(ack)
}

func (h *PresenceHandler) handleInternalCheckOffline(ctx context.Context, env *ws.Envelope) {
    var payload struct {
        UserID string `json:"user_id"`
    }
    if err := json.Unmarshal(env.Payload, &payload); err != nil {
        return
    }

    online, err := h.store.IsOnline(ctx, payload.UserID)
    if err != nil || online {
        return // still online or error — do not emit offline
    }

    offlineEnv, _ := ws.NewOutbound(ws.TypePresenceOffline, map[string]string{
        "user_id": payload.UserID,
    })
    b, _ := ws.MarshalEnvelope(offlineEnv)

    // broadcast offline to everyone
    h.hub.Broadcast <- ws.OutboundMessage{Global: true, Payload: b}
}
```

---

## 9. Feature 2 — Notifications

```go
// transport/ws/handlers/notification.go
package handlers

import (
    "context"

    "myapp/internal/transport/ws"
)

// NotificationSender is the local interface for marking notifications read.
type NotificationSender interface {
    MarkRead(ctx context.Context, userID, notifID string) error
}

type NotificationHandler struct {
    usecase NotificationSender
}

func NewNotificationHandler(uc NotificationSender) *NotificationHandler {
    return &NotificationHandler{usecase: uc}
}

func (h *NotificationHandler) Types() []string {
    return []string{ws.TypeNotificationRead}
}

func (h *NotificationHandler) OnConnect(ctx context.Context, client *ws.Client) {
    // could send unread notification count on connect — optional
}

func (h *NotificationHandler) OnDisconnect(ctx context.Context, client *ws.Client) {
    // nothing to clean up for notifications
}

func (h *NotificationHandler) Handle(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    switch env.Type {
    case ws.TypeNotificationRead:
        h.handleRead(ctx, client, env)
    }
}

func (h *NotificationHandler) handleRead(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    var payload struct {
        NotifID string `json:"notif_id"`
    }
    if err := json.Unmarshal(env.Payload, &payload); err != nil {
        client.SendError(env.RequestID, "BAD_PAYLOAD", "invalid notification read payload")
        return
    }

    if err := h.usecase.MarkRead(ctx, client.UserID, payload.NotifID); err != nil {
        client.SendError(env.RequestID, "MARK_READ_FAILED", err.Error())
        return
    }

    ack, _ := ws.NewOutbound(ws.TypeSystemAck, map[string]string{
        "request_id": env.RequestID,
        "notif_id":   payload.NotifID,
    })
    _ = client.SendEnvelope(ack)
}
```

Notifications are **pushed from the server** when a feature usecase calls the publisher. The handler above only handles the client→server "mark as read" direction. Server→client delivery goes through `hub.Broadcast` via the `Publisher`.

---

## 10. Feature 3 — Typing Indicators

Typing indicators are **ephemeral** — no persistence, pure real-time fan-out to the room. TTL in Redis prevents stale "still typing" state if a client disconnects without sending `typing.stop`.

```go
// transport/ws/handlers/typing.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/redis/go-redis/v9"
    "myapp/internal/transport/ws"
)

const typingTTL = 5 * time.Second // auto-expire typing state if no update

// TypingStore is the local interface for ephemeral typing state.
type TypingStore interface {
    SetTyping(ctx context.Context, roomID, userID string) error
    ClearTyping(ctx context.Context, roomID, userID string) error
    GetTypingUsers(ctx context.Context, roomID string) ([]string, error)
}

type TypingHandler struct {
    store TypingStore
    hub   *ws.Hub
}

func NewTypingHandler(store TypingStore, hub *ws.Hub) *TypingHandler {
    return &TypingHandler{store: store, hub: hub}
}

func (h *TypingHandler) Types() []string {
    return []string{ws.TypeTypingStart, ws.TypeTypingStop}
}

func (h *TypingHandler) OnConnect(ctx context.Context, client *ws.Client) {}

func (h *TypingHandler) OnDisconnect(ctx context.Context, client *ws.Client) {
    // clear typing state if client disconnects mid-type
    if client.RoomID != "" {
        _ = h.store.ClearTyping(context.Background(), client.RoomID, client.UserID)
        h.broadcastTypingState(context.Background(), client.RoomID)
    }
}

func (h *TypingHandler) Handle(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    if client.RoomID == "" {
        client.SendError(env.RequestID, "NO_ROOM", "typing requires a room connection")
        return
    }
    switch env.Type {
    case ws.TypeTypingStart:
        _ = h.store.SetTyping(ctx, client.RoomID, client.UserID)
        h.broadcastTypingState(ctx, client.RoomID)
    case ws.TypeTypingStop:
        _ = h.store.ClearTyping(ctx, client.RoomID, client.UserID)
        h.broadcastTypingState(ctx, client.RoomID)
    }
}

func (h *TypingHandler) broadcastTypingState(ctx context.Context, roomID string) {
    users, err := h.store.GetTypingUsers(ctx, roomID)
    if err != nil {
        slog.Error("typing: get typing users", "roomID", roomID, "error", err)
        return
    }

    env, _ := ws.NewOutbound(ws.TypeTypingState, map[string]any{
        "room_id":      roomID,
        "typing_users": users,
    })
    b, _ := ws.MarshalEnvelope(env)

    h.hub.Broadcast <- ws.OutboundMessage{RoomID: roomID, Payload: b}
}
```

`TypingStore` implementation using Redis SADD with key expiry:

```go
// infra/redis/typing.go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

type TypingStore struct{ rdb *redis.Client }

func NewTypingStore(rdb *redis.Client) *TypingStore { return &TypingStore{rdb: rdb} }

func typingKey(roomID string) string { return fmt.Sprintf("ws:typing:%s", roomID) }

func (s *TypingStore) SetTyping(ctx context.Context, roomID, userID string) error {
    key := typingKey(roomID)
    pipe := s.rdb.Pipeline()
    pipe.SAdd(ctx, key, userID)
    pipe.Expire(ctx, key, 5*time.Second)
    _, err := pipe.Exec(ctx)
    return err
}

func (s *TypingStore) ClearTyping(ctx context.Context, roomID, userID string) error {
    return s.rdb.SRem(ctx, typingKey(roomID), userID).Err()
}

func (s *TypingStore) GetTypingUsers(ctx context.Context, roomID string) ([]string, error) {
    return s.rdb.SMembers(ctx, typingKey(roomID)).Result()
}
```

---

## 11. Feature 4 — Live Cursor / Collaborative

Cursor positions are the most high-frequency messages. Store only in memory (not Redis) — they are too ephemeral for persistence.

```go
// transport/ws/handlers/cursor.go
package handlers

import (
    "context"
    "encoding/json"
    "sync"

    "myapp/internal/transport/ws"
)

type CursorPosition struct {
    UserID string  `json:"user_id"`
    X      float64 `json:"x"`
    Y      float64 `json:"y"`
    PageID string  `json:"page_id,omitempty"`
}

// CursorHandler tracks cursor positions in memory — no Redis, too ephemeral.
// Positions are stored per room per user. Cleared on disconnect.
type CursorHandler struct {
    hub      *ws.Hub
    mu       sync.RWMutex
    positions map[string]map[string]CursorPosition // roomID → userID → position
}

func NewCursorHandler(hub *ws.Hub) *CursorHandler {
    return &CursorHandler{
        hub:       hub,
        positions: make(map[string]map[string]CursorPosition),
    }
}

func (h *CursorHandler) Types() []string {
    return []string{ws.TypeCursorMove}
}

func (h *CursorHandler) OnConnect(ctx context.Context, client *ws.Client) {
    if client.RoomID == "" {
        return
    }
    // send current cursor snapshot to the newly connected client
    h.mu.RLock()
    positions := h.snapshotRoom(client.RoomID)
    h.mu.RUnlock()

    if len(positions) == 0 {
        return
    }
    env, _ := ws.NewOutbound(ws.TypeCursorState, map[string]any{
        "room_id":   client.RoomID,
        "positions": positions,
    })
    _ = client.SendEnvelope(env)
}

func (h *CursorHandler) OnDisconnect(ctx context.Context, client *ws.Client) {
    if client.RoomID == "" {
        return
    }
    h.mu.Lock()
    if room, ok := h.positions[client.RoomID]; ok {
        delete(room, client.UserID)
        if len(room) == 0 {
            delete(h.positions, client.RoomID)
        }
    }
    h.mu.Unlock()
    h.broadcastCursorState(ctx, client.RoomID)
}

func (h *CursorHandler) Handle(ctx context.Context, client *ws.Client, env *ws.Envelope) {
    if client.RoomID == "" {
        return
    }

    var pos CursorPosition
    if err := json.Unmarshal(env.Payload, &pos); err != nil {
        client.SendError(env.RequestID, "BAD_PAYLOAD", "invalid cursor position")
        return
    }
    pos.UserID = client.UserID // always authoritative from server

    h.mu.Lock()
    if _, ok := h.positions[client.RoomID]; !ok {
        h.positions[client.RoomID] = make(map[string]CursorPosition)
    }
    h.positions[client.RoomID][client.UserID] = pos
    h.mu.Unlock()

    h.broadcastCursorState(ctx, client.RoomID)
}

func (h *CursorHandler) broadcastCursorState(ctx context.Context, roomID string) {
    h.mu.RLock()
    positions := h.snapshotRoom(roomID)
    h.mu.RUnlock()

    env, _ := ws.NewOutbound(ws.TypeCursorState, map[string]any{
        "room_id":   roomID,
        "positions": positions,
    })
    b, _ := ws.MarshalEnvelope(env)
    h.hub.Broadcast <- ws.OutboundMessage{RoomID: roomID, Payload: b}
}

func (h *CursorHandler) snapshotRoom(roomID string) []CursorPosition {
    room, ok := h.positions[roomID]
    if !ok {
        return nil
    }
    result := make([]CursorPosition, 0, len(room))
    for _, p := range room {
        result = append(result, p)
    }
    return result
}
```

---

## 12. Publisher Interface — Feature Integration

Features publish to the WebSocket layer via a local interface. No feature ever imports `transport/ws`.

```go
// transport/ws/publisher.go
package ws

import "context"

// Publisher wraps Hub and satisfies any feature's wsPublisher local interface.
type Publisher struct{ hub *Hub }

func NewPublisher(hub *Hub) *Publisher { return &Publisher{hub: hub} }

// Publish sends a message to a specific user, a room, or globally.
func (p *Publisher) Publish(ctx context.Context, roomID, userID string, payload []byte) error {
    select {
    case p.hub.Broadcast <- OutboundMessage{
        RoomID:  roomID,
        UserID:  userID,
        Global:  roomID == "" && userID == "",
        Payload: payload,
    }:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

Usage in a feature usecase:

```go
// feature/notification/usecase_impl.go

// wsPublisher is the local interface — defined here, satisfied by ws.Publisher
type wsPublisher interface {
    Publish(ctx context.Context, roomID, userID string, payload []byte) error
}

func (u *usecase) NotifyUser(ctx context.Context, userID string, msg Message) error {
    b, err := json.Marshal(ws.Envelope{
        Type:    ws.TypeNotificationNew,
        Payload: mustMarshal(msg),
    })
    if err != nil {
        return err
    }
    return u.wsPub.Publish(ctx, "", userID, b)
}
```

---

## 13. Transport Server

```go
// transport/ws/server.go
package ws

import (
    "context"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/gorilla/websocket"

    "myapp/internal/settings"
    "myapp/internal/transport/ws/handlers"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // restrict in production
    },
}

// Handlers groups all ws feature handlers for registration.
// Add one field here when adding a new feature handler.
type Handlers struct {
    Presence     *handlers.PresenceHandler
    Notification *handlers.NotificationHandler
    Typing       *handlers.TypingHandler
    Cursor       *handlers.CursorHandler
}

type Server struct {
    hub      *Hub
    registry *Registry
    cfg      *settings.Config
}

func New(cfg *settings.Config, hub *Hub, registry *Registry, h Handlers) *Server {
    // register all feature handlers — order does not matter
    registry.Register(h.Presence)
    registry.Register(h.Notification)
    registry.Register(h.Typing)
    registry.Register(h.Cursor)

    return &Server{hub: hub, registry: registry, cfg: cfg}
}

// RegisterRoutes mounts WS endpoints onto the chi router.
// Called from transport/http/server.go.
func (s *Server) RegisterRoutes(r chi.Router) {
    // global connection (no room)
    r.Get("/ws", s.handleUpgrade(""))

    // room-scoped connection
    r.Get("/ws/room/{roomID}", func(w http.ResponseWriter, r *http.Request) {
        roomID := chi.URLParam(r, "roomID")
        s.handleUpgrade(roomID)(w, r)
    })
}

func (s *Server) handleUpgrade(roomID string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            return
        }
        userID := userIDFromCtx(r.Context())
        if userID == "" {
            conn.Close()
            return
        }
        client := NewClient(s.hub, conn, userID, roomID)
        client.Start()
    }
}

// Start runs the hub. Called from bootstrap/server.go errgroup.
func (s *Server) Start(ctx context.Context) error {
    s.hub.Run(ctx)
    return nil
}

func (s *Server) Stop() {} // hub stops when ctx cancels
```

---

## 14. Bootstrap Wiring

```go
// bootstrap/wire.go

// infra
rdb := redis.New(cfg.RedisAddr)

// stores
presenceStore := redisinfra.NewPresenceStore(rdb)
typingStore   := redisinfra.NewTypingStore(rdb)

// ws core
wsRegistry := ws.NewRegistry()
wsHub      := ws.NewHub(wsRegistry, rdb)
wsPublisher := ws.NewPublisher(wsHub)

// feature usecases (inject wsPublisher as local interface)
notifUsecase := notification.NewUsecase(notifRepo, wsPublisher)
userUsecase  := user.NewUsecase(userRepo, authRepo, wsPublisher, asynqClient)

// ws feature handlers — each gets the deps it needs
wsHandlers := transportWS.Handlers{
    Presence:     handlers.NewPresenceHandler(presenceStore, wsHub),
    Notification: handlers.NewNotificationHandler(notifUsecase),
    Typing:       handlers.NewTypingHandler(typingStore, wsHub),
    Cursor:       handlers.NewCursorHandler(wsHub),
}

wsServer := transportWS.New(cfg, wsHub, wsRegistry, wsHandlers)

// http server gets wsServer for route registration
httpServer := transportHTTP.New(cfg, mw, httpHandlers, wsServer, mcpServer)

return &App{
    HTTP:   httpServer,
    GRPC:   grpcServer,
    Worker: workerServer,
    WS:     wsServer,
    DB:     pool,
    Redis:  rdb,
}, nil
```

---

## 15. Testing WebSocket Features

Each feature handler is a plain struct implementing `FeatureHandler` — testable without a real WebSocket connection.

```go
// transport/ws/handlers/presence_test.go
package handlers_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "myapp/internal/transport/ws"
    "myapp/internal/transport/ws/handlers"
    "myapp/internal/transport/ws/handlers/mocks"
)

func TestPresenceHandler_OnConnect(t *testing.T) {
    ctx := context.Background()

    t.Run("increments counter and broadcasts online", func(t *testing.T) {
        store := mocks.NewPresenceStore(t)
        hub   := ws.NewTestHub() // hub with a captured Broadcast channel
        h     := handlers.NewPresenceHandler(store, hub)

        client := ws.NewTestClient("usr_1", "")

        store.On("Connect", ctx, "usr_1").Return(nil)

        h.OnConnect(ctx, client)

        store.AssertExpectations(t)

        // verify broadcast was sent
        select {
        case msg := <-hub.Broadcast:
            assert.True(t, msg.Global)
            assert.Contains(t, string(msg.Payload), "presence.online")
        default:
            t.Fatal("expected broadcast message")
        }
    })
}

func TestPresenceHandler_OnDisconnect_LastConnection(t *testing.T) {
    ctx := context.Background()
    store := mocks.NewPresenceStore(t)
    hub   := ws.NewTestHub()
    h     := handlers.NewPresenceHandler(store, hub)

    client := ws.NewTestClient("usr_1", "")

    // returning 0 means this was the last connection
    store.On("Disconnect", ctx, "usr_1").Return(int64(0), nil)

    h.OnDisconnect(ctx, client)

    store.AssertExpectations(t)

    // verify offline schedule was sent to hub
    select {
    case event := <-hub.ScheduleOfflineCh:
        assert.Equal(t, "usr_1", event.UserID)
    default:
        t.Fatal("expected offline schedule")
    }
}
```

Test helpers in `transport/ws/`:

```go
// transport/ws/testing.go
package ws

// NewTestHub creates a Hub with exported channels for test assertions.
func NewTestHub() *TestHub {
    return &TestHub{
        Broadcast:         make(chan OutboundMessage, 16),
        ScheduleOfflineCh: make(chan OfflineSchedule, 4),
    }
}

type TestHub struct {
    Broadcast         chan OutboundMessage
    ScheduleOfflineCh chan OfflineSchedule
}

func (h *TestHub) ScheduleOffline(userID string, after time.Duration) {
    h.ScheduleOfflineCh <- OfflineSchedule{UserID: userID, After: after}
}

// NewTestClient creates a Client stub for use in handler tests.
func NewTestClient(userID, roomID string) *Client {
    return &Client{
        UserID: userID,
        RoomID: roomID,
        send:   make(chan []byte, 16),
    }
}
```

---

## 16. Adding a New Feature — Step-by-Step Checklist

```
Example: adding "reaction" feature (users react to messages with emoji)

Step 1 — Add message type constants to transport/ws/message.go
         TypeReactionAdd    = "reaction.add"    // client → server
         TypeReactionRemove = "reaction.remove" // client → server
         TypeReactionState  = "reaction.state"  // server → client

Step 2 — Create transport/ws/handlers/reaction.go
         Implement FeatureHandler interface:
           Types()        → []string{TypeReactionAdd, TypeReactionRemove}
           Handle()       → routes to handleAdd / handleRemove
           OnConnect()    → send current reactions snapshot if needed
           OnDisconnect() → nothing for reactions

Step 3 — If persistence needed, add infra/redis/reaction.go or
         use existing feature repo via local interface

Step 4 — Add ReactionHandler field to transport/ws/server.go Handlers struct
         registry.Register(h.Reaction) in New()

Step 5 — Add to bootstrap/wire.go
         wsHandlers.Reaction = handlers.NewReactionHandler(reactionStore, wsHub)

Step 6 — Write handler tests in transport/ws/handlers/reaction_test.go

That is all. hub.go, client.go, registry.go, message.go — none of these change.
```

---

## 17. Rules Cheatsheet

### Hub

```
□ Hub is a router only — never implements feature logic
□ One Hub.Run() goroutine — no locks needed inside the event loop
□ registry.Dispatch() is the only place inbound messages are handled
□ registry.OnConnect/OnDisconnect notify ALL feature handlers
□ Broadcast publishes to Redis AND delivers locally
□ Redis subscription delivers locally only — never re-publish
□ scheduleOffline channel + 3s delay absorbs reconnect window
□ Hub never imports feature packages
```

### Registry

```
□ Duplicate message type registration panics at startup — caught immediately
□ Register() called once per feature handler at server init
□ Unknown message types return TypeSystemError to the client — never silently drop
□ FeatureHandler.Types() drives all routing — single source of truth per feature
```

### Feature Handlers

```
□ Each handler lives in transport/ws/handlers/<feature>.go
□ Implement FeatureHandler interface: Types, Handle, OnConnect, OnDisconnect
□ Use local interfaces for any store or usecase deps — never import infra or feature directly
□ OnDisconnect always uses context.Background() — r.Context() may already be cancelled
□ Cursor state is in-memory only — too ephemeral for Redis
□ Typing state uses Redis SADD + short TTL — auto-expires if client drops without stop
□ Presence state uses Redis INCR/DECR + Lua for atomic delete-on-zero
```

### Message Envelope

```
□ Every message inbound and outbound uses Envelope — no raw strings
□ type field format: "<feature>.<action>" e.g. "presence.ping"
□ Inbound types use imperative: "typing.start"
□ Outbound types use past tense / state: "typing.state"
□ request_id echoed in ack — client can correlate responses
□ error field only set on error envelopes — never on success
```

### Client

```
□ One readPump goroutine + one writePump goroutine per connection
□ readPump owns unregister and conn.Close() on exit
□ writePump exits when send channel closes (hub calls close(c.send))
□ Send() is non-blocking — returns false if buffer full, hub then unregisters
□ SendError() is always non-blocking — fire and forget
□ maxMessageSize enforced in readPump via SetReadLimit
```

### Adding New Features

```
□ Add constants to message.go
□ Create handlers/<feature>.go implementing FeatureHandler
□ Add to Handlers struct in server.go
□ Register in New() via registry.Register(h.Feature)
□ Wire in bootstrap/wire.go
□ hub.go never changes when adding features
□ client.go never changes when adding features
□ registry.go never changes when adding features
```

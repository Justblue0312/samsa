# HTTP-WS Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate HTTP handlers with WebSocket infrastructure to enable real-time notifications after HTTP state changes, fix naming inconsistencies, and establish clean patterns for cross-transport communication.

**Architecture:** 
- Create a `Notifier` service that both HTTP handlers and WS handlers can use
- HTTP handlers call notifier methods after successful operations
- Notifier publishes to WS hub for real-time delivery to connected clients
- Fix interface naming consistency between WS handler and usecase
- Add room-based notification broadcasting for collaborative features

**Tech Stack:** Go 1.25.7, chi/v5, gorilla/websocket, Redis pub/sub for cross-instance messaging

---

## Task 1: Fix Naming Inconsistencies

**Files:**
- Modify: `server/internal/feature/notification/usecase.go:44-48`
- Modify: `server/internal/feature/integrations/ws/notification/ws_handler.go:11-15`

**Step 1: Fix interface naming in usecase**

The WS handler expects `MarkRead` but usecase has `MarkAsRead`. Standardize on `MarkAsRead`:

```go
// In usecase.go - change NotificationSender interface
type NotificationSender interface {
	MarkAsRead(ctx context.Context, userID, notifID uuid.UUID) error
}
```

**Step 2: Update WS handler to match**

```go
// In ws_handler.go - update interface and call
type NotificationSender interface {
	MarkAsRead(ctx context.Context, userID, notifID uuid.UUID) error
}

func (h *NotificationHandler) handleRead(ctx context.Context, client *ws.Client, env *ws.Envelope) {
	// ... existing code ...
	if err := h.usecase.MarkAsRead(ctx, client.UserID, payload.NotifID); err != nil {
		// ... error handling ...
	}
	// ... rest unchanged ...
}
```

**Step 3: Run tests to verify**

```bash
cd /home/justblue/Projects/samsa/server
go test ./internal/feature/notification/... -v
go test ./internal/feature/integrations/ws/notification/... -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add server/internal/feature/notification/usecase.go
git add server/internal/feature/integrations/ws/notification/ws_handler.go
git commit -m "fix: standardize notification MarkAsRead naming across HTTP and WS"
```

---

## Task 2: Create Notifier Service

**Files:**
- Create: `server/internal/feature/notification/notifier.go`
- Modify: `server/internal/feature/notification/usecase.go` (inject notifier)

**Step 1: Create notifier interface and implementation**

```go
// server/internal/feature/notification/notifier.go
package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/justblue/samsa/internal/transport/ws"
)

// Notifier sends real-time notifications to connected WebSocket clients.
// Used by HTTP handlers after state-changing operations.
type Notifier interface {
	// NotifyNew sends a new notification to a specific user
	NotifyNew(ctx context.Context, userID uuid.UUID, notif *NotificationResponse) error
	// NotifyRead broadcasts that a notification was marked as read
	NotifyRead(ctx context.Context, userID, notifID uuid.UUID) error
	// NotifyDeleted broadcasts that a notification was deleted
	NotifyDeleted(ctx context.Context, userID, notifID uuid.UUID) error
}

type notifier struct {
	publisher *ws.Publisher
}

func NewNotifier(publisher *ws.Publisher) Notifier {
	return &notifier{publisher: publisher}
}

func (n *notifier) NotifyNew(ctx context.Context, userID uuid.UUID, notif *NotificationResponse) error {
	msg := Message{
		UserID:         userID,
		NotificationID: notif.Notification.ID,
	}
	for _, recipient := range notif.Notification.Recipients {
		msg.RecipientID = recipient.ID
		pl, _ := json.Marshal(msg)
		env, _ := ws.NewOutbound(ws.TypeNotificationNew, pl)
		b, _ := ws.MarshalEnvelope(env)
		if err := n.publisher.Publish(ctx, uuid.Nil, userID, b); err != nil {
			return err
		}
	}
	return nil
}

func (n *notifier) NotifyRead(ctx context.Context, userID, notifID uuid.UUID) error {
	env, _ := ws.NewOutbound("notification.read_ack", map[string]any{
		"notif_id": notifID,
	})
	b, _ := ws.MarshalEnvelope(env)
	return n.publisher.Publish(ctx, uuid.Nil, userID, b)
}

func (n *notifier) NotifyDeleted(ctx context.Context, userID, notifID uuid.UUID) error {
	env, _ := ws.NewOutbound("notification.deleted", map[string]any{
		"notif_id": notifID,
	})
	b, _ := ws.MarshalEnvelope(env)
	return n.publisher.Publish(ctx, uuid.Nil, userID, b)
}
```

**Step 2: Update usecase to accept notifier**

```go
// In usecase.go - add notifier field and parameter
type usecase struct {
	cfg         *config.Config
	wsPublisher *ws.Publisher
	notifier    Notifier  // Add this
	notiRepo    Repository
	notiRepRepo notificationrecipient.Repository
}

func NewUseCase(
	cfg *config.Config,
	wsPublisher *ws.Publisher,
	notifier Notifier,  // Add this parameter
	notiRepo Repository,
	notiRepRepo notificationrecipient.Repository,
) UseCase {
	return &usecase{
		cfg:         cfg,
		wsPublisher: wsPublisher,
		notifier:    notifier,
		notiRepo:    notiRepo,
		notiRepRepo: notiRepRepo,
	}
}
```

**Step 3: Run tests**

```bash
cd /home/justblue/Projects/samsa/server
go test ./internal/feature/notification/... -v
```

Expected: Compilation errors in tests - will fix in Task 3

**Step 4: Commit**

```bash
git add server/internal/feature/notification/notifier.go
git add server/internal/feature/notification/usecase.go
git commit -m "feat: add Notifier service for HTTP-to-WS real-time updates"
```

---

## Task 3: Integrate Notifier into HTTP Handlers

**Files:**
- Modify: `server/internal/feature/notification/http_handler.go`
- Modify: `server/internal/feature/notification/usecase.go` (use notifier)

**Step 1: Update usecase methods to use notifier**

```go
// In usecase.go - update Create method
func (u *usecase) Create(ctx context.Context, user *sqlc.User, req *CreateNotificationRequest, recipentIds *[]uuid.UUID) (*NotificationResponse, error) {
	// ... existing creation logic ...
	
	// Replace direct wsPublisher calls with notifier
	if u.notifier != nil && recipentIds != nil {
		for _, recipientID := range *recipentIds {
			// Notify each recipient
			_ = u.notifier.NotifyNew(ctx, recipientID, result)
		}
	}
	
	return result, nil
}

// Add new method for HTTP handler to call
func (u *usecase) MarkAsReadWithNotify(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	if err := u.MarkAsRead(ctx, user, notiID); err != nil {
		return err
	}
	if u.notifier != nil {
		_ = u.notifier.NotifyRead(ctx, user.ID, notiID)
	}
	return nil
}
```

**Step 2: Update HTTP handler to use new method**

```go
// In http_handler.go - update MarkAsRead handler
func (h *HTTPHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	idStr := chi.URLParam(r, "notification_id")
	notiID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest(err.Error()))
		return
	}

	// Use new method that also notifies WS clients
	err = h.u.MarkAsReadWithNotify(ctx, sub.User, notiID)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusOK, map[string]string{"message": "notification marked as read"})
}
```

**Step 3: Run tests**

```bash
cd /home/justblue/Projects/samsa/server
go test ./internal/feature/notification/... -v
go build ./...
```

Expected: PASS

**Step 4: Commit**

```bash
git add server/internal/feature/notification/http_handler.go
git add server/internal/feature/notification/usecase.go
git commit -m "feat: integrate Notifier into HTTP handlers for real-time WS updates"
```

---

## Task 4: Add Room-Based Notification Broadcasting

**Files:**
- Modify: `server/internal/transport/ws/hub.go`
- Modify: `server/internal/feature/notification/notifier.go`

**Step 1: Add room-based publish method to Publisher**

```go
// In publisher.go - add room-scoped method
func (p *Publisher) PublishToRoom(ctx context.Context, roomID uuid.UUID, payload []byte) error {
	select {
	case p.hub.Broadcast <- OutboundMessage{
		RoomID:  roomID,
		Global:  false,
		Payload: payload,
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

**Step 2: Add room notification method to Notifier**

```go
// In notifier.go - add room method
func (n *notifier) NotifyRoom(ctx context.Context, roomID uuid.UUID, notifType string, payload any) error {
	env, _ := ws.NewOutbound(notifType, payload)
	b, _ := ws.MarshalEnvelope(env)
	return n.publisher.PublishToRoom(ctx, roomID, b)
}
```

**Step 3: Run tests**

```bash
cd /home/justblue/Projects/samsa/server
go test ./internal/transport/ws/... -v
go test ./internal/feature/notification/... -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add server/internal/transport/ws/publisher.go
git add server/internal/feature/notification/notifier.go
git commit -m "feat: add room-based notification broadcasting"
```

---

## Task 5: Fix Typo and Update Bootstrap

**Files:**
- Rename: `server/internal/feature/integrations/ws/presense/` → `presence/`
- Modify: `server/bootstrap/server.go` (or wherever DI container is)

**Step 1: Rename directory**

```bash
cd /home/justblue/Projects/samsa/server
git mv internal/feature/integrations/ws/presense internal/feature/integrations/ws/presence
```

**Step 2: Update all imports**

```bash
# Find and replace imports
grep -r "presense" server/ --include="*.go" -l
# Update each file's import from:
#   "github.com/justblue/samsa/internal/feature/integrations/ws/presense"
# to:
#   "github.com/justblue/samsa/internal/feature/integrations/ws/presence"
```

**Step 3: Update bootstrap to wire Notifier**

```go
// In bootstrap/server.go or wherever DI happens
notifier := notification.NewNotifier(wsPublisher)
notificationUseCase := notification.NewUseCase(cfg, wsPublisher, notifier, notiRepo, notiRepRepo)
```

**Step 4: Run full build and tests**

```bash
cd /home/justblue/Projects/samsa/server
go build ./...
go test ./... -v
task lint
```

Expected: PASS, no lint errors

**Step 5: Commit**

```bash
git add server/internal/feature/integrations/ws/presence/
git add server/bootstrap/server.go
git commit -m "fix: rename presense to presence and wire Notifier in bootstrap"
```

---

## Task 6: Add Documentation

**Files:**
- Create: `server/internal/feature/notification/README.md`

**Step 1: Create documentation**

```markdown
# Notification Feature - HTTP/WS Integration

## Architecture

The notification feature supports both HTTP REST API and real-time WebSocket updates:

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│ HTTP Client │────▶│ HTTP Handler │────▶│  UseCase    │
└─────────────┘     └──────────────┘     └─────────────┘
                                              │
                                              ▼
                                         ┌─────────────┐
                                         │  Notifier   │
                                         └─────────────┘
                                              │
                                              ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  WS Client  │◀────│  WS Handler  │◀────│   Hub       │
└─────────────┘     └──────────────┘     └─────────────┘
```

## Usage

### HTTP Endpoints

```http
GET    /notifications          # List all notifications
GET    /notifications/unread   # Get unread count
GET    /notifications/:id      # Get by ID
POST   /notifications/:id/read # Mark as read (triggers WS update)
DELETE /notifications/:id      # Delete notification
```

### WebSocket Messages

**Client → Server:**
```json
{
  "type": "notification.read",
  "request_id": "req-123",
  "payload": {
    "notif_id": "uuid-here"
  }
}
```

**Server → Client:**
```json
{
  "type": "notification.new",
  "payload": {
    "user_id": "uuid",
    "notification_id": "uuid",
    "recipient_id": "uuid"
  },
  "ts": "2026-03-03T..."
}
```

## Real-Time Updates

When a notification is created/updated/deleted via HTTP:
1. HTTP handler processes the request
2. UseCase performs database operation
3. Notifier publishes to WebSocket Hub
4. All connected clients receive real-time update

## Testing

```bash
go test ./internal/feature/notification/... -v
go test ./internal/feature/integrations/ws/notification/... -v
```
```

**Step 2: Commit**

```bash
git add server/internal/feature/notification/README.md
git commit -m "docs: add HTTP/WS integration documentation for notifications"
```

---

## Verification

After all tasks complete:

```bash
cd /home/justblue/Projects/samsa/server

# Build
task build

# Test
task test

# Lint
task lint

# Run and manually test:
# 1. Start server: task run
# 2. Connect WS client
# 3. Make HTTP request to create notification
# 4. Verify WS client receives real-time update
```

---

## Summary

This plan establishes a clean pattern for HTTP-WS integration:
- **Notifier service** abstracts WS publishing from business logic
- **HTTP handlers** trigger real-time updates after state changes
- **WS handlers** remain focused on real-time interactions
- **Room-based broadcasting** for collaborative features
- **Consistent naming** across all interfaces

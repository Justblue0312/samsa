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

## Components

### Notifier Service

The `Notifier` interface abstracts WebSocket publishing from business logic:

```go
type Notifier interface {
    NotifyNew(ctx context.Context, userID uuid.UUID, notif *NotificationResponse) error
    NotifyRead(ctx context.Context, userID, notifID uuid.UUID) error
    NotifyDeleted(ctx context.Context, userID, notifID uuid.UUID) error
    NotifyRoom(ctx context.Context, roomID uuid.UUID, notifType string, payload any) error
}
```

### Room-Based Broadcasting

Use `NotifyRoom` to send notifications to all clients in a specific room:

```go
notifier.NotifyRoom(ctx, storyID, "story.update", map[string]any{
    "story_id": storyID,
    "action":   "published",
})
```

## Testing

```bash
go test ./internal/feature/notification/... -v
go test ./internal/feature/integrations/ws/notification/... -v
```

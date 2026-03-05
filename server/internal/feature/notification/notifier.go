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
	// NotifyRoom sends a notification to all clients in a room
	NotifyRoom(ctx context.Context, roomID uuid.UUID, notifType string, payload any) error
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

func (n *notifier) NotifyRoom(ctx context.Context, roomID uuid.UUID, notifType string, payload any) error {
	env, _ := ws.NewOutbound(notifType, payload)
	b, _ := ws.MarshalEnvelope(env)
	return n.publisher.PublishToRoom(ctx, roomID, b)
}

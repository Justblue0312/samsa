package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/transport/ws"
)

// NotificationSender is the local interface for marking notifications read.
type NotificationSender interface {
	MarkAsRead(ctx context.Context, user *sqlc.User, notifID uuid.UUID) error
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
	if env.Type == ws.TypeNotificationRead {
		h.handleRead(ctx, client, env)
	}
}

func (h *NotificationHandler) handleRead(ctx context.Context, client *ws.Client, env *ws.Envelope) {
	var payload struct {
		NotifID uuid.UUID `json:"notif_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		client.SendError(env.RequestID, "BAD_PAYLOAD", "invalid notification read payload")
		return
	}

	user := &sqlc.User{ID: client.UserID}
	if err := h.usecase.MarkAsRead(ctx, user, payload.NotifID); err != nil {
		client.SendError(env.RequestID, "MARK_READ_FAILED", err.Error())
		return
	}

	ack, _ := ws.NewOutbound(ws.TypeSystemAck, map[string]any{
		"request_id": env.RequestID,
		"notif_id":   payload.NotifID,
	})
	_ = client.SendEnvelope(ack)
}

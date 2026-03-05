package presence

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/internal/transport/ws"
)

// PresenceStore is the local interface — handler never imports infra/redis
type PresenceStore interface {
	Connect(ctx context.Context, userID uuid.UUID) error
	Disconnect(ctx context.Context, userID uuid.UUID) (remaining int64, err error)
	Refresh(ctx context.Context, userID uuid.UUID) error
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
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

	env, _ := ws.NewOutbound(ws.TypePresenceOnline, map[string]any{
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
		UserID uuid.UUID `json:"user_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}

	online, err := h.store.IsOnline(ctx, payload.UserID)
	if err != nil || online {
		return // still online or error — do not emit offline
	}

	offlineEnv, _ := ws.NewOutbound(ws.TypePresenceOffline, map[string]any{
		"user_id": payload.UserID,
	})
	b, _ := ws.MarshalEnvelope(offlineEnv)

	// broadcast offline to everyone
	h.hub.Broadcast <- ws.OutboundMessage{Global: true, Payload: b}
}

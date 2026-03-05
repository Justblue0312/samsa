package typing

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/internal/transport/ws"
)

const typingTTL = 5 * time.Second // auto-expire typing state if no update

// TypingStore is the local interface for ephemeral typing state.
type TypingStore interface {
	SetTyping(ctx context.Context, roomID, userID uuid.UUID) error
	ClearTyping(ctx context.Context, roomID, userID uuid.UUID) error
	GetTypingUsers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error)
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
	if client.RoomID != uuid.Nil {
		_ = h.store.ClearTyping(context.Background(), client.RoomID, client.UserID)
		h.broadcastTypingState(context.Background(), client.RoomID)
	}
}

func (h *TypingHandler) Handle(ctx context.Context, client *ws.Client, env *ws.Envelope) {
	if client.RoomID == uuid.Nil {
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

func (h *TypingHandler) broadcastTypingState(ctx context.Context, roomID uuid.UUID) {
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

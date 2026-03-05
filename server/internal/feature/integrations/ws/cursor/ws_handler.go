package cursor

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/justblue/samsa/internal/transport/ws"
)

type CursorPosition struct {
	UserID uuid.UUID `json:"user_id"`
	X      float64   `json:"x"`
	Y      float64   `json:"y"`
	PageID string    `json:"page_id,omitempty"`
}

// CursorHandler tracks cursor positions in memory — no Redis, too ephemeral.
// Positions are stored per room per user. Cleared on disconnect.
type CursorHandler struct {
	hub       *ws.Hub
	mu        sync.RWMutex
	positions map[uuid.UUID]map[uuid.UUID]CursorPosition // roomID → userID → position
}

func NewCursorHandler(hub *ws.Hub) *CursorHandler {
	return &CursorHandler{
		hub:       hub,
		positions: make(map[uuid.UUID]map[uuid.UUID]CursorPosition),
	}
}

func (h *CursorHandler) Types() []string {
	return []string{ws.TypeCursorMove}
}

func (h *CursorHandler) OnConnect(ctx context.Context, client *ws.Client) {
	if client.RoomID == uuid.Nil {
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
	if client.RoomID == uuid.Nil {
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
	if client.RoomID == uuid.Nil {
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
		h.positions[client.RoomID] = make(map[uuid.UUID]CursorPosition)
	}
	h.positions[client.RoomID][client.UserID] = pos
	h.mu.Unlock()

	h.broadcastCursorState(ctx, client.RoomID)
}

func (h *CursorHandler) broadcastCursorState(ctx context.Context, roomID uuid.UUID) {
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

func (h *CursorHandler) snapshotRoom(roomID uuid.UUID) []CursorPosition {
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

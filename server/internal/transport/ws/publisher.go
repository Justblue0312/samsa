package ws

import (
	"context"

	"github.com/google/uuid"
)

// Publisher wraps Hub and satisfies any feature's wsPublisher local interface.
type Publisher struct{ hub *Hub }

func NewPublisher(hub *Hub) *Publisher { return &Publisher{hub: hub} }

// Publish sends a message to a specific user, a room, or globally.
func (p *Publisher) Publish(ctx context.Context, roomID, userID uuid.UUID, payload []byte) error {
	select {
	case p.hub.Broadcast <- OutboundMessage{
		RoomID:  roomID,
		UserID:  userID,
		Global:  roomID == uuid.Nil && userID == uuid.Nil,
		Payload: payload,
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishToRoom sends a message to all clients in a specific room.
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

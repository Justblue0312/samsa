package ws

import "github.com/google/uuid"

// Room is a scoped group of clients (e.g. per projectID, per chatID).
// The Hub manages rooms — rooms themselves are just client sets.
type Room struct {
	ID      uuid.UUID
	clients map[*Client]struct{}
}

func newRoom(id uuid.UUID) *Room {
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

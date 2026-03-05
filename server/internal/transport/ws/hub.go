package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// OutboundMessage is what the hub fans out to connected clients.
type OutboundMessage struct {
	RoomID  uuid.UUID // empty = not scoped to a room
	UserID  uuid.UUID // empty = all clients in room; set to target one user
	Global  bool      // true = send to all connected clients regardless of room
	Payload []byte    // pre-marshalled Envelope
}

// OfflineSchedule tells the hub to check and emit offline after a delay.
type OfflineSchedule struct {
	UserID uuid.UUID
	After  time.Duration
}

type Hub struct {
	register        chan *Client
	unregister      chan *Client
	inbound         chan InboundMessage
	Broadcast       chan OutboundMessage
	scheduleOffline chan OfflineSchedule

	rooms   map[uuid.UUID]*Room
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
		rooms:           make(map[uuid.UUID]*Room),
		globals:         make(map[*Client]struct{}),
		registry:        registry,
		rdb:             rdb,
	}
}

// ScheduleOffline schedules a check for user offline status after a delay.
func (h *Hub) ScheduleOffline(userID uuid.UUID, after time.Duration) {
	h.scheduleOffline <- OfflineSchedule{UserID: userID, After: after}
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
	if c.RoomID == uuid.Nil {
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
	delete(h.globals, c)
	if c.RoomID != uuid.Nil {
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
	case msg.RoomID != uuid.Nil:
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
		if msg.UserID != uuid.Nil && c.UserID != msg.UserID {
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
			Payload: mustMarshal(map[string]any{"user_id": event.UserID}),
		})
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

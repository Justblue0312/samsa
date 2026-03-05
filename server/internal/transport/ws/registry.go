package ws

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// FeatureHandler handles all inbound messages for a specific feature.
// Each feature in transport/ws/handlers/ implements this interface.
type FeatureHandler interface {
	// Types returns the list of message types this handler owns.
	// e.g. []string{"presence.ping", "presence.status"}
	Types() []string

	// Handle is called by the hub for every inbound message whose type
	// matches one of the types returned by Types().
	Handle(ctx context.Context, client *Client, env *Envelope)

	// OnConnect is called when a client first connects.
	// Use for initialising per-client state (e.g. send initial presence snapshot).
	OnConnect(ctx context.Context, client *Client)

	// OnDisconnect is called when a client disconnects.
	// Use for cleanup (e.g. decrement presence counter, broadcast offline).
	OnDisconnect(ctx context.Context, client *Client)
}

// Registry holds all registered feature handlers and dispatches inbound messages.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]FeatureHandler // msgType → handler
	all      []FeatureHandler          // for OnConnect/OnDisconnect iteration
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]FeatureHandler),
	}
}

// Register adds a feature handler. Called once at startup from server.go.
func (r *Registry) Register(h FeatureHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range h.Types() {
		if existing, ok := r.handlers[t]; ok {
			// two handlers claiming the same type is a programmer error
			panic("ws: duplicate handler for type " + t + " (already registered by " +
				fmt.Sprintf("%T", existing) + ")")
		}
		r.handlers[t] = h
	}
	r.all = append(r.all, h)
}

// Dispatch routes an inbound message to the correct feature handler.
func (r *Registry) Dispatch(ctx context.Context, client *Client, env *Envelope) {
	r.mu.RLock()
	h, ok := r.handlers[env.Type]
	r.mu.RUnlock()

	if !ok {
		slog.Warn("ws: no handler for message type", "type", env.Type, "userID", client.UserID)
		// send error back to the specific client
		client.SendError(env.RequestID, "UNKNOWN_TYPE",
			"no handler registered for type: "+env.Type)
		return
	}

	h.Handle(ctx, client, env)
}

// OnConnect notifies all registered handlers that a client connected.
func (r *Registry) OnConnect(ctx context.Context, client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.all {
		h.OnConnect(ctx, client)
	}
}

// OnDisconnect notifies all registered handlers that a client disconnected.
func (r *Registry) OnDisconnect(ctx context.Context, client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.all {
		h.OnDisconnect(ctx, client)
	}
}

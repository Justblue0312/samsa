package ws

// WSHandlers holds all registered WebSocket feature handlers.
// Handlers are created by bootstrap/deps and registered into the Registry via Register.
type WSHandlers struct {
	handlers []FeatureHandler
}

// Add appends one or more feature handlers to the list.
func (h *WSHandlers) Add(handlers ...FeatureHandler) {
	h.handlers = append(h.handlers, handlers...)
}

// Register registers all held handlers into the given Registry.
// Call this once at startup after all handlers have been added via Add.
func (h *WSHandlers) Register(r *Registry) {
	for _, handler := range h.handlers {
		r.Register(handler)
	}
}

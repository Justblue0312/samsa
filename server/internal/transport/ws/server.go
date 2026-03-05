package ws

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // restrict in production
	},
}

type Server struct {
	hub      *Hub
	registry *Registry
	cfg      *config.Config
}

func New(cfg *config.Config, hub *Hub, registry *Registry) *Server {
	return &Server{hub: hub, registry: registry, cfg: cfg}
}

// RegisterRoutes mounts WS endpoints onto the chi router.
// Called from transport/http/server.go.
func (s *Server) RegisterRoutes(r chi.Router) {
	// global connection (no room)
	r.Get("/ws", s.handleUpgrade(uuid.Nil))

	// room-scoped connection
	r.Get("/ws/room/{roomID}", func(w http.ResponseWriter, r *http.Request) {
		roomIDParam := chi.URLParam(r, "roomID")
		roomID, err := uuid.Parse(roomIDParam)
		if err != nil {
			return
		}
		s.handleUpgrade(roomID)(w, r)
	})
}

func (s *Server) handleUpgrade(roomID uuid.UUID) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		sub, err := common.GetUserSubject(ctx)
		if err != nil {
			conn.Close()
			return
		}
		userID := sub.User.ID
		client := NewClient(s.hub, conn, userID, roomID)
		client.Start()
	}
}

// Start runs the hub event loop. Called from bootstrap/server.go in an errgroup.
func (s *Server) Start(ctx context.Context) error {
	s.hub.Run(ctx)
	return nil
}

func (s *Server) Stop() {
	// hub stops when ctx is cancelled — no explicit stop needed
}

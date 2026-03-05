package ws

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // 54s
	maxMessageSize = 4096
)

// InboundMessage is a raw message received from a client before parsing.
type InboundMessage struct {
	Client *Client
	Data   []byte
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	UserID uuid.UUID
	RoomID uuid.UUID
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, roomID uuid.UUID) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		UserID: userID,
		RoomID: roomID,
	}
}

// Start registers the client with the hub and launches pumps.
func (c *Client) Start() {
	c.hub.register <- c
	go c.writePump()
	go c.readPump()
}

// Send queues a pre-marshalled message for delivery. Non-blocking.
// Returns false if the buffer is full.
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// SendEnvelope marshals and queues an envelope.
func (c *Client) SendEnvelope(env *Envelope) error {
	b, err := MarshalEnvelope(env)
	if err != nil {
		return err
	}
	if !c.Send(b) {
		return fmt.Errorf("ws: send buffer full for user %s", c.UserID)
	}
	return nil
}

// SendError sends a typed error back to this specific client.
func (c *Client) SendError(requestID, code, msg string) {
	env := NewErrorOutbound(requestID, code, msg)
	b, _ := MarshalEnvelope(env)
	c.Send(b)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		// hub registry handles TTL refresh via OnPong if needed
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Warn("ws: unexpected close", "userID", c.UserID, "error", err)
			}
			return
		}
		c.hub.inbound <- InboundMessage{Client: c, Data: data}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				slog.Warn("ws: write error", "userID", c.UserID, "error", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

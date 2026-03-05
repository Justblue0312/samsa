package ws

import (
	"encoding/json"
	"fmt"
	"time"
)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"ts"`
	Error     *EnvelopeError  `json:"error,omitempty"`
}

type EnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ── Message type constants ────────────────────────────────────────────────────
// Convention: "<feature>.<action>"
// Inbound types (client → server) use imperative form: "presence.ping"
// Outbound types (server → client) use past tense:    "presence.online"

const (
	// system
	TypeSystemError = "system.error"
	TypeSystemAck   = "system.ack"

	// presence (feature 1)
	TypePresencePing    = "presence.ping"    // client → server
	TypePresenceOnline  = "presence.online"  // server → client
	TypePresenceOffline = "presence.offline" // server → client

	// notifications (feature 2)
	TypeNotificationNew  = "notification.new"  // server → client
	TypeNotificationRead = "notification.read" // client → server (mark read)

	// typing (feature 3)
	TypeTypingStart = "typing.start" // client → server
	TypeTypingStop  = "typing.stop"  // client → server
	TypeTypingState = "typing.state" // server → client (broadcast to room)

	// cursor (feature 4)
	TypeCursorMove  = "cursor.move"  // client → server
	TypeCursorState = "cursor.state" // server → client (broadcast to room)
)

// NewOutbound builds a server→client envelope.
func NewOutbound(msgType string, payload any) (*Envelope, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ws: marshal outbound payload: %w", err)
	}
	return &Envelope{
		Type:      msgType,
		Payload:   b,
		Timestamp: time.Now(),
	}, nil
}

// NewErrorOutbound builds an error envelope to send to a specific client.
func NewErrorOutbound(requestID, code, msg string) *Envelope {
	return &Envelope{
		Type:      TypeSystemError,
		RequestID: requestID,
		Timestamp: time.Now(),
		Error:     &EnvelopeError{Code: code, Message: msg},
	}
}

// MarshalEnvelope serialises an envelope to bytes for the wire.
func MarshalEnvelope(e *Envelope) ([]byte, error) {
	return json.Marshal(e)
}

// ParseEnvelope deserialises bytes from the wire into an envelope.
func ParseEnvelope(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("ws: parse envelope: %w", err)
	}
	return &e, nil
}

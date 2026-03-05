package redis

// Channel name conventions for cross-instance WS broadcasting.
const (
	ChanWSBroadcast = "ws:broadcast" // global broadcast
	ChanWSRoom      = "ws:room:%s"   // per-room broadcast, format with roomID
	ChanWSUser      = "ws:user:%s"   // per-user, format with userID
)

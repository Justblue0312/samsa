package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const presenceTTL = 90 * time.Second

type PresenceStore struct {
	rdb *redis.Client
}

func NewPresenceStore(rdb *redis.Client) *PresenceStore {
	return &PresenceStore{rdb: rdb}
}

func presenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("ws:presence:%s", userID)
}

// Connect increments the connection counter and sets TTL.
// Safe to call from multiple goroutines — Redis INCR is atomic.
func (p *PresenceStore) Connect(ctx context.Context, userID uuid.UUID) error {
	key := presenceKey(userID)
	pipe := p.rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, presenceTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// Disconnect decrements the counter.
// If counter reaches 0 or below, deletes the key immediately.
// Returns the remaining connection count.
func (p *PresenceStore) Disconnect(ctx context.Context, userID uuid.UUID) (int64, error) {
	key := presenceKey(userID)

	script := redis.NewScript(`
        local count = redis.call('DECR', KEYS[1])
        if count <= 0 then
            redis.call('DEL', KEYS[1])
            return 0
        end
        return count
    `)
	result, err := script.Run(ctx, p.rdb, []string{key}).Int64()
	if err != nil && err != redis.Nil {
		return 0, err
	}
	return result, nil
}

// Refresh extends the TTL — called on every successful pong.
// Does nothing if the key doesn't exist (connection already cleaned up).
func (p *PresenceStore) Refresh(ctx context.Context, userID uuid.UUID) error {
	return p.rdb.Expire(ctx, presenceKey(userID), presenceTTL).Err()
}

// IsOnline returns true if the user has at least one active connection.
func (p *PresenceStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	count, err := p.rdb.Get(ctx, presenceKey(userID)).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

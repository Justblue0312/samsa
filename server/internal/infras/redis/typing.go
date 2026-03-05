package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const typingTTL = 5 * time.Second

type TypingStore struct {
	rdb *redis.Client
}

func NewTypingStore(rdb *redis.Client) *TypingStore {
	return &TypingStore{rdb: rdb}
}

func typingKey(roomID uuid.UUID) string {
	return fmt.Sprintf("ws:typing:%s", roomID)
}

func (s *TypingStore) SetTyping(ctx context.Context, roomID, userID uuid.UUID) error {
	key := typingKey(roomID)
	pipe := s.rdb.Pipeline()
	pipe.SAdd(ctx, key, userID.String())
	pipe.Expire(ctx, key, typingTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *TypingStore) ClearTyping(ctx context.Context, roomID, userID uuid.UUID) error {
	return s.rdb.SRem(ctx, typingKey(roomID), userID.String()).Err()
}

func (s *TypingStore) GetTypingUsers(ctx context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	members, err := s.rdb.SMembers(ctx, typingKey(roomID)).Result()
	if err != nil {
		return nil, err
	}

	users := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		id, err := uuid.Parse(m)
		if err != nil {
			continue
		}
		users = append(users, id)
	}
	return users, nil
}

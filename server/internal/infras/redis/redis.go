package redis

import (
	"context"

	"github.com/justblue/samsa/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisOpts(c *config.Config) *redis.Options {
	return &redis.Options{
		Addr:     c.Redis.Host + ":" + c.Redis.Port,
		Password: c.Redis.Pwd,
		DB:       c.Redis.DB,
	}
}

func New(ctx context.Context, opts *redis.Options) (*redis.Client, error) {
	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

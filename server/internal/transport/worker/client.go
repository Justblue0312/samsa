package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/justblue/samsa/config"
)

type Client struct {
	client *asynq.Client
}

// NewClient creates a new Client with the given Redis options.
func NewClient(cfg *config.Config) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Pwd,
	}
	return &Client{
		client: asynq.NewClient(redisOpt),
	}
}

func (c *Client) Enqueue(ctx context.Context, task *TaskDefinition, payload any, opts ...asynq.Option) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for task %q: %w", task.Type, err)
	}

	// Combine default options from TaskDefinition with runtime options.
	mergedOpts := append(task.Options, opts...)

	t := asynq.NewTask(task.Type, jsonPayload, mergedOpts...)
	info, err := c.client.EnqueueContext(ctx, t)
	if err != nil {
		return fmt.Errorf("failed to enqueue task %q: %w", task.Type, err)
	}

	slog.Info("task enqueued",
		slog.String("type", task.Type),
		slog.String("id", info.ID),
		slog.String("queue", info.Queue),
	)

	return nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

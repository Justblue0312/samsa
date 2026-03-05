package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/justblue/samsa/config"
)

const (
	// QueueHigh is the name for the high priority queue.
	QueueHigh = "high"
	// QueueDefault is the name for the default priority queue.
	QueueDefault = "default"
	// QueueLow is the name for the low priority queue.
	QueueLow = "low"
)

// Server handles task processing and scheduling.
type Server struct {
	cfg *config.Config

	server    *asynq.Server
	scheduler *asynq.Scheduler

	tasks []*TaskDefinition
}

// NewServer creates a new worker Server.
func NewServer(cfg *config.Config) (*Server, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Pwd,
	}

	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				QueueHigh:    10,
				QueueDefault: 5,
				QueueLow:     1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("process task failed",
					slog.String("type", task.Type()),
					slog.String("payload", string(task.Payload())),
					slog.String("error", err.Error()),
				)
			}),
		},
	)

	loc, err := time.LoadLocation(cfg.Location)
	if err != nil {
		return nil, fmt.Errorf("invalid time location %q: %w", cfg.Location, err)
	}

	scheduler := asynq.NewScheduler(
		redisOpt,
		&asynq.SchedulerOpts{
			Location: loc,
		},
	)

	return &Server{
		server:    srv,
		scheduler: scheduler,
		tasks:     make([]*TaskDefinition, 0),
	}, nil
}

// Register registers tasks with the server.
func (s *Server) Register(tasks ...*TaskDefinition) {
	for _, t := range tasks {
		s.tasks = append(s.tasks, t)
		slog.Info("registered definition", slog.String("type", t.Type))
	}
}

// Start begins processing tasks and running the scheduler.
func (s *Server) Start() error {
	mux := asynq.NewServeMux()

	for _, task := range s.tasks {
		if task.Schedule != "" {
			// Register scheduled task.
			// The task payload for scheduled tasks is usually nil or needs to be handled specifically.
			entryID, err := s.scheduler.Register(task.Schedule, asynq.NewTask(task.Type, nil), task.Options...)
			if err != nil {
				slog.Error("failed to register scheduled task",
					slog.String("type", task.Type), slog.String("schedule", task.Schedule), slog.String("error", err.Error()))
				continue
			}
			slog.Info("scheduled task",
				slog.String("type", task.Type),
				slog.String("schedule", task.Schedule),
				slog.String("entry_id", entryID),
			)
		}

		// Register task handler.
		mux.HandleFunc(task.Type, task.Handler)
		slog.Info("registered handler", slog.String("type", task.Type))
	}

	// Start scheduler in a separate goroutine.
	go func() {
		if err := s.scheduler.Run(); err != nil {
			slog.Error("failed to start scheduler", slog.String("error", err.Error()))
		}
	}()

	return s.server.Start(mux)
}

// Close gracefully stops the server and scheduler.
func (s *Server) Close() {
	s.scheduler.Shutdown()
	s.server.Shutdown()
}

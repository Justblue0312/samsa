package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/transport/worker"
)

type TaskHandler struct {
	q *sqlc.Queries
}

func NewTaskHandler(q *sqlc.Queries) *TaskHandler {
	return &TaskHandler{
		q: q,
	}
}

func (t *TaskHandler) RegisterTasks() []*worker.TaskDefinition {
	return []*worker.TaskDefinition{
		{
			Type:    "task:user:on_after_signup",
			Handler: t.OnAfterSignUp,
			Options: []asynq.Option{asynq.Queue(worker.QueueLow)},
		},
	}
}

type OnAfterSignUpPayload struct {
	UserID uuid.UUID
}

func (t *TaskHandler) OnAfterSignUp(ctx context.Context, task *asynq.Task) error {
	var payload OnAfterSignUpPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	_, err := t.q.GetUserByID(ctx, sqlc.GetUserByIDParams{
		UserID:    payload.UserID,
		IsDeleted: false,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("The user with id %s does not exist.", payload.UserID.String())
		}
		return fmt.Errorf("failed to get user by id: %w", err)
	}

	return nil
}

func NewTaskOnAfterSignUp() *worker.TaskDefinition {
	return &worker.TaskDefinition{
		Type:    "task:auth:send_email",
		Options: []asynq.Option{asynq.Queue(worker.QueueLow)},
	}
}

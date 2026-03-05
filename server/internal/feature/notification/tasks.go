package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/transport/worker"
)

const (
	taskMaxRetries           = 3
	TaskTypeSendNotification = "task:notification:send"
)

type TaskHandler struct {
	notiUsecase UseCase
}

func NewTaskHandler(notiUsecase UseCase) *TaskHandler {
	return &TaskHandler{
		notiUsecase: notiUsecase,
	}
}

func (t *TaskHandler) RegisterTasks() []*worker.TaskDefinition {
	return []*worker.TaskDefinition{
		{
			Type:    TaskTypeSendNotification,
			Handler: t.SendNotification,
			Options: []asynq.Option{asynq.Queue(worker.QueueDefault)},
		},
	}
}

type SendNotificationPayload struct {
	User          sqlc.User
	CreateNotiReq CreateNotificationRequest
	RecipentIds   *[]uuid.UUID
}

// SendNotification will send a notification to the users through background worker.
func (t *TaskHandler) SendNotification(ctx context.Context, task *asynq.Task) error {
	var payload SendNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	_, err := t.notiUsecase.Create(ctx, &payload.User, &payload.CreateNotiReq, payload.RecipentIds)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// Task definition for sending email in notification feature

func NewTaskSendNotification(payload *SendNotificationPayload) *worker.TaskDefinition {
	return &worker.TaskDefinition{
		Type:    TaskTypeSendNotification,
		Options: []asynq.Option{asynq.MaxRetry(taskMaxRetries), asynq.Queue(worker.QueueDefault)},
	}
}

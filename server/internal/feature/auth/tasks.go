package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/justblue/samsa/internal/infras/email"
	"github.com/justblue/samsa/internal/transport/worker"
	"github.com/pkg/errors"
)

const (
	taskMaxRetries    = 3
	TaskTypeSendEmail = "task:auth:send_email"
)

type TaskHandler struct {
	emailClient *email.Client
}

func NewTaskHandler(emailClient *email.Client) *TaskHandler {
	return &TaskHandler{
		emailClient: emailClient,
	}
}

func (t *TaskHandler) RegisterTasks() []*worker.TaskDefinition {
	return []*worker.TaskDefinition{
		{
			Type:    TaskTypeSendEmail,
			Handler: t.HandleSendEmail,
			Options: []asynq.Option{asynq.MaxRetry(taskMaxRetries), asynq.Queue(worker.QueueDefault)},
		},
	}
}

type SendMailPayload struct {
	Email        string            `json:"email"`
	TemplateName string            `json:"template_name"`
	Subject      string            `json:"subject"`
	Metadata     map[string]string `json:"metadata"`
}

func (t *TaskHandler) HandleSendEmail(_ context.Context, task *asynq.Task) error {
	var payload SendMailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	msg := email.NewTemplateMessage(
		[]string{payload.Email},
		payload.Subject,
		payload.TemplateName,
		payload.Metadata,
	)

	err := t.emailClient.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send email")
	}
	return nil
}

// Task definition for sending email in auth feature

func NewTaskSendEmailDefinition() *worker.TaskDefinition {
	return &worker.TaskDefinition{
		Type:    "task:auth:send_email",
		Options: []asynq.Option{asynq.MaxRetry(taskMaxRetries), asynq.Queue(worker.QueueDefault)},
	}
}

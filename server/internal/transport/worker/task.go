package worker

import "github.com/hibiken/asynq"

// TaskDefinition defines a task with its type, handler, optional schedule, and processing options.
type TaskDefinition struct {
	// Type is the unique identifier for the task type.
	Type string

	// Handler is the function that processes the task.
	Handler asynq.HandlerFunc

	// Schedule is an optional cron specification for scheduled tasks (e.g., "@daily", "0 12 * * *").
	// If empty, the task is not scheduled automatically.
	Schedule string

	// Options are the default options applied when enqueuing or registering the task.
	Options []asynq.Option
}

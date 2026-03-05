package submission

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/notification"
	"github.com/justblue/samsa/internal/feature/submission_status_history"
	"github.com/justblue/samsa/internal/transport/worker"
)

const (
	// TaskSubmissionTimeout auto-timeouts pending submissions idle for 30+ days.
	TaskSubmissionTimeout = "submission:timeout"

	// TaskSubmissionAutoArchive auto-archives terminal submissions idle for 15+ days.
	TaskSubmissionAutoArchive = "submission:auto_archive"

	// DefaultTimeoutDays is the number of inactivity days before a submission is timeouted.
	DefaultTimeoutDays = 30

	// DefaultArchiveDays is the number of inactivity days before a terminal submission is archived.
	DefaultArchiveDays = 15
)

// TaskHandler handles submission background tasks.
type TaskHandler struct {
	repo        Repository
	historyRepo submission_status_history.Repository
	notifRepo   notification.Repository
}

// NewTaskHandler creates a new submission task handler.
func NewTaskHandler(repo Repository, historyRepo submission_status_history.Repository, notifRepo notification.Repository) *TaskHandler {
	return &TaskHandler{
		repo:        repo,
		historyRepo: historyRepo,
		notifRepo:   notifRepo,
	}
}

// RegisterTasks returns all task definitions for the submission domain.
func (h *TaskHandler) RegisterTasks() []*worker.TaskDefinition {
	deps := TaskDeps{Repo: h.repo, HistoryRepo: h.historyRepo, NotifRepo: h.notifRepo}
	return []*worker.TaskDefinition{
		timeoutTaskDefinition(deps),
		autoArchiveTaskDefinition(deps),
	}
}

// TaskDeps groups dependencies shared across task handlers.
type TaskDeps struct {
	Repo        Repository
	HistoryRepo submission_status_history.Repository
	NotifRepo   notification.Repository
}

// ── Timeout task ─────────────────────────────────────────────────────────────

func timeoutTaskDefinition(deps TaskDeps) *worker.TaskDefinition {
	return &worker.TaskDefinition{
		Type:     TaskSubmissionTimeout,
		Handler:  handleSubmissionTimeout(deps),
		Schedule: "0 2 * * *", // Daily at 02:00 AM
		Options: []asynq.Option{
			asynq.Queue(worker.QueueLow),
			asynq.MaxRetry(3),
			asynq.Timeout(10 * time.Minute),
		},
	}
}

func handleSubmissionTimeout(deps TaskDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		slog.Info("processing submission timeout task")

		candidates, err := deps.Repo.GetTimeoutedCandidates(ctx, DefaultTimeoutDays)
		if err != nil {
			return fmt.Errorf("failed to get timeout candidates: %w", err)
		}
		if len(candidates) == 0 {
			slog.Info("no submissions to timeout")
			return nil
		}

		slog.Info("found submissions to timeout", "count", len(candidates))
		count := 0
		for _, s := range candidates {
			if err := processTimeoutSubmission(ctx, deps, s); err != nil {
				slog.Error("failed to timeout submission", "submission_id", s.ID, "error", err)
				continue
			}
			count++
		}
		slog.Info("completed submission timeout task", "processed", count, "total", len(candidates))
		return nil
	}
}

func processTimeoutSubmission(ctx context.Context, deps TaskDeps, submission *sqlc.Submission) error {
	now := time.Now()

	if _, err := deps.Repo.UpdateStatus(ctx, submission.ID, sqlc.SubmissionStatusTimeouted); err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	_, _ = deps.HistoryRepo.Create(ctx, &sqlc.SubmissionStatusHistory{
		ID:           uuid.New(),
		SubmissionID: submission.ID,
		ChangedBy:    nil, // system action
		OldStatus:    sqlc.SubmissionStatusPending,
		NewStatus:    sqlc.SubmissionStatusTimeouted,
		Reason:       common.Ptr(fmt.Sprintf("Auto-timeouted after %d days of inactivity", DefaultTimeoutDays)),
		CreatedAt:    now,
	})

	body, _ := json.Marshal(map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          "submission_timeouted",
		"reason":        fmt.Sprintf("This submission was automatically timeouted after %d days of inactivity", DefaultTimeoutDays),
	})

	_, err := deps.NotifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    submission.RequesterID,
		Title:     common.Ptr("Submission Auto-timeouted"),
		Icon:      common.Ptr("clock"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelHigh,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      body,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return fmt.Errorf("failed to send timeout notification: %w", err)
	}

	slog.Info("timeouted submission", "submission_id", submission.ID, "requester_id", submission.RequesterID)
	return nil
}

// ── Auto-archive task ─────────────────────────────────────────────────────────

func autoArchiveTaskDefinition(deps TaskDeps) *worker.TaskDefinition {
	return &worker.TaskDefinition{
		Type:     TaskSubmissionAutoArchive,
		Handler:  handleAutoArchive(deps),
		Schedule: "0 3 * * *", // Daily at 03:00 AM
		Options: []asynq.Option{
			asynq.Queue(worker.QueueLow),
			asynq.MaxRetry(3),
			asynq.Timeout(10 * time.Minute),
		},
	}
}

func handleAutoArchive(deps TaskDeps) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		slog.Info("processing submission auto-archive task")

		candidates, err := deps.Repo.GetArchiveCandidates(ctx, DefaultArchiveDays)
		if err != nil {
			return fmt.Errorf("failed to get archive candidates: %w", err)
		}
		if len(candidates) == 0 {
			slog.Info("no submissions to archive")
			return nil
		}

		slog.Info("found submissions to archive", "count", len(candidates))

		ids := make([]uuid.UUID, len(candidates))
		for i, s := range candidates {
			ids[i] = s.ID
		}

		archived, err := deps.Repo.BulkMarkArchived(ctx, ids)
		if err != nil {
			return fmt.Errorf("failed to bulk archive submissions: %w", err)
		}

		now := time.Now()
		reason := common.Ptr(fmt.Sprintf("Auto-archived after %d days of inactivity", DefaultArchiveDays))

		// Build a lookup map for old status from candidates
		oldStatusMap := make(map[uuid.UUID]sqlc.SubmissionStatus, len(candidates))
		for _, s := range candidates {
			oldStatusMap[s.ID] = s.Status
		}

		for _, s := range archived {
			oldStatus := oldStatusMap[s.ID]

			// Record history
			_, _ = deps.HistoryRepo.Create(ctx, &sqlc.SubmissionStatusHistory{
				ID:           uuid.New(),
				SubmissionID: s.ID,
				ChangedBy:    nil, // system action
				OldStatus:    oldStatus,
				NewStatus:    sqlc.SubmissionStatusArchived,
				Reason:       reason,
				CreatedAt:    now,
			})

			// Notify requester
			body, _ := json.Marshal(map[string]any{
				"submission_id": s.ID.String(),
				"title":         s.Title,
				"type":          "submission_archived",
				"reason":        fmt.Sprintf("This submission was automatically archived after %d days without activity", DefaultArchiveDays),
			})
			_, _ = deps.NotifRepo.Create(ctx, &sqlc.Notification{
				ID:        uuid.New(),
				UserID:    s.RequesterID,
				Title:     common.Ptr("Submission Auto-archived"),
				Icon:      common.Ptr("archive"),
				ActionUrl: common.Ptr("/submissions/" + s.ID.String()),
				Level:     sqlc.NotificationLevelLow,
				IsRead:    common.Ptr(false),
				Type:      "submission",
				Body:      body,
				IsDeleted: common.Ptr(false),
				CreatedAt: &now,
				UpdatedAt: &now,
			})
		}

		slog.Info("completed submission auto-archive task", "archived", len(archived))
		return nil
	}
}

package submission

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/notification"
)

// Notifier handles sending notifications for submission events
type Notifier interface {
	// NotifyAssignment sends notification when user is assigned to submission
	NotifyAssignment(ctx context.Context, submission *sqlc.Submission, assigneeID, assignedBy uuid.UUID) error
	// NotifyApproval sends notification when submission is approved
	NotifyApproval(ctx context.Context, submission *sqlc.Submission, approverID uuid.UUID, reason string) error
	// NotifyRejection sends notification when submission is rejected
	NotifyRejection(ctx context.Context, submission *sqlc.Submission, approverID uuid.UUID, reason string) error
	// NotifyClaim sends notification when staff claims a submission
	NotifyClaim(ctx context.Context, submission *sqlc.Submission, claimerID uuid.UUID) error
	// NotifyTimeout sends notification when submission is auto-timeouted
	NotifyTimeout(ctx context.Context, submission *sqlc.Submission, reason string) error
}

type notifier struct {
	notifRepo notification.Repository
	notifier  notification.Notifier
}

func NewNotifier(notifRepo notification.Repository, wsNotifier notification.Notifier) Notifier {
	return &notifier{
		notifRepo: notifRepo,
		notifier:  wsNotifier,
	}
}

func (n *notifier) NotifyAssignment(ctx context.Context, submission *sqlc.Submission, assigneeID, assignedBy uuid.UUID) error {
	now := time.Now()

	body := map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          "submission_assigned",
		"assigned_by":   assignedBy.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	// Create persistent notification
	persistedNotif, err := n.notifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    assigneeID,
		Title:     common.Ptr("New Submission Assignment"),
		Icon:      common.Ptr("submission"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelMedium,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      bodyBytes,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: nil,
	})
	if err != nil {
		return err
	}

	// Send real-time WebSocket notification
	_ = n.notifier.NotifyNew(ctx, assigneeID, &notification.NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *persistedNotif,
			Recipients: []sqlc.NotificationRecipient{
				{
					ID:             uuid.New(),
					NotificationID: persistedNotif.ID,
					UserID:         assigneeID,
					IsRead:         common.Ptr(false),
					CreatedAt:      &now,
				},
			},
		},
	})

	return nil
}

func (n *notifier) NotifyApproval(ctx context.Context, submission *sqlc.Submission, approverID uuid.UUID, reason string) error {
	now := time.Now()

	body := map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          "submission_approved",
		"approver_id":   approverID.String(),
		"reason":        reason,
	}
	bodyBytes, _ := json.Marshal(body)

	// Create persistent notification
	persistedNotif, err := n.notifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    submission.RequesterID,
		Title:     common.Ptr("Submission Approved"),
		Icon:      common.Ptr("check"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelLow,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      bodyBytes,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: nil,
	})
	if err != nil {
		return err
	}

	// Send real-time WebSocket notification
	_ = n.notifier.NotifyNew(ctx, submission.RequesterID, &notification.NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *persistedNotif,
			Recipients: []sqlc.NotificationRecipient{
				{
					ID:             uuid.New(),
					NotificationID: persistedNotif.ID,
					UserID:         submission.RequesterID,
					IsRead:         common.Ptr(false),
					CreatedAt:      &now,
				},
			},
		},
	})

	return nil
}

func (n *notifier) NotifyRejection(ctx context.Context, submission *sqlc.Submission, approverID uuid.UUID, reason string) error {
	now := time.Now()

	body := map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          "submission_rejected",
		"approver_id":   approverID.String(),
		"reason":        reason,
	}
	bodyBytes, _ := json.Marshal(body)

	// Create persistent notification
	persistedNotif, err := n.notifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    submission.RequesterID,
		Title:     common.Ptr("Submission Rejected"),
		Icon:      common.Ptr("x"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelHigh,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      bodyBytes,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: nil,
	})
	if err != nil {
		return err
	}

	// Send real-time WebSocket notification
	_ = n.notifier.NotifyNew(ctx, submission.RequesterID, &notification.NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *persistedNotif,
			Recipients: []sqlc.NotificationRecipient{
				{
					ID:             uuid.New(),
					NotificationID: persistedNotif.ID,
					UserID:         submission.RequesterID,
					IsRead:         common.Ptr(false),
					CreatedAt:      &now,
				},
			},
		},
	})

	return nil
}

func (n *notifier) NotifyClaim(ctx context.Context, submission *sqlc.Submission, claimerID uuid.UUID) error {
	now := time.Now()

	body := map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          submission.Type,
		"claimer_id":    claimerID.String(),
	}
	bodyBytes, _ := json.Marshal(body)

	// Create persistent notification
	persistedNotif, err := n.notifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    submission.RequesterID,
		Title:     common.Ptr("Submission Claimed"),
		Icon:      common.Ptr("user"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelMedium,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      bodyBytes,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: nil,
	})
	if err != nil {
		return err
	}

	// Send real-time WebSocket notification
	_ = n.notifier.NotifyNew(ctx, submission.RequesterID, &notification.NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *persistedNotif,
			Recipients: []sqlc.NotificationRecipient{
				{
					ID:             uuid.New(),
					NotificationID: persistedNotif.ID,
					UserID:         submission.RequesterID,
					IsRead:         common.Ptr(false),
					CreatedAt:      &now,
				},
			},
		},
	})

	return nil
}

func (n *notifier) NotifyTimeout(ctx context.Context, submission *sqlc.Submission, reason string) error {
	now := time.Now()

	body := map[string]any{
		"submission_id": submission.ID.String(),
		"title":         submission.Title,
		"type":          "submission_timeouted",
		"reason":        reason,
	}
	bodyBytes, _ := json.Marshal(body)

	// Create persistent notification
	persistedNotif, err := n.notifRepo.Create(ctx, &sqlc.Notification{
		ID:        uuid.New(),
		UserID:    submission.RequesterID,
		Title:     common.Ptr("Submission Auto-timeouted"),
		Icon:      common.Ptr("clock"),
		ActionUrl: common.Ptr("/submissions/" + submission.ID.String()),
		Level:     sqlc.NotificationLevelHigh,
		IsRead:    common.Ptr(false),
		Type:      "submission",
		Body:      bodyBytes,
		IsDeleted: common.Ptr(false),
		CreatedAt: &now,
		UpdatedAt: &now,
		DeletedAt: nil,
	})
	if err != nil {
		return err
	}

	// Send real-time WebSocket notification
	_ = n.notifier.NotifyNew(ctx, submission.RequesterID, &notification.NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *persistedNotif,
			Recipients: []sqlc.NotificationRecipient{
				{
					ID:             uuid.New(),
					NotificationID: persistedNotif.ID,
					UserID:         submission.RequesterID,
					IsRead:         common.Ptr(false),
					CreatedAt:      &now,
				},
			},
		},
	})

	return nil
}

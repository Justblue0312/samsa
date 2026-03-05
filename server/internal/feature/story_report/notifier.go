package story_report

import (
	"context"

	"github.com/google/uuid"
)

// Notifier sends real-time notifications for report events
type Notifier interface {
	// NotifyNewReport notifies moderators when a new report is created
	NotifyNewReport(ctx context.Context, reportID, storyID, reporterID uuid.UUID) error
	// NotifyReportResolved notifies the reporter when their report is resolved
	NotifyReportResolved(ctx context.Context, reportID, reporterID uuid.UUID, notes *string) error
	// NotifyReportRejected notifies the reporter when their report is rejected
	NotifyReportRejected(ctx context.Context, reportID, reporterID uuid.UUID, reason *string) error
}

// noopNotifier is a no-op implementation for when notifications are not needed
type noopNotifier struct{}

func NewNoopNotifier() Notifier {
	return &noopNotifier{}
}

func (n *noopNotifier) NotifyNewReport(ctx context.Context, reportID, storyID, reporterID uuid.UUID) error {
	return nil
}

func (n *noopNotifier) NotifyReportResolved(ctx context.Context, reportID, reporterID uuid.UUID, notes *string) error {
	return nil
}

func (n *noopNotifier) NotifyReportRejected(ctx context.Context, reportID, reporterID uuid.UUID, reason *string) error {
	return nil
}

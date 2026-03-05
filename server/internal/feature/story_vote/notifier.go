package story_vote

import (
	"context"

	"github.com/google/uuid"
)

// Notifier sends real-time notifications for vote events
type Notifier interface {
	// NotifyVoteCreate notifies when a new vote is created
	NotifyVoteCreate(ctx context.Context, storyID, userID uuid.UUID, rating int32) error
	// NotifyVoteUpdate notifies when a vote is updated
	NotifyVoteUpdate(ctx context.Context, storyID, userID uuid.UUID, rating int32) error
	// NotifyVoteDelete notifies when a vote is deleted
	NotifyVoteDelete(ctx context.Context, storyID, userID uuid.UUID) error
}

// noopNotifier is a no-op implementation for when notifications are not needed
type noopNotifier struct{}

func NewNoopNotifier() Notifier {
	return &noopNotifier{}
}

func (n *noopNotifier) NotifyVoteCreate(ctx context.Context, storyID, userID uuid.UUID, rating int32) error {
	return nil
}

func (n *noopNotifier) NotifyVoteUpdate(ctx context.Context, storyID, userID uuid.UUID, rating int32) error {
	return nil
}

func (n *noopNotifier) NotifyVoteDelete(ctx context.Context, storyID, userID uuid.UUID) error {
	return nil
}

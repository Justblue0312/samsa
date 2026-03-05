package notification_recipient

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	GetByNotificationID(ctx context.Context, notificationID uuid.UUID) (*[]sqlc.NotificationRecipient, error)
	Create(ctx context.Context, notificationID, userID uuid.UUID) (*sqlc.NotificationRecipient, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(q *sqlc.Queries) Repository {
	return &repository{q: q}
}

func (r *repository) Create(ctx context.Context, notificationID, userID uuid.UUID) (*sqlc.NotificationRecipient, error) {
	nr, err := r.q.CreateNotificationRecipient(ctx, sqlc.CreateNotificationRecipientParams{
		NotificationID: notificationID,
		UserID:         userID,
		CreatedAt:      common.Ptr(time.Now()),
	})
	if err != nil {
		return nil, err
	}
	return &nr, nil
}

func (r *repository) GetByNotificationID(ctx context.Context, notificationID uuid.UUID) (*[]sqlc.NotificationRecipient, error) {
	nrs, err := r.q.ListNotificationRecipientsByNotificationID(ctx, notificationID)
	if err != nil {
		return nil, err
	}
	return &nrs, nil
}

package notification

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
)

type Repository interface {
	Create(ctx context.Context, notification *sqlc.Notification) (*sqlc.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*sqlc.Notification, error)
	GetByUserId(ctx context.Context, userID uuid.UUID, isRead *bool, includeDeleted bool, limit, offset int32) (*[]sqlc.Notification, error)
	List(ctx context.Context, f *NotificationFilter, includeDeleted bool) (*[]sqlc.Notification, *[]sqlc.NotificationRecipient, int64, error)
	Update(ctx context.Context, notification *sqlc.Notification) (*sqlc.Notification, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	MarkAsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error
	MarkAsUnread(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error
}

type repository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewRepository(q *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config) Repository {
	return &repository{
		q:    q,
		pool: pool,
		cfg:  cfg,
	}
}

func (r *repository) MarkAsUnread(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	return r.q.MarkNotificationsAsUnread(ctx, sqlc.MarkNotificationsAsUnreadParams{
		UserID:          userID,
		NotificationIds: notificationIDs,
	})
}

func (r *repository) MarkAsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	return r.q.MarkNotificationsAsRead(ctx, sqlc.MarkNotificationsAsReadParams{
		UserID:          userID,
		NotificationIds: notificationIDs,
	})
}
func (r *repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return r.q.MarkAllNotificationsAsRead(ctx, userID)
}

func (r *repository) Create(ctx context.Context, notification *sqlc.Notification) (*sqlc.Notification, error) {
	noti, err := r.q.CreateNotification(ctx, sqlc.CreateNotificationParams{
		UserID:    notification.UserID,
		Level:     notification.Level,
		IsRead:    notification.IsRead,
		Type:      notification.Type,
		Body:      notification.Body,
		CreatedAt: notification.CreatedAt,
		DeletedAt: notification.DeletedAt,
	})
	if err != nil {
		return nil, err
	}
	return &noti, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteNotification(ctx, id)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*sqlc.Notification, error) {
	noti, err := r.q.GetNotificationByID(ctx, sqlc.GetNotificationByIDParams{
		ID:        id,
		IsDeleted: common.Ptr(includeDeleted),
	})
	if err != nil {
		return nil, err
	}
	return &noti, nil
}

func (r *repository) GetByUserId(ctx context.Context, userID uuid.UUID, includeRead *bool, includeDeleted bool, limit, offset int32) (*[]sqlc.Notification, error) {
	notis, err := r.q.GetNotificationsByUserID(ctx, sqlc.GetNotificationsByUserIDParams{
		UserID:    userID,
		IsDeleted: common.Ptr(includeDeleted),
		IsRead:    includeRead,
		RowOffset: offset,
		RowLimit:  limit,
	})
	if err != nil {
		return nil, err
	}
	return &notis, nil
}

func (r *repository) Update(ctx context.Context, notification *sqlc.Notification) (*sqlc.Notification, error) {
	noti, err := r.q.UpdateNotification(ctx, sqlc.UpdateNotificationParams{
		ID:        notification.ID,
		UserID:    notification.UserID,
		Level:     notification.Level,
		IsRead:    notification.IsRead,
		Type:      notification.Type,
		Body:      notification.Body,
		DeletedAt: notification.DeletedAt,
	})
	if err != nil {
		return nil, err
	}
	return &noti, nil
}

func (r *repository) List(ctx context.Context, f *NotificationFilter, includeDeleted bool) (*[]sqlc.Notification, *[]sqlc.NotificationRecipient, int64, error) {
	var nl sqlc.NullNotificationLevel
	if f.Level != nil {
		nl = sqlc.NullNotificationLevel{NotificationLevel: *f.Level, Valid: true}
	}

	rows, err := r.q.ListNotifications(ctx, sqlc.ListNotificationsParams{
		UserID:    f.UserID,
		IsDeleted: common.Ptr(includeDeleted),
		IsRead:    f.IsRead,
		Type:      f.Type,
		Level:     nl,
		OrderBy:   f.GetOrderByEntry(),
		RowOffset: f.GetOffset(),
		RowLimit:  f.GetLimit(),
	})
	if err != nil {
		return nil, nil, 0, err
	}

	if len(rows) == 0 {
		return &[]sqlc.Notification{}, &[]sqlc.NotificationRecipient{}, 0, nil
	}

	notifications := make([]sqlc.Notification, 0, len(rows))
	recipients := make([]sqlc.NotificationRecipient, 0, len(rows))
	var totalCount int64

	for _, row := range rows {
		notifications = append(notifications, row.Notification)
		recipients = append(recipients, row.NotificationRecipient)
		totalCount = row.TotalCount
	}

	return &notifications, &recipients, totalCount, nil
}

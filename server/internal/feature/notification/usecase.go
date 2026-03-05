package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/multierr"

	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	notificationrecipient "github.com/justblue/samsa/internal/feature/notification_recipient"
	"github.com/justblue/samsa/internal/transport/ws"
	"github.com/justblue/samsa/pkg/queryparam"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	Create(ctx context.Context, user *sqlc.User, req *CreateNotificationRequest, recipentIds *[]uuid.UUID) (*NotificationResponse, error)
	GetByID(ctx context.Context, user *sqlc.User, notiID uuid.UUID) (*NotificationResponse, error)
	GetUnread(ctx context.Context, user *sqlc.User, limit, page int32) (*NotificationsResponse, error)
	List(ctx context.Context, user *sqlc.User, f *NotificationFilter) (*NotificationsResponse, error)
	MarkAsRead(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error
	MarkAsUnread(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, user *sqlc.User) error
	Delete(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error
	MarkAsReadWithNotify(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error
	DeleteWithNotify(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error
}

type usecase struct {
	cfg         *config.Config
	wsPubliser  *ws.Publisher
	notifier    Notifier
	notiRepo    Repository
	notiRepRepo notificationrecipient.Repository
}

func NewUseCase(
	cfg *config.Config,
	wsPubliser *ws.Publisher,
	notifier Notifier,
	notiRepo Repository,
	notiRepRepo notificationrecipient.Repository,
) UseCase {
	return &usecase{
		cfg:         cfg,
		wsPubliser:  wsPubliser,
		notifier:    notifier,
		notiRepo:    notiRepo,
		notiRepRepo: notiRepRepo,
	}
}

func (u *usecase) NotifyUser(ctx context.Context, userID uuid.UUID, msg Message) error {
	pl, _ := json.Marshal(msg)
	b, err := json.Marshal(ws.Envelope{
		Type:    ws.TypeNotificationNew,
		Payload: pl,
	})
	if err != nil {
		return err
	}
	return u.wsPubliser.Publish(ctx, uuid.Nil, userID, b)
}

func (u *usecase) Create(ctx context.Context, user *sqlc.User, req *CreateNotificationRequest, recipentIds *[]uuid.UUID) (*NotificationResponse, error) {
	bodyBytes, err := json.Marshal(req.Body)
	if err != nil {
		return nil, err
	}

	noti, err := u.notiRepo.Create(ctx, &sqlc.Notification{
		UserID: user.ID,
		Level:  sqlc.NotificationLevelDefault,
		IsRead: common.Ptr(false),
		Type:   req.Type,
		Body:   bodyBytes,
	})
	if err != nil {
		return nil, err
	}

	var repErr error
	var recipients []sqlc.NotificationRecipient
	if recipentIds != nil {
		for _, id := range *recipentIds {
			recipient, err := u.notiRepRepo.Create(ctx, noti.ID, id)
			if err != nil {
				repErr = multierr.Append(repErr, err)
			}
			recipients = append(recipients, *recipient)

			// If user is online, publish notification via notifier
			if u.notifier != nil {
				result := &NotificationResponse{
					Notification: struct {
						sqlc.Notification
						Recipients []sqlc.NotificationRecipient `json:"recipients"`
					}{
						Notification: *noti,
						Recipients:   recipients,
					},
				}
				err = u.notifier.NotifyNew(ctx, id, result)
				if err != nil {
					repErr = multierr.Append(repErr, err)
				}
			}
		}
	}
	if repErr != nil {
		return nil, repErr
	}

	return &NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *noti,
			Recipients:   recipients,
		},
	}, nil
}

func (u *usecase) GetByID(ctx context.Context, user *sqlc.User, notiID uuid.UUID) (*NotificationResponse, error) {
	noti, err := u.notiRepo.GetByID(ctx, notiID, true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}

	if noti.UserID != user.ID {
		return nil, ErrNotificationAccessDenied
	}

	notiReps, err := u.notiRepRepo.GetByNotificationID(ctx, notiID)
	if err != nil {
		return nil, err
	}

	return &NotificationResponse{
		Notification: struct {
			sqlc.Notification
			Recipients []sqlc.NotificationRecipient `json:"recipients"`
		}{
			Notification: *noti,
			Recipients:   *notiReps,
		},
	}, nil
}

func (u *usecase) GetUnread(ctx context.Context, user *sqlc.User, limit, page int32) (*NotificationsResponse, error) {
	return u.listNotifications(ctx, &NotificationFilter{
		PaginationParams: &queryparam.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		UserID: user.ID,
		IsRead: common.Ptr(false),
	})
}

func (u *usecase) List(ctx context.Context, user *sqlc.User, f *NotificationFilter) (*NotificationsResponse, error) {
	f.UserID = user.ID
	return u.listNotifications(ctx, f)
}

func (u *usecase) listNotifications(ctx context.Context, f *NotificationFilter) (*NotificationsResponse, error) {
	notis, notiReps, count, err := u.notiRepo.List(ctx, f, true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &NotificationsResponse{
				Notifications: []NotificationResponse{},
				Meta:          queryparam.NewPaginationMeta(f.Limit, f.Page, count),
			}, nil
		}
		return nil, err
	}

	recipientMap := make(map[uuid.UUID][]sqlc.NotificationRecipient)
	for _, rep := range *notiReps {
		recipientMap[rep.NotificationID] = append(recipientMap[rep.NotificationID], rep)
	}

	res := make([]NotificationResponse, 0, len(*notis))
	for _, noti := range *notis {
		res = append(res, NotificationResponse{
			Notification: struct {
				sqlc.Notification
				Recipients []sqlc.NotificationRecipient `json:"recipients"`
			}{
				Notification: noti,
				Recipients:   recipientMap[noti.ID],
			},
		})
	}

	return &NotificationsResponse{
		Notifications: res,
		Meta:          queryparam.NewPaginationMeta(f.Limit, f.Page, count),
	}, nil
}

func (u *usecase) MarkAllAsRead(ctx context.Context, user *sqlc.User) error {
	return u.notiRepo.MarkAllAsRead(ctx, user.ID)
}

func (u *usecase) MarkAsRead(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	noti, err := u.notiRepo.GetByID(ctx, notiID, false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotificationNotFound
		}
		return err
	}

	if noti.UserID != user.ID {
		return ErrNotificationAccessDenied
	}

	return u.notiRepo.MarkAsRead(ctx, user.ID, []uuid.UUID{notiID})
}

func (u *usecase) MarkAsUnread(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	noti, err := u.notiRepo.GetByID(ctx, notiID, false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotificationNotFound
		}
		return err
	}

	if noti.UserID != user.ID {
		return ErrNotificationAccessDenied
	}

	return u.notiRepo.MarkAsUnread(ctx, user.ID, []uuid.UUID{notiID})
}

func (u *usecase) Delete(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	noti, err := u.notiRepo.GetByID(ctx, notiID, false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotificationNotFound
		}
		return err
	}

	if noti.UserID != user.ID {
		return ErrNotificationAccessDenied
	}

	return u.notiRepo.Delete(ctx, notiID)
}

// MarkAsReadWithNotify marks a notification as read and notifies WS clients.
func (u *usecase) MarkAsReadWithNotify(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	if err := u.MarkAsRead(ctx, user, notiID); err != nil {
		return err
	}
	if u.notifier != nil {
		_ = u.notifier.NotifyRead(ctx, user.ID, notiID)
	}
	return nil
}

// DeleteWithNotify deletes a notification and notifies WS clients.
func (u *usecase) DeleteWithNotify(ctx context.Context, user *sqlc.User, notiID uuid.UUID) error {
	if err := u.Delete(ctx, user, notiID); err != nil {
		return err
	}
	if u.notifier != nil {
		_ = u.notifier.NotifyDeleted(ctx, user.ID, notiID)
	}
	return nil
}

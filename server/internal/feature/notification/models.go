package notification

import (
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type CreateNotificationRequest struct {
	UserID    *uuid.UUID     `json:"user_id"`
	Type      string         `json:"type" validate:"required"`
	Level     string         `json:"level" validate:"required oneof=low medium high default"`
	Icon      string         `json:"icon"`
	ActionURL string         `json:"action_url"`
	Body      map[string]any `json:"body" validate:"required"`
}

type NotificationResponse struct {
	Notification struct {
		sqlc.Notification
		Recipients []sqlc.NotificationRecipient `json:"recipients"`
	} `json:"notification"`
}

type NotificationsResponse struct {
	Notifications []NotificationResponse    `json:"notifications"`
	Meta          queryparam.PaginationMeta `json:"meta"`
}

type Message struct {
	UserID         uuid.UUID `json:"user_id"`
	NotificationID uuid.UUID `json:"notification_id"`
	RecipientID    uuid.UUID `json:"recipient_id"`
}

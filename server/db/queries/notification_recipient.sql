-- name: CreateNotificationRecipient :one
INSERT INTO notification_recipient (id, notification_id, user_id, is_read, read_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetNotificationRecipient :one
SELECT * FROM notification_recipient
WHERE notification_id = $1 AND user_id = $2;

-- name: ListNotificationRecipientsByNotificationID :many
SELECT *
FROM notification_recipient
WHERE notification_id = @notification_id;

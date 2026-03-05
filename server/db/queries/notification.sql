-- name: GetNotificationByID :one
SELECT * FROM notification WHERE id = @id AND is_deleted = @is_deleted;

-- name: GetNotificationByIDWithRecipients :many
SELECT sqlc.embed(notification), sqlc.embed(notification_recipient)
FROM notification
JOIN notification_recipient ON notification.id = notification_recipient.notification_id
WHERE notification.id = @id AND notification.is_deleted = @is_deleted;


-- name: GetNotificationsByUserID :many
SELECT *
FROM notification
WHERE
    user_id = @user_id
    AND is_deleted = @is_deleted
    AND (is_read = @is_read OR @is_read IS NULL)
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CreateNotification :one
INSERT INTO notification (user_id, level, is_read, type, body, created_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateNotification :one
UPDATE notification
SET
    user_id = $2,
    level = $3,
    is_read = $4,
    type = $5,
    body = $6,
    deleted_at = $7
WHERE id = $1
RETURNING *;

-- name: DeleteNotification :exec
DELETE FROM notification WHERE id = $1;

-- name: ListNotifications :many
-- ListNotifications list notifications
-- orderBy options: created_at_asc, created_at_desc, updated_at_asc, updated_at_desc
SELECT
    sqlc.embed(notification),
    sqlc.embed(notification_recipient),
    COUNT(*) OVER() AS total_count
FROM notification
JOIN notification_recipient ON notification.id = notification_recipient.notification_id
WHERE
    notification.user_id = @user_id
    AND notification.is_deleted = @is_deleted
    AND (notification_recipient.is_read = @is_read OR @is_read IS NULL)
    AND (notification.type = sqlc.narg('type') OR sqlc.narg('type') IS NULL)
    AND (notification.level = sqlc.narg('level') OR sqlc.narg('level') IS NULL)
ORDER BY
    CASE WHEN @order_by::text = 'created_at_asc' THEN notification.created_at END ASC,
    CASE WHEN @order_by::text = 'created_at_desc' THEN notification.created_at END DESC,
    CASE WHEN @order_by::text = 'updated_at_asc' THEN notification.updated_at END ASC,
    CASE WHEN @order_by::text = 'updated_at_desc' THEN notification.updated_at END DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notification
SET
    is_read = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationsAsRead :exec
UPDATE notification
SET
    is_read = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE
    user_id = @user_id
    AND id = ANY(@notification_ids::uuid[])
    AND is_read = FALSE;

-- name: MarkNotificationsAsUnread :exec
UPDATE notification
SET
    is_read = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE
    user_id = @user_id
    AND id = ANY(@notification_ids::uuid[])
    AND is_read = TRUE;

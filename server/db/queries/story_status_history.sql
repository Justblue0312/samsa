-- name: CreateStoryStatusHistory :one
INSERT INTO story_status_history (
    story_id, set_status_by, content, status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: ListStoryStatusHistoryByStory :many
SELECT * FROM story_status_history
WHERE story_id = $1
ORDER BY created_at DESC;

-- name: GetStoryStatusHistoryByID :one
SELECT * FROM story_status_history WHERE id = $1;

-- name: ListStoryStatusHistoryByStoryPaginated :many
SELECT * FROM story_status_history
WHERE story_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoryStatusHistoryByStory :one
SELECT COUNT(*) FROM story_status_history WHERE story_id = $1;

-- name: DeleteStoryStatusHistory :exec
DELETE FROM story_status_history WHERE id = $1;

-- name: DeleteStoryStatusHistoryByStory :exec
DELETE FROM story_status_history WHERE story_id = $1;

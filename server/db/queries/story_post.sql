-- name: CreateStoryPost :one
INSERT INTO story_post (
    author_id, content, media_ids, story_id, chapter_id, is_notify_followers
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetStoryPostByID :one
SELECT * FROM story_post WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStoryPostByIDWithDeleted :one
SELECT * FROM story_post WHERE id = $1;

-- name: ListStoryPostsByAuthor :many
SELECT * FROM story_post
WHERE author_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListStoryPostsByStory :many
SELECT * FROM story_post
WHERE story_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListStoryPostsByStoryWithFilters :many
SELECT * FROM story_post
WHERE (sqlc.narg('story_id')::uuid IS NULL OR story_id = sqlc.narg('story_id')::uuid)
  AND (sqlc.narg('author_id')::uuid IS NULL OR author_id = sqlc.narg('author_id')::uuid)
  AND (sqlc.narg('include_deleted')::boolean IS TRUE OR deleted_at IS NULL)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStoryPostsByStory :one
SELECT COUNT(*) FROM story_post WHERE story_id = $1 AND deleted_at IS NULL;

-- name: CountStoryPostsByAuthor :one
SELECT COUNT(*) FROM story_post WHERE author_id = $1 AND deleted_at IS NULL;

-- name: GetStoryPostsByIDs :many
SELECT * FROM story_post WHERE id = ANY(sqlc.slice('ids')) AND deleted_at IS NULL;

-- name: RestoreStoryPost :one
UPDATE story_post
SET deleted_at = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: PermanentlyDeleteStoryPost :exec
DELETE FROM story_post WHERE id = $1;

-- name: BulkDeleteStoryPosts :exec
UPDATE story_post SET deleted_at = CURRENT_TIMESTAMP WHERE id = ANY(sqlc.slice('ids'));

-- name: UpdateStoryPost :one
UPDATE story_post
SET
    content = $2,
    media_ids = $3,
    is_notify_followers = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteStoryPost :exec
UPDATE story_post SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: GetStoryByID :one
SELECT * FROM story WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStoryBySlug :one
SELECT * FROM story WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetStoriesByOwnerID :many
SELECT * FROM story WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CreateStory :one
INSERT INTO story (owner_id, media_id, name, slug, synopsis, is_verified, is_recommended, status, first_published_at, last_published_at, settings, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: UpdateStory :one
UPDATE story
SET
    media_id = $2,
    name = $3,
    slug = $4,
    synopsis = $5,
    is_verified = $6,
    is_recommended = $7,
    status = $8,
    first_published_at = $9,
    last_published_at = $10,
    settings = $11,
    deleted_at = $12
WHERE id = $1
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM story WHERE id = $1;

-- name: SoftDeleteStory :one
UPDATE story SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

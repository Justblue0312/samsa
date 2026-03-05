-- name: GetChapterByID :one
SELECT * FROM chapter WHERE id = $1 AND deleted_at IS NULL;

-- name: GetChapterByStoryAndNumber :one
SELECT * FROM chapter WHERE story_id = $1 AND number = $2 AND deleted_at IS NULL;

-- name: GetChaptersByStoryID :many
SELECT * FROM chapter WHERE story_id = $1 AND deleted_at IS NULL ORDER BY sort_order, number;

-- name: GetPublishedChaptersByStoryID :many
SELECT * FROM chapter WHERE story_id = $1 AND is_published = true AND deleted_at IS NULL ORDER BY sort_order, number;

-- name: CreateChapter :one
INSERT INTO chapter (story_id, title, number, sort_order, summary, is_published, published_at, total_words, total_views, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateChapter :one
UPDATE chapter
SET
    title = $2,
    number = $3,
    sort_order = $4,
    summary = $5,
    is_published = $6,
    published_at = $7,
    total_words = $8,
    total_views = $9,
    updated_at = $10
WHERE id = $1
RETURNING *;

-- name: DeleteChapter :exec
DELETE FROM chapter WHERE id = $1;

-- name: SoftDeleteChapter :one
UPDATE chapter SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: PublishChapter :one
UPDATE chapter
SET is_published = true, published_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UnpublishChapter :one
UPDATE chapter
SET is_published = false, published_at = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: IncrementChapterViews :one
UPDATE chapter
SET total_views = COALESCE(total_views, 0) + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateChapterStats :one
UPDATE chapter
SET
    total_words = $2,
    total_votes = $3,
    total_favorites = $4,
    total_bookmarks = $5,
    total_flags = $6,
    total_reports = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetNextChapterSortOrder :one
SELECT COALESCE(MAX(sort_order), -1) + 1 FROM chapter WHERE story_id = $1 AND deleted_at IS NULL;

-- name: ReorderChapters :exec
UPDATE chapter
SET sort_order = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND story_id = $2;

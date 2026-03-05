-- name: GetCommentByID :one
SELECT * FROM comment WHERE id = $1 and entity_type = $2 and is_deleted = $3;

-- name: GetCommentByIDWithDeleted :one
SELECT * FROM comment WHERE id = $1 AND entity_type = $2;

-- name: CreateComment :one
INSERT INTO comment (user_id, parent_id, content, depth, score, is_deleted, is_resolved, is_archived, is_reported, reported_at, reported_by, is_pinned, pinned_at, pinned_by, entity_type, entity_id, source, reply_count, reaction_count, metadata, deleted_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
RETURNING *;

-- name: UpdateComment :one
UPDATE comment
SET
    user_id = $3,
    parent_id = $4,
    content = $5,
    depth = $6,
    score = $7,
    is_resolved = $8,
    is_archived = $9,
    is_reported = $10,
    reported_at = $11,
    reported_by = $12,
    is_pinned = $13,
    pinned_at = $14,
    pinned_by = $15,
    entity_id = $16,
    source = $17,
    reply_count = $18,
    reaction_count = $19,
    metadata = $20,
    deleted_by = $21
WHERE entity_type = $1 AND id = $2
RETURNING *;

-- name: SoftDeleteComment :one
UPDATE comment SET is_deleted = TRUE WHERE id = $1 and entity_type = $2 RETURNING *;

-- name: BulkDeleteComments :many
UPDATE comment SET is_deleted = TRUE, deleted_by = $2 WHERE id = ANY($1::uuid[]) RETURNING *;

-- name: BulkArchiveComments :many
UPDATE comment SET is_archived = TRUE WHERE id = ANY($1::uuid[]) RETURNING *;

-- name: BulkResolveComments :many
UPDATE comment SET is_resolved = TRUE WHERE id = ANY($1::uuid[]) RETURNING *;

-- name: BulkPinComments :many
UPDATE comment SET is_pinned = TRUE, pinned_at = CURRENT_TIMESTAMP, pinned_by = $2 WHERE id = ANY($1::uuid[]) RETURNING *;

-- name: BulkUnpinComments :many
UPDATE comment SET is_pinned = FALSE WHERE id = ANY($1::uuid[]) RETURNING *;

-- name: ListCommentsByEntity :many
SELECT * FROM comment
WHERE entity_type = $1 AND entity_id = $2 AND is_deleted = FALSE AND parent_id IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListCommentsByEntityWithFilters :many
SELECT * FROM comment
WHERE entity_type = @entity_type
  AND entity_id = @entity_id
  AND (sqlc.narg('is_deleted')::boolean IS NULL OR is_deleted = sqlc.narg('is_deleted'))
  AND (sqlc.narg('is_resolved')::boolean IS NULL OR is_resolved = sqlc.narg('is_resolved'))
  AND (sqlc.narg('is_archived')::boolean IS NULL OR is_archived = sqlc.narg('is_archived'))
  AND (sqlc.narg('is_reported')::boolean IS NULL OR is_reported = sqlc.narg('is_reported'))
  AND (sqlc.narg('is_pinned')::boolean IS NULL OR is_pinned = sqlc.narg('is_pinned'))
  AND (sqlc.narg('parent_id')::uuid IS NULL OR parent_id = sqlc.narg('parent_id')::uuid)
ORDER BY 
    CASE WHEN is_pinned THEN 0 ELSE 1 END,
    created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountCommentsByEntity :one
SELECT COUNT(*) FROM comment WHERE entity_type = $1 AND entity_id = $2 AND is_deleted = FALSE;

-- name: CountCommentsWithFilters :one
SELECT COUNT(*) FROM comment
WHERE entity_type = @entity_type
  AND entity_id = @entity_id
  AND (sqlc.narg('is_deleted')::boolean IS NULL OR is_deleted = sqlc.narg('is_deleted'))
  AND (sqlc.narg('is_resolved')::boolean IS NULL OR is_resolved = sqlc.narg('is_resolved'))
  AND (sqlc.narg('is_archived')::boolean IS NULL OR is_archived = sqlc.narg('is_archived'))
  AND (sqlc.narg('is_reported')::boolean IS NULL OR is_reported = sqlc.narg('is_reported'));

-- name: SearchComments :many
SELECT * FROM comment
WHERE entity_type = @entity_type
  AND entity_id = @entity_id
  AND is_deleted = FALSE
  AND content ILIKE '%' || @search || '%'
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetCommentsByIDs :many
SELECT * FROM comment WHERE id = ANY(sqlc.slice('ids'));

-- name: GetCommentReplies :many
SELECT *
FROM comment
WHERE
    parent_id = @parent_id and entity_type = @entity_type and is_deleted = @is_deleted
ORDER BY created_at ASC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetCommentNestingDepth :one
WITH RECURSIVE comment_tree AS (
    SELECT c.id, c.parent_id, 1 as depth
    FROM comment c
    WHERE c.id = $1 AND c.entity_type = $2 and c.is_deleted = $3
    UNION ALL
    SELECT c2.id, c2.parent_id, ct.depth + 1
    FROM comment c2
    JOIN comment_tree ct ON c2.id = ct.parent_id
)
SELECT COALESCE(MAX(depth), 0)::int FROM comment_tree;

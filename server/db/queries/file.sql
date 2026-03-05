-- name: GetFileByID :one
SELECT * FROM file WHERE id = $1 AND is_deleted = @is_deleted;

-- name: GetFileByIDWithDeleted :one
SELECT * FROM file WHERE id = $1;

-- name: GetFilesByOwnerID :many
SELECT * FROM file
WHERE
    owner_id = @owner_id
    AND is_deleted = @is_deleted
ORDER BY created_at DESC
LIMIT @row_limit
OFFSET @row_offset;

-- name: CreateFile :one
INSERT INTO file (owner_id, name, path, mime_type, size, reference, payload, service, source, is_archived, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateFile :one
UPDATE file
SET
    name = $2,
    path = $3,
    mime_type = $4,
    size = $5,
    reference = $6,
    payload = $7,
    service = $8,
    source = $9,
    is_archived = $10,
    updated_at = $11,
    is_deleted = $12
WHERE id = $1
RETURNING *;

-- name: DeleteFile :exec
DELETE FROM file WHERE id = $1;

-- name: SoftDeleteFile :one
UPDATE file SET is_deleted = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: RestoreFile :one
UPDATE file SET is_deleted = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: GetFilesByIDs :many
SELECT * FROM file WHERE id IN (sqlc.slice(file_ids));

-- name: CountFilesByOwner :one
SELECT COUNT(*) FROM file WHERE owner_id = $1 AND is_deleted = FALSE;

-- name: GetFileByPath :one
SELECT * FROM file WHERE path = $1 AND is_deleted = FALSE;

-- name: GetFilesByOwnerAndType :many
SELECT * FROM file
WHERE owner_id = $1 
  AND mime_type LIKE $2
  AND is_deleted = FALSE
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSharedFiles :many
SELECT * FROM file
WHERE reference = 'shared'
  AND is_deleted = FALSE
  AND is_archived = FALSE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ShareFile :one
UPDATE file
SET reference = 'shared', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UnshareFile :one
UPDATE file
SET reference = 'private', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetFilesByMimeType :many
SELECT * FROM file
WHERE mime_type = $1 AND is_deleted = FALSE
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFilesByMimeType :one
SELECT COUNT(*) FROM file WHERE mime_type = $1 AND is_deleted = FALSE;

-- name: GetTotalSizeByOwner :one
SELECT COALESCE(SUM(size), 0)::bigint FROM file WHERE owner_id = $1 AND is_deleted = FALSE;

-- name: ListFilesWithFilters :many
SELECT * FROM file
WHERE (sqlc.narg('owner_id')::uuid IS NULL OR owner_id = sqlc.narg('owner_id')::uuid)
  AND (sqlc.narg('mime_type')::text IS NULL OR mime_type = sqlc.narg('mime_type')::text)
  AND (sqlc.narg('reference')::text IS NULL OR reference = sqlc.narg('reference')::text)
  AND (sqlc.narg('is_archived')::boolean IS NULL OR is_archived = sqlc.narg('is_archived'))
  AND is_deleted = FALSE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountFilesWithFilters :one
SELECT COUNT(*) FROM file
WHERE (sqlc.narg('owner_id')::uuid IS NULL OR owner_id = sqlc.narg('owner_id')::uuid)
  AND (sqlc.narg('mime_type')::text IS NULL OR mime_type = sqlc.narg('mime_type')::text)
  AND (sqlc.narg('reference')::text IS NULL OR reference = sqlc.narg('reference')::text)
  AND (sqlc.narg('is_archived')::boolean IS NULL OR is_archived = sqlc.narg('is_archived'))
  AND is_deleted = FALSE;

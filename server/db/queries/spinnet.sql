-- name: GetSpinnetByID :one
SELECT * FROM spinnet WHERE id = $1 AND is_deleted = FALSE;

-- name: GetSpinnetBySmartSyntax :one
SELECT * FROM spinnet WHERE smart_syntax = $1 AND is_deleted = FALSE;

-- name: ListSpinnets :many
SELECT * FROM spinnet
WHERE is_deleted = FALSE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateSpinnet :one
INSERT INTO spinnet (id, owner_id, name, content, category, smart_syntax, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateSpinnet :one
UPDATE spinnet
SET
    name = $2,
    content = $3,
    category = $4,
    smart_syntax = $5,
    updated_at = $6
WHERE id = $1 AND is_deleted = FALSE
RETURNING *;

-- name: DeleteSpinnet :exec
UPDATE spinnet SET is_deleted = TRUE, deleted_at = $2 WHERE id = $1;

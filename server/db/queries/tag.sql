-- name: GetTagByID :one
SELECT * FROM tag WHERE id = $1 AND entity_type = $2;

-- name: GetTagByNameAndType :one
SELECT * FROM tag
WHERE name = $1 AND entity_type = $2 AND color = $3;

-- name: GetTagsByEntityID :many
SELECT * FROM tag
WHERE
    entity_id = @entity_id
    AND entity_type = @entity_type
    AND (is_hidden = @is_hidden OR @is_hidden is NULL)
    AND (is_system = @is_system OR @is_system is NULL)
    AND (is_recommended = @is_recommended OR @is_recommended is NULL)
ORDER BY name ASC;

-- name: GetTagsByOwnerID :many
SELECT * FROM tag
WHERE
    owner_id = @owner_id
    AND entity_type = @entity_type
    AND (is_hidden = @is_hidden OR @is_hidden is NULL)
    AND (is_system = @is_system OR @is_system is NULL)
    AND (is_recommended = @is_recommended OR @is_recommended is NULL)
ORDER BY name ASC
LIMIT @row_limit OFFSET @row_offset;

-- name: CreateTag :one
INSERT INTO tag (id, owner_id, name, description, color, entity_type, entity_id, is_hidden, is_system, is_recommended, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateTag :one
UPDATE tag
SET
    name = $2,
    description = $3,
    color = $4,
    is_hidden = $5,
    is_system = $6,
    is_recommended = $7,
    updated_at = $8
WHERE id = $1 AND entity_type = $9
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tag WHERE id = $1 AND entity_type = $2;

-- name: GetTagsByIDs :many
SELECT * FROM tag WHERE id = ANY(sqlc.slice(tag_ids)::uuid[]) AND entity_type = sqlc.arg(entity_type);

-- name: CountTagsByEntity :one
SELECT COUNT(*) FROM tag
WHERE entity_id = $1 AND entity_type = $2;

-- name: CountTagsByOwner :one
SELECT COUNT(*) FROM tag
WHERE owner_id = @owner_id AND entity_type = @entity_type;

-- name: SearchTags :many
SELECT * FROM tag
WHERE
    entity_type = @entity_type
    AND (
        @search_query IS NULL OR
        name ILIKE '%' || @search_query || '%' OR
        description ILIKE '%' || @search_query || '%'
    )
    AND (is_hidden = @is_hidden OR @is_hidden is NULL)
    AND (is_system = @is_system OR @is_system is NULL)
    AND (is_recommended = @is_recommended OR @is_recommended is NULL)
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: UpsertTag :one
INSERT INTO tag (id, owner_id, name, description, color, entity_type, entity_id, is_hidden, is_system, is_recommended, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (entity_type, name, color)
DO UPDATE SET
    description = EXCLUDED.description,
    is_hidden = EXCLUDED.is_hidden,
    is_system = EXCLUDED.is_system,
    is_recommended = EXCLUDED.is_recommended,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetTagsByNames :many
SELECT DISTINCT entity_id
FROM tag
WHERE name IN (sqlc.slice(names))
AND entity_type = @entity_type
AND entity_id IS NOT NULL;

-- name: CreateFlag :one
INSERT INTO flag (
    story_id,
    chapter_id,
    inspector_id,
    title,
    description,
    flag_type,
    flag_rate,
    flag_score
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetFlagByID :one
SELECT * FROM flag
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFlags :many
SELECT * FROM flag
WHERE deleted_at IS NULL
    AND (sqlc.narg('story_id')::uuid IS NULL OR story_id = sqlc.narg('story_id')::uuid)
    AND (sqlc.narg('chapter_id')::uuid IS NULL OR chapter_id = sqlc.narg('chapter_id')::uuid)
    AND (sqlc.narg('inspector_id')::uuid IS NULL OR inspector_id = sqlc.narg('inspector_id')::uuid)
    AND (sqlc.narg('flag_type')::flag_types IS NULL OR flag_type = sqlc.narg('flag_type')::flag_types)
    AND (sqlc.narg('flag_rate')::flag_rate IS NULL OR flag_rate = sqlc.narg('flag_rate')::flag_rate)
    AND (sqlc.narg('min_score')::float8 IS NULL OR flag_score >= sqlc.narg('min_score')::float8)
    AND (sqlc.narg('max_score')::float8 IS NULL OR flag_score <= sqlc.narg('max_score')::float8)
ORDER BY flag_score DESC, created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListFlagsByStory :many
SELECT * FROM flag
WHERE story_id = $1 AND deleted_at IS NULL
ORDER BY flag_score DESC, created_at DESC;

-- name: ListFlagsByChapter :many
SELECT * FROM flag
WHERE chapter_id = $1 AND deleted_at IS NULL
ORDER BY flag_score DESC, created_at DESC;

-- name: ListFlagsByInspector :many
SELECT * FROM flag
WHERE inspector_id = $1 AND deleted_at IS NULL
ORDER BY flag_score DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateFlag :one
UPDATE flag
SET
    title = COALESCE($2, title),
    description = COALESCE($3, description),
    flag_type = COALESCE($4, flag_type),
    flag_rate = COALESCE($5, flag_rate),
    flag_score = COALESCE($6, flag_score)
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteFlag :exec
UPDATE flag
SET deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: HardDeleteFlag :exec
DELETE FROM flag WHERE id = $1;

-- name: GetFlagCount :one
SELECT COUNT(*) FROM flag
WHERE deleted_at IS NULL
    AND (@story_id::uuid IS NULL OR story_id = @story_id)
    AND (@chapter_id::uuid IS NULL OR chapter_id = @chapter_id)
    AND (@inspector_id::uuid IS NULL OR inspector_id = @inspector_id)
    AND (@flag_type::flag_types IS NULL OR flag_type = @flag_type)
    AND (@flag_rate::flag_rate IS NULL OR flag_rate = @flag_rate);

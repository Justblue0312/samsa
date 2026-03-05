-- name: GetCommentReactionByID :one
SELECT * FROM comment_reaction WHERE id = $1;

-- name: GetReactionsByCommentID :many
SELECT * FROM comment_reaction
WHERE comment_id = @comment_id AND entity_type = @entity_type
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountReactionsByCommentID :many
SELECT
    comment_id,
    entity_type,
    reaction_type,
    total
FROM comment_reaction_count_mv
WHERE comment_id = $1 AND entity_type = $2;

-- name: UpsertCommentReaction :one
INSERT INTO comment_reaction (entity_type, comment_id, user_id, reaction_type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (entity_type, comment_id, user_id, reaction_type)
DO UPDATE SET
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: DeleteCommentReaction :exec
DELETE FROM comment_reaction
WHERE id = $1;

-- name: CountTotalCommentReactions :one
SELECT COUNT(*) FROM comment_reaction
WHERE comment_id = @comment_id AND entity_type = @entity_type;

-- name: GetCommentVoteByID :one
SELECT * FROM comment_vote WHERE id = $1;

-- name: GetCommentVotesByCommentID :many
SELECT * FROM comment_vote
WHERE comment_id = @comment_id AND entity_type = @entity_type
LIMIT @row_limit OFFSET @row_offset;

-- name: CountTotalCommentVotesByCommentID :one
SELECT COUNT(*) FROM comment_vote
WHERE comment_id = @comment_id AND entity_type = @entity_type;

-- name: DeleteCommentVote :exec
DELETE FROM comment_vote
WHERE id = $1;

-- name: UpsertCommentVote :one
INSERT INTO comment_vote (comment_id, entity_type, user_id, vote_type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (entity_type, comment_id, user_id, vote_type)
DO UPDATE SET
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetCommentVoteCountByCommentID :many
SELECT
    comment_id,
    entity_type,
    vote_type,
    total
FROM comment_vote_count_mv
WHERE comment_id = $1 AND entity_type = $2;

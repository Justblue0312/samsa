-- name: UpsertStoryVote :one
INSERT INTO story_vote (
    story_id, user_id, rating, created_at, updated_at
) VALUES (
    $1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT (story_id, user_id) DO UPDATE
SET 
    rating = EXCLUDED.rating,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetStoryVote :one
SELECT * FROM story_vote WHERE story_id = $1 AND user_id = $2;

-- name: DeleteStoryVote :exec
DELETE FROM story_vote WHERE story_id = $1 AND user_id = $2;

-- name: GetStoryVoteStats :one
SELECT
    COUNT(*) as total_votes,
    AVG(rating)::float4 as average_rating
FROM story_vote
WHERE story_id = $1;

-- name: GetStoryVoteByID :one
SELECT * FROM story_vote WHERE id = $1;

-- name: ListStoryVotesByStory :many
SELECT * FROM story_vote
WHERE story_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoryVotesByStory :one
SELECT COUNT(*) FROM story_vote WHERE story_id = $1;

-- name: ListStoryVotesByUser :many
SELECT * FROM story_vote
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoryVotesByUser :one
SELECT COUNT(*) FROM story_vote WHERE user_id = $1;

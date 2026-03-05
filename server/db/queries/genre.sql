-- name: CreateGenre :one
INSERT INTO genre (name, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetGenreByID :one
SELECT * FROM genre WHERE id = $1;

-- name: GetGenreByName :one
SELECT * FROM genre WHERE name = $1;

-- name: ListGenres :many
SELECT * FROM genre ORDER BY name ASC;

-- name: UpdateGenre :one
UPDATE genre
SET
    name = $2,
    description = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteGenre :exec
DELETE FROM genre WHERE id = $1;

-- name: AddGenreToStory :exec
INSERT INTO story_genre (story_id, genre_id)
VALUES ($1, $2)
ON CONFLICT (story_id, genre_id) DO NOTHING;

-- name: RemoveGenreFromStory :exec
DELETE FROM story_genre
WHERE story_id = $1 AND genre_id = $2;

-- name: GetGenresByStoryID :many
SELECT g.* FROM genre g
JOIN story_genre sg ON g.id = sg.genre_id
WHERE sg.story_id = $1;

-- name: ClearGenresFromStory :exec
DELETE FROM story_genre WHERE story_id = $1;

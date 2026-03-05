-- name: GetAuthorByID :one
SELECT * FROM author WHERE id = @id AND is_deleted = @is_deleted;

-- name: GetAuthorBySlug :one
SELECT * FROM author WHERE slug = @slug AND is_deleted = @is_deleted;

-- name: GetAuthorByStageName :one
SELECT * FROM author WHERE stage_name = @stage_name AND is_deleted = @is_deleted;

-- name: GetAuthorByUserID :one
SELECT * FROM author WHERE user_id = @user_id AND is_deleted = @is_deleted;

-- name: CreateAuthor :one
INSERT INTO author (user_id, media_id, stage_name, gender, slug, first_name, last_name, dob, phone, bio, description, accepted_terms_of_service, email_newsletters_and_changelogs, email_promotions_and_events, is_recommended, is_deleted, stats, created_at, updated_at, deleted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING *;

-- name: UpdateAuthor :one
UPDATE author
SET
    user_id = $2,
    media_id = $3,
    stage_name = $4,
    gender = $5,
    slug = $6,
    first_name = $7,
    last_name = $8,
    dob = $9,
    phone = $10,
    bio = $11,
    description = $12,
    accepted_terms_of_service = $13,
    email_newsletters_and_changelogs = $14,
    email_promotions_and_events = $15,
    is_recommended = $16,
    is_deleted = $17,
    stats = $18,
    deleted_at = $19
WHERE id = $1
RETURNING *;

-- name: SoftDeleteAuthor :exec
UPDATE author SET is_deleted = true, deleted_at = now() WHERE id = $1 AND user_id = $2;

-- name: RestoreAuthor :exec
UPDATE author SET is_deleted = false, deleted_at = null WHERE id = $1 AND user_id = $2;

-- name: DeleteAuthor :exec
DELETE FROM author WHERE id = $1 AND user_id = $2;

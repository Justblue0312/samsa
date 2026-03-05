-- name: GetUserByID :one
SELECT * FROM "user" WHERE id = @user_id AND is_deleted = @is_deleted;

-- name: GetUserByEmail :one
SELECT * FROM "user" WHERE email = @email AND is_deleted = @is_deleted;

-- name: CreateUser :one
INSERT INTO "user" (email, email_verified, password_hash, is_deleted, is_active, is_admin, is_staff, is_author, is_banned, banned_at, ban_reason, last_login_at, rate_limit_group, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: UpdateUser :one
UPDATE "user"
SET
    email = $2,
    email_verified = $3,
    password_hash = $4,
    is_deleted = $5,
    is_active = $6,
    is_admin = $7,
    is_staff = $8,
    is_author = $9,
    is_banned = $10,
    banned_at = $11,
    ban_reason = $12,
    last_login_at = $13,
    rate_limit_group = $14,
    deleted_at = $15
WHERE id = $1
RETURNING *;

-- name: GetUserByOAuthAccount :one
SELECT "user".*
FROM "user"
JOIN oauth_account ON "user".id = oauth_account.user_id
WHERE
    oauth_account.account_id = @account_id
    AND oauth_account.provider = @provider
    AND "user".is_deleted = @is_deleted
LIMIT 1;

-- name: GetAccountByID :one
SELECT * FROM oauth_account WHERE id = $1;

-- name: GetAccountsByUserID :many
SELECT * FROM oauth_account
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetAccountByProviderAndAccountID :one
SELECT * FROM oauth_account
WHERE provider = @provider AND account_id = @account_id
LIMIT 1;

-- name: GetAccountByProviderAndUserID :many
SELECT * FROM oauth_account
WHERE provider = @provider AND user_id = @user_id;


-- name: CountOtherAccounts :one
SELECT count(oauth_account.id) FROM oauth_account
WHERE user_id = @user_id AND id NOT IN (sqlc.slice(account_ids));

-- name: DeleteAccountByID :exec
DELETE FROM oauth_account WHERE id = $1;

-- name: DeleteAccountByUserIDAndProvider :exec
DELETE FROM oauth_account
WHERE user_id = @user_id AND provider = @provider;

-- name: CreateAccount :one
INSERT INTO oauth_account (user_id, provider, access_token, expires_at, refresh_token, refresh_token_expires_at, account_id, account_email, account_username, account_avatar_url, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateAccount :one
UPDATE oauth_account
SET
    user_id = $2,
    provider = $3,
    access_token = $4,
    expires_at = $5,
    refresh_token = $6,
    refresh_token_expires_at = $7,
    account_id = $8,
    account_email = $9,
    account_username = $10,
    account_avatar_url = $11
WHERE id = $1
RETURNING *;

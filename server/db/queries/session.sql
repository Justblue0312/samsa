-- name: CreateSession :one
INSERT INTO session (user_id, token, ip_address, user_agent, device_info, scopes, metadata, is_active, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetSession :one
SELECT *
FROM session
WHERE
    token = @token
    AND is_active = TRUE
    AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

-- name: GetSessionByToken :one
SELECT sqlc.embed(s), sqlc.embed(u)
FROM session AS s
JOIN "user" AS u ON s.user_id = u.id
WHERE s.token = @token
AND (
  @is_expired::bool
  OR (s.expires_at IS NOT NULL AND s.expires_at > CURRENT_TIMESTAMP)
);

-- name: DeleteExpiredSessions :exec
DELETE FROM session
WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;

-- name: DeleteSessionsByUserId :exec
-- TODO: This will delete all sessions of the user, including active ones. Consider if we want to only delete expired sessions of the user.
-- Might including device info or other metadata to only delete sessions from specific devices or locations.
DELETE FROM session
WHERE user_id = $1;

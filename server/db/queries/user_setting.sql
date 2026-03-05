-- name: GetUserSettingByKey :one
SELECT * FROM user_setting
WHERE user_id = @user_id and key = @key LIMIT 1;

-- name: GetUserSettings :many
SELECT * FROM user_setting WHERE user_id = @user_id;

-- name: UpdateUserSetting :one
INSERT INTO user_setting (user_id, key, value)
VALUES (@user_id, @key, @value::JSONB)
ON CONFLICT (user_id, key) DO UPDATE
SET value = EXCLUDED.value
RETURNING *;

-- name: DeleteUserSetting :exec
DELETE FROM user_setting
WHERE
    user_id = @user_id
    AND key = @key;

-- name: CreateSubmissionStatusHistory :one
INSERT INTO submission_status_history (submission_id, changed_by, old_status, new_status, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSubmissionStatusHistoryBySubmissionID :many
SELECT * FROM submission_status_history
WHERE submission_id = $1
ORDER BY created_at DESC;

-- name: GetSubmissionStatusHistoryByID :one
SELECT * FROM submission_status_history
WHERE id = $1;

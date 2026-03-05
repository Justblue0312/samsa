-- name: CreateSubmissionAssignment :one
INSERT INTO submission_assignment (submission_id, assigned_by, assigned_to, assigned_at, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSubmissionAssignment :one
SELECT * FROM submission_assignment
WHERE id = $1;

-- name: GetSubmissionAssignmentBySubmissionID :one
SELECT * FROM submission_assignment
WHERE submission_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetSubmissionAssignmentsByAssignedTo :many
SELECT * FROM submission_assignment
WHERE assigned_to = $1
ORDER BY created_at DESC;

-- name: GetSubmissionAssignmentsByAssignedBy :many
SELECT * FROM submission_assignment
WHERE assigned_by = $1
ORDER BY created_at DESC;

-- name: UpdateSubmissionAssignment :one
UPDATE submission_assignment
SET
    assigned_by = $2,
    assigned_to = $3,
    assigned_at = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

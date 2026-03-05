-- name: GetSubmissionByID :one
SELECT * FROM submission WHERE id = @id AND is_deleted = @is_deleted;

-- name: GetSubmissionByExposeID :one
SELECT * FROM submission WHERE expose_id = @expose_id AND is_deleted = @expose_id;

-- name: CreateSubmission :one
INSERT INTO submission (
    requester_id, approver_id, approved_at,
    message, title, type,
    is_deleted, status, context,
    created_at, updated_at, deleted_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateSubmission :one
UPDATE submission
SET
    requester_id = $2,
    approver_id  = $3,
    approved_at  = $4,
    message      = $5,
    title        = $6,
    type         = $7,
    is_deleted   = $8,
    status       = $9,
    context      = $10,
    deleted_at   = $11,
    updated_at   = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateSubmissionStatus :one
UPDATE submission
SET
    status     = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateSubmissionApproved :one
UPDATE submission
SET
    status      = 'approved',
    approved_at = $2,
    approver_id = $3,
    updated_at  = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteSubmission :exec
DELETE FROM submission WHERE id = $1;

-- name: SoftDeleteSubmission :one
UPDATE submission
SET
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP,
    is_deleted = TRUE
WHERE id = $1
RETURNING *;

-- name: ListSubmissions :many
SELECT * FROM submission
WHERE is_deleted = @is_deleted
    AND (@requester_id::uuid IS NULL OR requester_id = @requester_id::uuid)
    AND (@approver_id::uuid  IS NULL OR approver_id  = @approver_id::uuid)
    AND (@type::submission_type IS NULL OR type = @type::submission_type)
    AND (@status::submission_status IS NULL OR status = @status::submission_status)
    AND (@expose_id::text    IS NULL OR expose_id     = @expose_id::text)
    AND (@title::text        IS NULL OR title ILIKE '%' || @title::text || '%')
    AND (@search::text       IS NULL OR search_vector @@ plainto_tsquery('english', @search::text))
ORDER BY
    CASE WHEN @order_by = 'created_at:asc'  THEN created_at END ASC,
    CASE WHEN @order_by = 'created_at:desc' THEN created_at END DESC,
    CASE WHEN @order_by = 'updated_at:asc'  THEN updated_at END ASC,
    CASE WHEN @order_by = 'updated_at:desc' THEN updated_at END DESC,
    CASE WHEN @order_by = 'title:asc'       THEN title      END ASC,
    CASE WHEN @order_by = 'title:desc'      THEN title      END DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountSubmissions :one
SELECT COUNT(*) FROM submission
WHERE is_deleted = @is_deleted
    AND (@requester_id::uuid IS NULL OR requester_id = @requester_id::uuid)
    AND (@approver_id::uuid  IS NULL OR approver_id  = @approver_id::uuid)
    AND (@type::submission_type IS NULL OR type = @type::submission_type)
    AND (@status::submission_status IS NULL OR status = @status::submission_status)
    AND (@expose_id::text    IS NULL OR expose_id     = @expose_id::text)
    AND (@title::text        IS NULL OR title ILIKE '%' || @title::text || '%')
    AND (@search::text       IS NULL OR search_vector @@ plainto_tsquery('english', @search::text));

-- name: GetSubmissionsByRequesterID :many
SELECT * FROM submission
WHERE requester_id = @requester_id AND is_deleted = @is_deleted
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetSubmissionsByApproverID :many
SELECT * FROM submission
WHERE approver_id = @approver_id AND is_deleted = @is_deleted
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetAvailableSubmissions :many
SELECT * FROM submission
WHERE is_deleted = @is_deleted
    AND status = 'pending'
    AND approver_id IS NULL
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountAvailableSubmissions :one
SELECT COUNT(*) FROM submission
WHERE is_deleted = @is_deleted
    AND status = 'pending'
    AND approver_id IS NULL;

-- name: GetTimeoutedCandidates :many
SELECT * FROM submission
WHERE status = 'pending'
  AND updated_at < NOW() - ($1 || ' days')::INTERVAL
  AND is_deleted = @is_deleted
ORDER BY updated_at ASC;

-- name: BulkMarkTimeouted :many
UPDATE submission
SET status     = 'timeouted',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::uuid[])
RETURNING *;

-- name: GetArchiveCandidates :many
SELECT * FROM submission
WHERE status IN ('approved', 'rejected', 'timeouted')
  AND updated_at < NOW() - (@days || ' days')::INTERVAL
  AND is_deleted = @is_deleted
ORDER BY updated_at ASC;

-- name: BulkMarkArchived :many
UPDATE submission
SET status     = 'archived',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::uuid[])
RETURNING *;

-- SLA Tracking Queries

-- name: GetSubmissionsExceedingSLA :many
SELECT * FROM submission
WHERE status = 'pending'
  AND is_deleted = FALSE
  AND created_at < NOW() - (@sla_hours || ' hours')::INTERVAL
ORDER BY created_at ASC;

-- name: CountSubmissionsExceedingSLA :one
SELECT COUNT(*) FROM submission
WHERE status = 'pending'
  AND is_deleted = FALSE
  AND created_at < NOW() - (@sla_hours || ' hours')::INTERVAL;

-- name: GetSLAComplianceStats :one
SELECT 
    COUNT(*) FILTER (WHERE status = 'pending' AND created_at >= NOW() - (@sla_hours || ' hours')::INTERVAL)::int as compliant_count,
    COUNT(*) FILTER (WHERE status = 'pending' AND created_at < NOW() - (@sla_hours || ' hours')::INTERVAL)::int as non_compliant_count,
    COUNT(*) FILTER (WHERE status = 'pending')::int as total_pending
FROM submission
WHERE is_deleted = FALSE;

-- name: GetAverageProcessingTime :one
SELECT 
    COALESCE(AVG(EXTRACT(EPOCH FROM (approved_at - created_at))), 0)::float as avg_seconds
FROM submission
WHERE status = 'approved'
  AND approved_at IS NOT NULL
  AND is_deleted = FALSE
  AND created_at >= NOW() - (@days || ' days')::INTERVAL;

-- name: GetSubmissionsBySLAStatus :many
SELECT * FROM submission
WHERE status = 'pending'
  AND is_deleted = FALSE
  AND CASE 
        WHEN @include_compliant::boolean THEN created_at >= NOW() - (@sla_hours || ' hours')::INTERVAL
        ELSE created_at < NOW() - (@sla_hours || ' hours')::INTERVAL
      END
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPendingDuration :one
SELECT 
    EXTRACT(EPOCH FROM (NOW() - created_at))::int as pending_seconds
FROM submission
WHERE id = $1 AND status = 'pending' AND is_deleted = FALSE;

-- name: BulkUpdateSLABreach :many
UPDATE submission
SET 
    metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('sla_breach', true, 'sla_breach_at', NOW()),
    updated_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::uuid[])
RETURNING *;

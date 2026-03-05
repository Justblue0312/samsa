-- name: CreateStoryReport :one
INSERT INTO story_report (
    story_id, chapter_id, reporter_id, title, description, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetStoryReportByID :one
SELECT * FROM story_report WHERE id = $1;

-- name: ListStoryReports :many
SELECT * FROM story_report 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;

-- name: UpdateStoryReportStatus :one
UPDATE story_report
SET
    status = $2,
    is_resolved = $3,
    resolved_at = $4,
    resolved_by = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetStoryReportByStoryAndReporter :one
SELECT * FROM story_report
WHERE story_id = $1 AND reporter_id = $2;

-- name: UpdateStoryReport :one
UPDATE story_report
SET
    title = $2,
    description = $3,
    status = $4,
    is_resolved = $5,
    resolved_at = $6,
    resolved_by = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteStoryReport :exec
DELETE FROM story_report WHERE id = $1;

-- name: ListStoryReportsByStory :many
SELECT * FROM story_report
WHERE story_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoryReportsByStory :one
SELECT COUNT(*) FROM story_report WHERE story_id = $1;

-- name: ListStoryReportsByReporter :many
SELECT * FROM story_report
WHERE reporter_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoryReportsByReporter :one
SELECT COUNT(*) FROM story_report WHERE reporter_id = $1;

-- name: ListPendingStoryReports :many
SELECT * FROM story_report
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPendingStoryReports :one
SELECT COUNT(*) FROM story_report WHERE status = 'pending';

-- name: ListStoryReportsWithFilters :many
SELECT * FROM story_report
WHERE (sqlc.narg('story_id')::uuid IS NULL OR story_id = sqlc.narg('story_id')::uuid)
  AND (sqlc.narg('reporter_id')::uuid IS NULL OR reporter_id = sqlc.narg('reporter_id')::uuid)
  AND (sqlc.narg('status')::report_status IS NULL OR status = sqlc.narg('status')::report_status)
  AND (sqlc.narg('is_resolved')::boolean IS NULL OR is_resolved = sqlc.narg('is_resolved'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountStoryReportsWithFilters :one
SELECT COUNT(*) FROM story_report
WHERE (sqlc.narg('story_id')::uuid IS NULL OR story_id = sqlc.narg('story_id')::uuid)
  AND (sqlc.narg('reporter_id')::uuid IS NULL OR reporter_id = sqlc.narg('reporter_id')::uuid)
  AND (sqlc.narg('status')::report_status IS NULL OR status = sqlc.narg('status')::report_status)
  AND (sqlc.narg('is_resolved')::boolean IS NULL OR is_resolved = sqlc.narg('is_resolved'));

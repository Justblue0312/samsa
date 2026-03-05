-- name: GetDocumentByID :one
SELECT * FROM document WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDocumentBySlug :one
SELECT * FROM document WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetDocumentsByOwnerID :many
SELECT * FROM document WHERE created_by = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetDocumentsByFolderID :many
SELECT * FROM document WHERE folder_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetDocumentsByStatus :many
SELECT * FROM document WHERE status = $1 AND deleted_at IS NULL ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: GetDocumentsByStoryID :many
SELECT * FROM document WHERE story_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetPendingReviewDocuments :many
SELECT * FROM document WHERE status = 'pending_review' AND deleted_at IS NULL ORDER BY created_at ASC LIMIT $1 OFFSET $2;

-- name: CreateDocument :one
INSERT INTO document (story_id, created_by, folder_id, language, branch_name, version_number, content, title, slug, summary, document_type, status, is_template, total_words, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateDocument :one
UPDATE document
SET
    folder_id = $2,
    language = $3,
    branch_name = $4,
    version_number = $5,
    content = $6,
    title = $7,
    slug = $8,
    summary = $9,
    document_type = $10,
    status = $11,
    is_locked = $12,
    is_template = $13,
    previous_version_id = $14,
    total_words = $15,
    total_views = $16,
    total_downloads = $17,
    total_shares = $18,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteDocument :exec
DELETE FROM document WHERE id = $1;

-- name: SoftDeleteDocument :one
UPDATE document SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: SubmitDocumentForReview :one
UPDATE document
SET status = 'pending_review', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ReviewDocument :one
UPDATE document
SET status = $2, is_locked = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ApproveDocument :one
UPDATE document
SET status = 'is_approved', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: RejectDocument :one
UPDATE document
SET status = 'rejected', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ArchiveDocument :one
UPDATE document
SET status = 'archived', updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: IncrementDocumentViews :one
UPDATE document
SET total_views = COALESCE(total_views, 0) + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: IncrementDocumentDownloads :one
UPDATE document
SET total_downloads = COALESCE(total_downloads, 0) + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: IncrementDocumentShares :one
UPDATE document
SET total_shares = COALESCE(total_shares, 0) + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateDocumentVersion :one
UPDATE document
SET
    version_number = $2,
    previous_version_id = $3,
    content = $4,
    total_words = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetDocumentVersionHistory :many
SELECT d.* FROM document d
WHERE d.story_id = (SELECT d2.story_id FROM document d2 WHERE d2.id = $1)
  AND d.created_by = (SELECT d3.created_by FROM document d3 WHERE d3.id = $1)
  AND d.deleted_at IS NULL
ORDER BY d.created_at DESC;

-- name: CreateDocumentStatusHistory :one
INSERT INTO document_status_history (document_id, set_status_by, content, status, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListDocumentStatusHistory :many
SELECT * FROM document_status_history WHERE document_id = $1 ORDER BY created_at DESC;

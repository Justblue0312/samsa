-- name: AddChapterToDocument :one
INSERT INTO document_chapter (document_id, chapter_id, sort_order, added_by, notes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: RemoveChapterFromDocument :exec
DELETE FROM document_chapter WHERE document_id = $1 AND chapter_id = $2;

-- name: GetChaptersByDocumentID :many
SELECT c.*, dc.sort_order, dc.notes FROM chapter c
INNER JOIN document_chapter dc ON c.id = dc.chapter_id
WHERE dc.document_id = $1 AND c.deleted_at IS NULL
ORDER BY dc.sort_order;

-- name: GetDocumentByChapterID :many
SELECT d.* FROM document d
INNER JOIN document_chapter dc ON d.id = dc.document_id
WHERE dc.chapter_id = $1 AND d.deleted_at IS NULL;

-- name: ReorderDocumentChapters :exec
UPDATE document_chapter
SET sort_order = $3, updated_at = CURRENT_TIMESTAMP
WHERE document_id = $1 AND chapter_id = $2;

-- name: GetNextDocumentChapterSortOrder :one
SELECT COALESCE(MAX(sort_order), -1) + 1 FROM document_chapter WHERE document_id = $1;

-- name: RecordDocumentView :one
INSERT INTO document_view (document_id, user_id, viewed_at, view_duration, completion_percentage, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (document_id, user_id) DO UPDATE SET
    viewed_at = EXCLUDED.viewed_at,
    view_duration = COALESCE(EXCLUDED.view_duration, document_view.view_duration),
    completion_percentage = GREATEST(document_view.completion_percentage, EXCLUDED.completion_percentage)
RETURNING *;

-- name: GetDocumentViewByUser :one
SELECT * FROM document_view WHERE document_id = $1 AND user_id = $2;

-- name: GetDocumentViewStats :one
SELECT COUNT(*) as total_views, COUNT(DISTINCT user_id) as unique_viewers
FROM document_view WHERE document_id = $1;

-- name: ShareDocument :one
INSERT INTO document_share (document_id, shared_by, shared_with_email, shared_with_user_id, share_token, share_message, expires_at, can_view, can_download, can_comment, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetDocumentShareByToken :one
SELECT * FROM document_share WHERE share_token = $1 AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);

-- name: GetDocumentShares :many
SELECT * FROM document_share WHERE document_id = $1 ORDER BY created_at DESC;

-- name: CreateDocumentComment :one
INSERT INTO document_comment (document_id, parent_comment_id, author_id, content, line_number, character_offset, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetDocumentComments :many
SELECT * FROM document_comment WHERE document_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: GetDocumentCommentReplies :many
SELECT * FROM document_comment WHERE parent_comment_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: ResolveDocumentComment :one
UPDATE document_comment
SET resolved = true, resolved_by = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteDocumentComment :one
UPDATE document_comment SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: GetUnresolvedDocumentComments :many
SELECT * FROM document_comment WHERE document_id = $1 AND resolved = false AND deleted_at IS NULL;

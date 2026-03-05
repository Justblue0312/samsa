-- name: GetDocumentFolderByID :one
SELECT * FROM document_folder WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDocumentFoldersByParentID :many
SELECT * FROM document_folder WHERE parent_id = $1 AND deleted_at IS NULL ORDER BY name;

-- name: GetRootDocumentFolders :many
SELECT * FROM document_folder WHERE parent_id IS NULL AND deleted_at IS NULL ORDER BY name;

-- name: GetDocumentFoldersByStoryID :many
SELECT * FROM document_folder WHERE story_id = $1 AND deleted_at IS NULL ORDER BY depth, name;

-- name: GetDocumentFoldersByOwnerID :many
SELECT * FROM document_folder WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: CreateDocumentFolder :one
INSERT INTO document_folder (story_id, owner_id, name, parent_id, depth, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateDocumentFolder :one
UPDATE document_folder
SET
    name = $2,
    parent_id = $3,
    depth = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteDocumentFolder :exec
DELETE FROM document_folder WHERE id = $1;

-- name: SoftDeleteDocumentFolder :one
UPDATE document_folder SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING *;

-- name: MoveDocumentFolder :one
UPDATE document_folder
SET parent_id = $2, depth = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetChildFoldersCount :one
SELECT COUNT(*) FROM document_folder WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: GetFolderDocumentsCount :one
SELECT COUNT(*) FROM document WHERE folder_id = $1 AND deleted_at IS NULL;

-- name: GetAncestorFolders :many
WITH RECURSIVE ancestors AS (
    SELECT df.id, df.parent_id, df.name, df.depth, 0 as distance
    FROM document_folder df
    WHERE df.id = $1 AND df.deleted_at IS NULL
    UNION ALL
    SELECT df.id, df.parent_id, df.name, df.depth, a.distance + 1
    FROM document_folder df
    INNER JOIN ancestors a ON df.id = a.parent_id
    WHERE df.deleted_at IS NULL
)
SELECT * FROM ancestors WHERE distance > 0 ORDER BY distance DESC;

-- name: GetDescendantFolders :many
WITH RECURSIVE descendants AS (
    SELECT df.id, df.parent_id, df.name, df.depth, 0 as child_depth
    FROM document_folder df
    WHERE df.id = $1 AND df.deleted_at IS NULL
    UNION ALL
    SELECT df.id, df.parent_id, df.name, df.depth, d.child_depth + 1
    FROM document_folder df
    INNER JOIN descendants d ON df.parent_id = d.id
    WHERE df.deleted_at IS NULL AND d.child_depth < 2
)
SELECT * FROM descendants WHERE child_depth > 0 ORDER BY depth, name;

-- name: GetFolderTree :many
WITH RECURSIVE folder_tree AS (
    SELECT df.id, df.parent_id, df.name, df.depth, 0 as tree_depth
    FROM document_folder df
    WHERE df.id = $1 AND df.deleted_at IS NULL
    UNION ALL
    SELECT df.id, df.parent_id, df.name, df.depth, ft.tree_depth + 1
    FROM document_folder df
    INNER JOIN folder_tree ft ON df.parent_id = ft.id
    WHERE df.deleted_at IS NULL AND ft.tree_depth < 2
)
SELECT * FROM folder_tree ORDER BY tree_depth, name;

-- name: ValidateFolderDepth :one
SELECT COALESCE(
    (SELECT depth FROM document_folder WHERE id = $1 AND deleted_at IS NULL),
    -1
) + 1 AS result;

-- name: GetSiblings :many
SELECT * FROM document_folder
WHERE parent_id = $1 AND id != $2 AND deleted_at IS NULL
ORDER BY name;

-- name: SearchDocumentFolders :many
SELECT * FROM document_folder
WHERE name ILIKE '%' || $1 || '%' AND deleted_at IS NULL
ORDER BY depth, name
LIMIT $2 OFFSET $3;

-- name: GetFolderWithPath :many
WITH RECURSIVE folder_path AS (
    SELECT df.id, df.parent_id, df.name, df.depth, ARRAY[df.name] as path_names, ARRAY[df.id] as path_ids
    FROM document_folder df
    WHERE df.id = $1 AND df.deleted_at IS NULL
    UNION ALL
    SELECT df.id, df.parent_id, df.name, df.depth, fp.path_names || df.name, fp.path_ids || df.id
    FROM document_folder df
    INNER JOIN folder_path fp ON df.id = fp.parent_id
    WHERE df.deleted_at IS NULL
)
SELECT * FROM folder_path ORDER BY depth DESC LIMIT 1;

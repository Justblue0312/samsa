package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// DocumentFolderOpts controls which fields are customised when creating a test document folder.
type DocumentFolderOpts struct {
	StoryID  uuid.UUID
	OwnerID  uuid.UUID
	Name     string
	ParentID *uuid.UUID
	Depth    *int32
}

// DocumentFolder creates and inserts a document folder into the DB and returns the created model.
func DocumentFolder(t *testing.T, db sqlc.DBTX, opts DocumentFolderOpts) *sqlc.DocumentFolder {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
	}
	if opts.OwnerID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.OwnerID = user.ID
	}
	if opts.Name == "" {
		opts.Name = "test-folder-" + randID()
	}
	if opts.Depth == nil {
		opts.Depth = Int32Ptr(0)
	}

	q := sqlc.New(db)
	n := now()

	folder, err := q.CreateDocumentFolder(context.Background(), sqlc.CreateDocumentFolderParams{
		StoryID:   opts.StoryID,
		OwnerID:   opts.OwnerID,
		Name:      opts.Name,
		ParentID:  opts.ParentID,
		Depth:     *opts.Depth,
		CreatedAt: n,
		UpdatedAt: n,
	})
	require.NoError(t, err, "factory: failed to create test document folder")

	return &folder
}

// RootDocumentFolder creates a root-level folder (depth 0, no parent).
func RootDocumentFolder(t *testing.T, db sqlc.DBTX, opts DocumentFolderOpts) *sqlc.DocumentFolder {
	t.Helper()

	opts.ParentID = nil
	opts.Depth = Int32Ptr(0)
	return DocumentFolder(t, db, opts)
}

// ChildDocumentFolder creates a child folder under the given parent.
func ChildDocumentFolder(t *testing.T, db sqlc.DBTX, parent *sqlc.DocumentFolder, opts DocumentFolderOpts) *sqlc.DocumentFolder {
	t.Helper()

	if opts.Name == "" {
		opts.Name = "child-folder-" + randID()
	}
	opts.ParentID = &parent.ID
	opts.Depth = Int32Ptr(parent.Depth + 1)
	opts.StoryID = parent.StoryID

	return DocumentFolder(t, db, opts)
}

// DocumentFolderWithDepth creates a folder at a specific depth level.
func DocumentFolderWithDepth(t *testing.T, db sqlc.DBTX, opts DocumentFolderOpts, depth int32) *sqlc.DocumentFolder {
	t.Helper()

	if depth == 0 {
		return RootDocumentFolder(t, db, opts)
	}

	// Create parent hierarchy
	var parent *sqlc.DocumentFolder
	if depth == 1 {
		parent = RootDocumentFolder(t, db, DocumentFolderOpts{
			StoryID: opts.StoryID,
			OwnerID: opts.OwnerID,
		})
	} else {
		parent = DocumentFolderWithDepth(t, db, opts, depth-1)
	}

	opts.Depth = Int32Ptr(depth)
	return ChildDocumentFolder(t, db, parent, opts)
}

// NestedDocumentFolders creates a nested folder structure and returns all folders.
// depth=1 creates root only, depth=2 creates root+child, etc.
func NestedDocumentFolders(t *testing.T, db sqlc.DBTX, opts DocumentFolderOpts, depth int) []*sqlc.DocumentFolder {
	t.Helper()

	if depth <= 0 {
		return []*sqlc.DocumentFolder{}
	}

	folders := make([]*sqlc.DocumentFolder, 0, depth)

	// Create root
	root := RootDocumentFolder(t, db, DocumentFolderOpts{
		StoryID: opts.StoryID,
		OwnerID: opts.OwnerID,
		Name:    "root-" + randID(),
	})
	folders = append(folders, root)

	// Create nested children
	for i := 1; i < depth; i++ {
		child := ChildDocumentFolder(t, db, folders[i-1], DocumentFolderOpts{
			StoryID: opts.StoryID,
			OwnerID: opts.OwnerID,
			Name:    "level-" + randID(),
		})
		folders = append(folders, child)
	}

	return folders
}

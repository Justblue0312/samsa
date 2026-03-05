package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// DocumentOpts controls which fields are customised when creating a test document.
type DocumentOpts struct {
	StoryID       uuid.UUID
	CreatedBy     uuid.UUID
	FolderID      *uuid.UUID
	Language      string
	BranchName    string
	Title         *string
	Slug          *string
	Summary       *string
	Content       []byte
	DocumentType  *string
	Status        sqlc.DocumentStatus
	IsLocked      *bool
	IsTemplate    *bool
	TotalWords    *int32
	VersionNumber *int32
}

// Document creates and inserts a document into the DB and returns the created model.
func Document(t *testing.T, db sqlc.DBTX, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
	}
	if opts.CreatedBy == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.CreatedBy = user.ID
	}
	if opts.Language == "" {
		opts.Language = "eng"
	}
	if opts.BranchName == "" {
		opts.BranchName = "main-" + randID()
	}
	if opts.Title == nil {
		opts.Title = StringPtr("test-document-" + randID())
	}
	if opts.Slug == nil {
		opts.Slug = StringPtr("test-document-" + randID())
	}
	if opts.Status == "" {
		opts.Status = sqlc.DocumentStatusDraft
	}
	if opts.IsLocked == nil {
		opts.IsLocked = BoolPtr(false)
	}
	if opts.IsTemplate == nil {
		opts.IsTemplate = BoolPtr(false)
	}
	if opts.TotalWords == nil {
		opts.TotalWords = Int32Ptr(0)
	}
	if opts.VersionNumber == nil {
		opts.VersionNumber = Int32Ptr(1)
	}

	q := sqlc.New(db)
	n := now()

	document, err := q.CreateDocument(context.Background(), sqlc.CreateDocumentParams{
		StoryID:       opts.StoryID,
		CreatedBy:     opts.CreatedBy,
		FolderID:      opts.FolderID,
		Language:      opts.Language,
		BranchName:    opts.BranchName,
		VersionNumber: *opts.VersionNumber,
		Content:       opts.Content,
		Title:         opts.Title,
		Slug:          opts.Slug,
		Summary:       opts.Summary,
		DocumentType:  opts.DocumentType,
		Status:        opts.Status,
		IsTemplate:    opts.IsTemplate,
		TotalWords:    opts.TotalWords,
		CreatedAt:     n,
		UpdatedAt:     n,
	})
	require.NoError(t, err, "factory: failed to create test document")

	return &document
}

// DocumentWithFolder creates a document with a new folder.
func DocumentWithFolder(t *testing.T, db sqlc.DBTX, opts DocumentOpts) (*sqlc.Document, *sqlc.DocumentFolder) {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
	}
	if opts.CreatedBy == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.CreatedBy = user.ID
	}

	folder := DocumentFolder(t, db, DocumentFolderOpts{
		StoryID: opts.StoryID,
		OwnerID: opts.CreatedBy,
	})
	opts.FolderID = &folder.ID

	return Document(t, db, opts), folder
}

// DocumentInFolder creates a document in an existing folder.
func DocumentInFolder(t *testing.T, db sqlc.DBTX, folder *sqlc.DocumentFolder, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	if opts.CreatedBy == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.CreatedBy = user.ID
	}

	opts.FolderID = &folder.ID
	opts.StoryID = folder.StoryID

	return Document(t, db, opts)
}

// PendingReviewDocument creates a document with pending_review status.
func PendingReviewDocument(t *testing.T, db sqlc.DBTX, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	opts.Status = sqlc.DocumentStatusPendingReview
	return Document(t, db, opts)
}

// ApprovedDocument creates a document with is_approved status.
func ApprovedDocument(t *testing.T, db sqlc.DBTX, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	opts.Status = sqlc.DocumentStatusIsApproved
	return Document(t, db, opts)
}

// RejectedDocument creates a document with rejected status.
func RejectedDocument(t *testing.T, db sqlc.DBTX, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	opts.Status = sqlc.DocumentStatusRejected
	return Document(t, db, opts)
}

// ArchivedDocument creates a document with archived status.
func ArchivedDocument(t *testing.T, db sqlc.DBTX, opts DocumentOpts) *sqlc.Document {
	t.Helper()

	opts.Status = sqlc.DocumentStatusArchived
	return Document(t, db, opts)
}

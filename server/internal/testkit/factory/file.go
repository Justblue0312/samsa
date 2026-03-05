package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// FileOpts controls which fields are customised when creating a test file.
// Any zero-value field gets a sensible default.
type FileOpts struct {
	OwnerID    uuid.UUID
	Name       string
	Path       string
	MimeType   *string
	Size       int64
	Reference  string
	Payload    string
	Service    *string
	Source     sqlc.FileUploadSource
	IsArchived bool
}

// File inserts a file into the DB and returns the created model.
// If OwnerID is zero, a new test user is created automatically.
func File(t *testing.T, db sqlc.DBTX, opts FileOpts) *sqlc.File {
	t.Helper()

	if opts.OwnerID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.OwnerID = user.ID
	}
	if opts.Name == "" {
		opts.Name = "test-file-" + randID() + ".txt"
	}
	if opts.Path == "" {
		opts.Path = "/uploads/" + opts.Name
	}
	if opts.MimeType == nil {
		mt := "text/plain"
		opts.MimeType = &mt
	}
	if opts.Size == 0 {
		opts.Size = 1024
	}
	if opts.Reference == "" {
		opts.Reference = "https://example.com/files/" + opts.Name
	}
	if opts.Payload == "" {
		opts.Payload = "{}"
	}
	if opts.Source == "" {
		opts.Source = sqlc.FileUploadSourceFile
	}

	q := sqlc.New(db)
	n := *now()

	file, err := q.CreateFile(context.Background(), sqlc.CreateFileParams{
		OwnerID:    opts.OwnerID,
		Name:       opts.Name,
		Path:       opts.Path,
		MimeType:   opts.MimeType,
		Size:       opts.Size,
		Reference:  opts.Reference,
		Payload:    opts.Payload,
		Service:    opts.Service,
		Source:     opts.Source,
		IsArchived: opts.IsArchived,
		CreatedAt:  n,
		UpdatedAt:  n,
	})
	require.NoError(t, err, "factory: failed to create test file")

	return &file
}

// SharedFileOpts controls which fields are customised when creating a test shared file scenario.
type SharedFileOpts struct {
	FileID     uuid.UUID
	OwnerID    uuid.UUID
	SharedWith uuid.UUID
	Name       string
	MimeType   *string
	Size       int64
}

// SharedFile creates a file and returns it along with the owner and recipient users.
// The caller is responsible for testing the sharing logic separately.
func SharedFile(t *testing.T, db sqlc.DBTX, opts SharedFileOpts) (*sqlc.File, *sqlc.User, *sqlc.User) {
	t.Helper()

	if opts.OwnerID == uuid.Nil {
		owner := User(t, db, UserOpts{})
		opts.OwnerID = owner.ID
	}
	if opts.SharedWith == uuid.Nil {
		sharedUser := User(t, db, UserOpts{})
		opts.SharedWith = sharedUser.ID
	}
	if opts.Name == "" {
		opts.Name = "shared-file-" + randID() + ".txt"
	}
	if opts.MimeType == nil {
		mt := "text/plain"
		opts.MimeType = &mt
	}
	if opts.Size == 0 {
		opts.Size = 2048
	}

	// Create owner and recipient users
	owner := &sqlc.User{ID: opts.OwnerID}
	recipient := &sqlc.User{ID: opts.SharedWith}

	q := sqlc.New(db)
	n := *now()

	// Create the file
	file, err := q.CreateFile(context.Background(), sqlc.CreateFileParams{
		OwnerID:    opts.OwnerID,
		Name:       opts.Name,
		Path:       "/uploads/" + opts.Name,
		MimeType:   opts.MimeType,
		Size:       opts.Size,
		Reference:  "https://example.com/files/" + opts.Name,
		Payload:    "{}",
		Service:    nil,
		Source:     sqlc.FileUploadSourceFile,
		IsArchived: false,
		CreatedAt:  n,
		UpdatedAt:  n,
	})
	require.NoError(t, err, "factory: failed to create test file for sharing")

	if opts.FileID != uuid.Nil {
		file.ID = opts.FileID
	}

	return &file, owner, recipient
}

package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// CommentOpts controls which fields are customised when creating a test comment.
// Any zero-value field gets a sensible default.
type CommentOpts struct {
	UserID        uuid.UUID
	ParentID      *uuid.UUID
	Content       []byte
	EntityType    sqlc.EntityType
	EntityID      uuid.UUID
	IsDeleted     *bool
	IsResolved    *bool
	IsArchived    *bool
	IsPinned      *bool
	Depth         *int32
	Score         *float32
	ReplyCount    *int32
	ReactionCount *int32
	Metadata      []byte
	Source        *string
}

// Comment inserts a comment into the DB and returns the created model.
// If UserID is zero, a new test user is created automatically.
func Comment(t *testing.T, db sqlc.DBTX, opts CommentOpts) *sqlc.Comment {
	t.Helper()

	if opts.UserID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.UserID = user.ID
	}
	if opts.Content == nil {
		opts.Content = []byte("Test comment content")
	}
	if opts.EntityType == "" {
		opts.EntityType = sqlc.EntityTypeStory
	}
	if opts.EntityID == uuid.Nil {
		story := Story(t, db, StoryOpts{OwnerID: opts.UserID})
		opts.EntityID = story.ID
	}
	if opts.IsDeleted == nil {
		f := false
		opts.IsDeleted = &f
	}
	if opts.IsResolved == nil {
		f := false
		opts.IsResolved = &f
	}
	if opts.IsArchived == nil {
		f := false
		opts.IsArchived = &f
	}
	if opts.IsPinned == nil {
		f := false
		opts.IsPinned = &f
	}
	if opts.Depth == nil {
		d := int32(0)
		opts.Depth = &d
	}
	if opts.Score == nil {
		s := float32(0)
		opts.Score = &s
	}
	if opts.ReplyCount == nil {
		r := int32(0)
		opts.ReplyCount = &r
	}
	if opts.ReactionCount == nil {
		r := int32(0)
		opts.ReactionCount = &r
	}

	q := sqlc.New(db)

	comment, err := q.CreateComment(context.Background(), sqlc.CreateCommentParams{
		UserID:        opts.UserID,
		ParentID:      opts.ParentID,
		Content:       opts.Content,
		Depth:         opts.Depth,
		Score:         opts.Score,
		IsDeleted:     opts.IsDeleted,
		IsResolved:    opts.IsResolved,
		IsArchived:    opts.IsArchived,
		IsReported:    nil,
		ReportedAt:    nil,
		ReportedBy:    nil,
		IsPinned:      opts.IsPinned,
		PinnedAt:      nil,
		PinnedBy:      nil,
		EntityType:    opts.EntityType,
		EntityID:      opts.EntityID,
		Source:        opts.Source,
		ReplyCount:    opts.ReplyCount,
		ReactionCount: opts.ReactionCount,
		Metadata:      opts.Metadata,
		DeletedBy:     nil,
	})
	require.NoError(t, err, "factory: failed to create test comment")

	return &comment
}

// CommentReply creates a reply to an existing comment.
func CommentReply(t *testing.T, db sqlc.DBTX, parentComment *sqlc.Comment, opts CommentOpts) *sqlc.Comment {
	t.Helper()

	opts.ParentID = &parentComment.ID
	depth := *parentComment.Depth + 1
	opts.Depth = &depth
	opts.EntityType = parentComment.EntityType
	opts.EntityID = parentComment.EntityID

	if opts.UserID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.UserID = user.ID
	}

	return Comment(t, db, opts)
}

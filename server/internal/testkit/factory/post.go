package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// StoryPostOpts controls which fields are customised when creating a test story post.
// Any zero-value field gets a sensible default.
type StoryPostOpts struct {
	AuthorID          uuid.UUID
	Content           string
	MediaIds          []uuid.UUID
	StoryID           *uuid.UUID
	ChapterID         *uuid.UUID
	IsNotifyFollowers *bool
}

// StoryPost inserts a story post into the DB and returns the created model.
// If AuthorID is zero, a new test author is created automatically.
func StoryPost(t *testing.T, db sqlc.DBTX, opts StoryPostOpts) *sqlc.StoryPost {
	t.Helper()

	if opts.AuthorID == uuid.Nil {
		author := Author(t, db, AuthorOpts{})
		opts.AuthorID = author.ID
	}
	if opts.Content == "" {
		opts.Content = "Test story post content " + randID()
	}
	if opts.MediaIds == nil {
		opts.MediaIds = []uuid.UUID{}
	}
	if opts.IsNotifyFollowers == nil {
		f := false
		opts.IsNotifyFollowers = &f
	}

	q := sqlc.New(db)
	n := now()

	post, err := q.CreateStoryPost(context.Background(), sqlc.CreateStoryPostParams{
		AuthorID:          opts.AuthorID,
		Content:           opts.Content,
		MediaIds:          opts.MediaIds,
		StoryID:           opts.StoryID,
		ChapterID:         opts.ChapterID,
		IsNotifyFollowers: opts.IsNotifyFollowers,
	})
	require.NoError(t, err, "factory: failed to create test story post")

	// Override timestamps if needed (they're set by DB triggers)
	post.CreatedAt = n
	post.UpdatedAt = n

	return &post
}

// StoryPostWithStory creates a story post associated with a specific story.
func StoryPostWithStory(t *testing.T, db sqlc.DBTX, opts StoryPostOpts) *sqlc.StoryPost {
	t.Helper()

	if opts.AuthorID == uuid.Nil {
		author := Author(t, db, AuthorOpts{})
		opts.AuthorID = author.ID
	}
	if opts.StoryID == nil {
		story := Story(t, db, StoryOpts{OwnerID: opts.AuthorID})
		opts.StoryID = &story.ID
	}

	return StoryPost(t, db, opts)
}

// StoryPostWithChapter creates a story post associated with a specific chapter.
func StoryPostWithChapter(t *testing.T, db sqlc.DBTX, opts StoryPostOpts) *sqlc.StoryPost {
	t.Helper()

	if opts.AuthorID == uuid.Nil {
		author := Author(t, db, AuthorOpts{})
		opts.AuthorID = author.ID
	}
	if opts.StoryID == nil {
		story := Story(t, db, StoryOpts{OwnerID: opts.AuthorID})
		opts.StoryID = &story.ID
	}
	if opts.ChapterID == nil {
		// For simplicity, we just set the story ID
		// In real tests, you may want to create an actual chapter
		opts.ChapterID = opts.StoryID
	}

	return StoryPost(t, db, opts)
}

package factory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// ChapterOpts controls which fields are customised when creating a test chapter.
type ChapterOpts struct {
	StoryID     uuid.UUID
	Title       string
	Number      *int32
	SortOrder   *int32
	Summary     *string
	IsPublished *bool
	TotalWords  *int32
}

// Chapter creates and inserts a chapter into the DB and returns the created model.
func Chapter(t *testing.T, db sqlc.DBTX, opts ChapterOpts) *sqlc.Chapter {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
	}
	if opts.Title == "" {
		opts.Title = "test-chapter-" + randID()
	}
	if opts.Number == nil {
		opts.Number = Int32Ptr(1)
	}
	if opts.SortOrder == nil {
		opts.SortOrder = Int32Ptr(1)
	}
	if opts.IsPublished == nil {
		opts.IsPublished = BoolPtr(false)
	}
	if opts.TotalWords == nil {
		opts.TotalWords = Int32Ptr(0)
	}

	q := sqlc.New(db)
	n := now()

	chapter, err := q.CreateChapter(context.Background(), sqlc.CreateChapterParams{
		StoryID:     opts.StoryID,
		Title:       opts.Title,
		Number:      opts.Number,
		SortOrder:   opts.SortOrder,
		Summary:     opts.Summary,
		IsPublished: opts.IsPublished,
		PublishedAt: nil,
		TotalWords:  opts.TotalWords,
		TotalViews:  Int32Ptr(0),
		CreatedAt:   n,
		UpdatedAt:   n,
	})
	require.NoError(t, err, "factory: failed to create test chapter")

	return &chapter
}

// ChapterWithStory creates a chapter with a new story and returns both.
func ChapterWithStory(t *testing.T, db sqlc.DBTX, opts ChapterOpts) (*sqlc.Chapter, *sqlc.Story) {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
		return Chapter(t, db, opts), story
	}

	return Chapter(t, db, opts), nil
}

// PublishedChapter creates a published chapter.
func PublishedChapter(t *testing.T, db sqlc.DBTX, opts ChapterOpts) *sqlc.Chapter {
	t.Helper()

	opts.IsPublished = BoolPtr(true)
	publishedAt := time.Now().Truncate(time.Second).UTC()
	opts.Summary = StringPtr("Published chapter summary")

	q := sqlc.New(db)
	n := now()

	if opts.StoryID == uuid.Nil {
		user := User(t, db, UserOpts{})
		author := Author(t, db, AuthorOpts{UserID: user.ID})
		story := Story(t, db, StoryOpts{OwnerID: author.UserID})
		opts.StoryID = story.ID
	}
	if opts.Title == "" {
		opts.Title = "test-chapter-" + randID()
	}
	if opts.Number == nil {
		opts.Number = Int32Ptr(1)
	}
	if opts.SortOrder == nil {
		opts.SortOrder = Int32Ptr(1)
	}
	if opts.TotalWords == nil {
		opts.TotalWords = Int32Ptr(1000)
	}

	chapter, err := q.CreateChapter(context.Background(), sqlc.CreateChapterParams{
		StoryID:     opts.StoryID,
		Title:       opts.Title,
		Number:      opts.Number,
		SortOrder:   opts.SortOrder,
		Summary:     opts.Summary,
		IsPublished: opts.IsPublished,
		PublishedAt: &publishedAt,
		TotalWords:  opts.TotalWords,
		TotalViews:  Int32Ptr(0),
		CreatedAt:   n,
		UpdatedAt:   n,
	})
	require.NoError(t, err, "factory: failed to create published test chapter")

	return &chapter
}

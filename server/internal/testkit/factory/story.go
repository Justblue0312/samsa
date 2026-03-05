package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// StoryOpts controls which fields are customised when creating a test story.
type StoryOpts struct {
	OwnerID       uuid.UUID
	MediaID       uuid.UUID
	Name          string
	Slug          string
	Synopsis      *string
	Status        sqlc.StoryStatus
	IsVerified    *bool
	IsRecommended *bool
}

// Story creates and inserts a story into the DB and returns the created model.
func Story(t *testing.T, db sqlc.DBTX, opts StoryOpts) *sqlc.Story {
	t.Helper()

	if opts.OwnerID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.OwnerID = user.ID
	}
	if opts.MediaID == uuid.Nil {
		opts.MediaID = uuid.New()
	}
	if opts.Name == "" {
		opts.Name = "test-story-" + randID()
	}
	if opts.Slug == "" {
		opts.Slug = "test-story-" + randID()
	}
	if opts.Status == "" {
		opts.Status = sqlc.StoryStatusDraft
	}

	q := sqlc.New(db)
	n := now()

	story, err := q.CreateStory(context.Background(), sqlc.CreateStoryParams{
		OwnerID:          opts.OwnerID,
		MediaID:          opts.MediaID,
		Name:             opts.Name,
		Slug:             opts.Slug,
		Synopsis:         opts.Synopsis,
		IsVerified:       opts.IsVerified,
		IsRecommended:    opts.IsRecommended,
		Status:           opts.Status,
		FirstPublishedAt: nil,
		LastPublishedAt:  nil,
		Settings:         nil,
		CreatedAt:        n,
		UpdatedAt:        n,
		DeletedAt:        nil,
	})
	require.NoError(t, err, "factory: failed to create test story")

	return &story
}

// StoryVoteOpts controls which fields are customised when creating a test story vote.
type StoryVoteOpts struct {
	StoryID uuid.UUID
	UserID  uuid.UUID
	Rating  int32
}

// StoryVote creates and inserts a story vote into the DB and returns the created model.
func StoryVote(t *testing.T, db sqlc.DBTX, opts StoryVoteOpts) *sqlc.StoryVote {
	t.Helper()

	if opts.StoryID == uuid.Nil {
		story := Story(t, db, StoryOpts{})
		opts.StoryID = story.ID
	}
	if opts.UserID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.UserID = user.ID
	}
	if opts.Rating == 0 {
		opts.Rating = 5
	}

	q := sqlc.New(db)

	vote, err := q.UpsertStoryVote(context.Background(), sqlc.UpsertStoryVoteParams{
		StoryID: opts.StoryID,
		UserID:  opts.UserID,
		Rating:  opts.Rating,
	})
	require.NoError(t, err, "factory: failed to create test story vote")

	return &vote
}

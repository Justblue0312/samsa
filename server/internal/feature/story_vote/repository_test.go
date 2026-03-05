package story_vote_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/story_vote"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoryVoteRepository_UpsertVote(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("creates vote successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		arg := sqlc.UpsertStoryVoteParams{
			StoryID: story.ID,
			UserID:  user.ID,
			Rating:  5,
		}

		result, err := repo.UpsertVote(ctx, arg)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, story.ID, result.StoryID)
		assert.Equal(t, user.ID, result.UserID)
		assert.Equal(t, int32(5), result.Rating)
	})

	t.Run("updates existing vote", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		// Create initial vote
		arg := sqlc.UpsertStoryVoteParams{
			StoryID: story.ID,
			UserID:  user.ID,
			Rating:  3,
		}

		first, err := repo.UpsertVote(ctx, arg)
		require.NoError(t, err)

		// Update vote
		arg.Rating = 5
		second, err := repo.UpsertVote(ctx, arg)
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, int32(5), second.Rating)
	})
}

func TestStoryVoteRepository_GetByID(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("returns vote when found", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})
		vote := factory.StoryVote(t, pool, factory.StoryVoteOpts{
			StoryID: story.ID,
			UserID:  user.ID,
			Rating:  4,
		})

		result, err := repo.GetByID(ctx, vote.ID)
		require.NoError(t, err)

		assert.Equal(t, vote.ID, result.ID)
		assert.Equal(t, vote.Rating, result.Rating)
	})

	t.Run("returns ErrNotFound when not found", func(t *testing.T) {
		result, err := repo.GetByID(ctx, uuid.New())
		assert.ErrorIs(t, err, story_vote.ErrNotFound)
		assert.Nil(t, result)
	})
}

func TestStoryVoteRepository_GetByStoryAndUser(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("returns vote when found", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})
		vote := factory.StoryVote(t, pool, factory.StoryVoteOpts{
			StoryID: story.ID,
			UserID:  user.ID,
			Rating:  3,
		})

		result, err := repo.GetByStoryAndUser(ctx, story.ID, user.ID)
		require.NoError(t, err)

		assert.Equal(t, vote.ID, result.ID)
		assert.Equal(t, vote.Rating, result.Rating)
	})

	t.Run("returns ErrNotFound when not found", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		result, err := repo.GetByStoryAndUser(ctx, story.ID, user.ID)
		assert.ErrorIs(t, err, story_vote.ErrNotFound)
		assert.Nil(t, result)
	})
}

func TestStoryVoteRepository_DeleteByStoryAndUser(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("deletes vote successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})
		_ = factory.StoryVote(t, pool, factory.StoryVoteOpts{
			StoryID: story.ID,
			UserID:  user.ID,
			Rating:  5,
		})

		err := repo.DeleteByStoryAndUser(ctx, story.ID, user.ID)
		require.NoError(t, err)

		// Verify vote is deleted
		result, err := repo.GetByStoryAndUser(ctx, story.ID, user.ID)
		assert.ErrorIs(t, err, story_vote.ErrNotFound)
		assert.Nil(t, result)
	})

	t.Run("deletes non-existent vote without error", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		err := repo.DeleteByStoryAndUser(ctx, story.ID, user.ID)
		// Should not error even if vote doesn't exist
		require.NoError(t, err)
	})
}

func TestStoryVoteRepository_GetStats(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("returns correct vote statistics", func(t *testing.T) {
		user1 := factory.User(t, pool, factory.UserOpts{})
		user2 := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user1.ID})

		_ = factory.StoryVote(t, pool, factory.StoryVoteOpts{
			StoryID: story.ID,
			UserID:  user1.ID,
			Rating:  5,
		})
		_ = factory.StoryVote(t, pool, factory.StoryVoteOpts{
			StoryID: story.ID,
			UserID:  user2.ID,
			Rating:  3,
		})

		stats, err := repo.GetStats(ctx, story.ID)
		require.NoError(t, err)

		assert.Equal(t, int64(2), stats.TotalVotes)
		assert.InDelta(t, float32(4.0), stats.AverageRating, 0.01)
	})

	t.Run("returns zero stats for story with no votes", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		stats, err := repo.GetStats(ctx, story.ID)
		require.NoError(t, err)

		assert.Equal(t, int64(0), stats.TotalVotes)
		assert.Equal(t, float32(0), stats.AverageRating)
	})
}

func TestStoryVoteRepository_ListByStory(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("lists votes for a story with pagination", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		// Create 5 votes
		for i := int32(1); i <= 5; i++ {
			u := factory.User(t, pool, factory.UserOpts{})
			_ = factory.StoryVote(t, pool, factory.StoryVoteOpts{
				StoryID: story.ID,
				UserID:  u.ID,
				Rating:  i,
			})
		}

		votes, total, err := repo.ListByStory(ctx, story.ID, 3, 0)
		require.NoError(t, err)

		assert.Len(t, votes, 3)
		assert.Equal(t, int64(5), total)
	})

	t.Run("returns empty list for story with no votes", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: user.ID})

		votes, total, err := repo.ListByStory(ctx, story.ID, 10, 0)
		require.NoError(t, err)

		assert.Empty(t, votes)
		assert.Equal(t, int64(0), total)
	})
}

func TestStoryVoteRepository_ListByUser(t *testing.T) {
	pool := testkit.NewDB(t)
	cfg := testkit.SetupConfig()
	repo := story_vote.NewRepository(pool, cfg, nil)

	ctx := context.Background()

	t.Run("lists votes by a user with pagination", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		
		// Create 5 stories and votes
		for i := int32(1); i <= 5; i++ {
			story := factory.Story(t, pool, factory.StoryOpts{})
			_ = factory.StoryVote(t, pool, factory.StoryVoteOpts{
				StoryID: story.ID,
				UserID:  user.ID,
				Rating:  i,
			})
		}

		votes, total, err := repo.ListByUser(ctx, user.ID, 3, 0)
		require.NoError(t, err)

		assert.Len(t, votes, 3)
		assert.Equal(t, int64(5), total)
	})

	t.Run("returns empty list for user with no votes", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})

		votes, total, err := repo.ListByUser(ctx, user.ID, 10, 0)
		require.NoError(t, err)

		assert.Empty(t, votes)
		assert.Equal(t, int64(0), total)
	})
}

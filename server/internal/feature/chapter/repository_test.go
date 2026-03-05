package chapter_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/chapter"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChapterRepository_Create(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("creates chapter successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		arg := sqlc.CreateChapterParams{
			StoryID:     story.ID,
			Title:       "Test Chapter",
			Number:      factory.Int32Ptr(1),
			SortOrder:   factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
			TotalWords:  factory.Int32Ptr(0),
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, story.ID, result.StoryID)
		assert.Equal(t, "Test Chapter", result.Title)
		assert.Equal(t, int32(1), *result.Number)
		assert.False(t, *result.IsPublished)
	})

	t.Run("creates published chapter successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		arg := sqlc.CreateChapterParams{
			StoryID:     story.ID,
			Title:       "Published Chapter",
			Number:      factory.Int32Ptr(2),
			SortOrder:   factory.Int32Ptr(2),
			IsPublished: factory.BoolPtr(true),
			TotalWords:  factory.Int32Ptr(1000),
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, *result.IsPublished)
		assert.NotNil(t, result.PublishedAt)
	})
}

func TestChapterRepository_GetByID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns chapter when found", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		result, err := repo.GetByID(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.Title, result.Title)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		result, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestChapterRepository_GetByStoryAndNumber(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns chapter when found", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{
			Number: factory.Int32Ptr(5),
		})

		result, err := repo.GetByStoryAndNumber(ctx, existing.StoryID, 5)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, int32(5), *result.Number)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		story := factory.Story(t, pool, factory.StoryOpts{})
		result, err := repo.GetByStoryAndNumber(ctx, story.ID, 999)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestChapterRepository_GetChaptersByStoryID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns all chapters for story", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		// Create multiple chapters
		chapter1 := factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID: story.ID,
			Number:  factory.Int32Ptr(1),
		})
		chapter2 := factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID: story.ID,
			Number:  factory.Int32Ptr(2),
		})
		chapter3 := factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID: story.ID,
			Number:  factory.Int32Ptr(3),
		})

		results, err := repo.GetChaptersByStoryID(ctx, story.ID)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		// Verify all chapters are present
		ids := make(map[uuid.UUID]bool)
		for _, c := range results {
			ids[c.ID] = true
		}
		assert.True(t, ids[chapter1.ID])
		assert.True(t, ids[chapter2.ID])
		assert.True(t, ids[chapter3.ID])
	})

	t.Run("returns empty slice for story with no chapters", func(t *testing.T) {
		story := factory.Story(t, pool, factory.StoryOpts{})

		results, err := repo.GetChaptersByStoryID(ctx, story.ID)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestChapterRepository_GetPublishedChaptersByStoryID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns only published chapters", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		// Create mix of published and draft chapters
		draft := factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID:     story.ID,
			Number:      factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
		})
		published := factory.PublishedChapter(t, pool, factory.ChapterOpts{
			StoryID: story.ID,
			Number:  factory.Int32Ptr(2),
		})

		results, err := repo.GetPublishedChaptersByStoryID(ctx, story.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify only published chapters are returned
		for _, c := range results {
			if c.ID == draft.ID {
				assert.False(t, *c.IsPublished)
			} else if c.ID == published.ID {
				assert.True(t, *c.IsPublished)
			}
		}
	})
}

func TestChapterRepository_Update(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("updates chapter successfully", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		arg := sqlc.UpdateChapterParams{
			ID:          existing.ID,
			Title:       "Updated Title",
			Number:      factory.Int32Ptr(10),
			SortOrder:   factory.Int32Ptr(5),
			Summary:     factory.StringPtr("Updated summary"),
			IsPublished: factory.BoolPtr(true),
			TotalWords:  factory.Int32Ptr(2000),
			TotalViews:  existing.TotalViews,
		}

		result, err := repo.Update(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Title", result.Title)
		assert.Equal(t, int32(10), *result.Number)
		assert.Equal(t, "Updated summary", *result.Summary)
		assert.True(t, *result.IsPublished)
		assert.Equal(t, int32(2000), *result.TotalWords)
	})

	t.Run("returns error when chapter not found", func(t *testing.T) {
		arg := sqlc.UpdateChapterParams{
			ID:          uuid.New(),
			Title:       "Non-existent",
			Number:      factory.Int32Ptr(1),
			SortOrder:   factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
			TotalWords:  factory.Int32Ptr(0),
		}

		result, err := repo.Update(ctx, arg)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestChapterRepository_Delete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("deletes chapter successfully", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		err := repo.Delete(ctx, existing.ID)
		require.NoError(t, err)

		// Verify chapter is deleted
		result, err := repo.GetByID(ctx, existing.ID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when chapter not found", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestChapterRepository_SoftDelete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("soft deletes chapter successfully", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		result, err := repo.SoftDelete(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DeletedAt)

		// Verify chapter is marked as deleted
		assert.NotNil(t, result.DeletedAt)
	})
}

func TestChapterRepository_Publish(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("publishes chapter successfully", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{
			IsPublished: factory.BoolPtr(false),
		})

		result, err := repo.Publish(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, *result.IsPublished)
		assert.NotNil(t, result.PublishedAt)
	})
}

func TestChapterRepository_Unpublish(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("unpublishes chapter successfully", func(t *testing.T) {
		existing := factory.PublishedChapter(t, pool, factory.ChapterOpts{})

		result, err := repo.Unpublish(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, *result.IsPublished)
		assert.Nil(t, result.PublishedAt)
	})
}

func TestChapterRepository_IncrementViews(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("increments view count", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		result, err := repo.IncrementViews(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, *result.TotalViews, int32(0))
	})
}

func TestChapterRepository_UpdateStats(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("updates chapter stats", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{})

		result, err := repo.UpdateStats(ctx, existing.ID, 1000, 10, 5, 3, 1, 2)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1000), *result.TotalWords)
		assert.Equal(t, int32(10), *result.TotalVotes)
		assert.Equal(t, int32(5), *result.TotalFavorites)
		assert.Equal(t, int32(3), *result.TotalBookmarks)
		assert.Equal(t, int32(1), *result.TotalFlags)
		assert.Equal(t, int32(2), *result.TotalReports)
	})
}

func TestChapterRepository_GetNextSortOrder(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns 1 for story with no chapters", func(t *testing.T) {
		story := factory.Story(t, pool, factory.StoryOpts{})

		result, err := repo.GetNextSortOrder(ctx, story.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(1), result)
	})

	t.Run("returns next order for story with chapters", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		// Create chapters with specific sort orders
		factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID:   story.ID,
			SortOrder: factory.Int32Ptr(1),
		})
		factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID:   story.ID,
			SortOrder: factory.Int32Ptr(2),
		})
		factory.Chapter(t, pool, factory.ChapterOpts{
			StoryID:   story.ID,
			SortOrder: factory.Int32Ptr(3),
		})

		result, err := repo.GetNextSortOrder(ctx, story.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(4), result)
	})
}

func TestChapterRepository_Reorder(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := chapter.NewRepository(pool)

	ctx := context.Background()

	t.Run("reorders chapter successfully", func(t *testing.T) {
		existing := factory.Chapter(t, pool, factory.ChapterOpts{
			SortOrder: factory.Int32Ptr(1),
		})

		err := repo.Reorder(ctx, existing.ID, existing.StoryID, 5)
		require.NoError(t, err)

		// Verify reorder
		updated, err := repo.GetByID(ctx, existing.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(5), *updated.SortOrder)
	})
}

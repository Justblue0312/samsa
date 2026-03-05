package chapter_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/chapter"
	chapterMocks "github.com/justblue/samsa/internal/feature/chapter/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChapterUseCase_CreateChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("creates chapter successfully", func(t *testing.T) {
		storyID := uuid.New()
		req := chapter.CreateChapterRequest{
			StoryID: storyID,
			Title:   "Test Chapter",
			Number:  factory.Int32Ptr(1),
		}

		expectedChapter := &sqlc.Chapter{
			ID:          uuid.New(),
			StoryID:     storyID,
			Title:       "Test Chapter",
			Number:      factory.Int32Ptr(1),
			SortOrder:   factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
			TotalWords:  factory.Int32Ptr(0),
		}

		mockRepo.EXPECT().
			GetNextSortOrder(ctx, storyID).
			Return(int32(1), nil)

		mockRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedChapter, nil)

		result, err := uc.CreateChapter(ctx, storyID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Chapter", result.Title)
	})

	t.Run("creates published chapter", func(t *testing.T) {
		storyID := uuid.New()
		req := chapter.CreateChapterRequest{
			StoryID:     storyID,
			Title:       "Published Chapter",
			IsPublished: factory.BoolPtr(true),
		}

		expectedChapter := &sqlc.Chapter{
			ID:          uuid.New(),
			StoryID:     storyID,
			Title:       "Published Chapter",
			IsPublished: factory.BoolPtr(true),
		}

		mockRepo.EXPECT().
			GetNextSortOrder(ctx, storyID).
			Return(int32(1), nil)

		mockRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedChapter, nil)

		result, err := uc.CreateChapter(ctx, storyID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsPublished)
	})
}

func TestChapterUseCase_GetChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("returns chapter when found", func(t *testing.T) {
		chapterID := uuid.New()
		expectedChapter := &sqlc.Chapter{
			ID:        chapterID,
			Title:     "Test Chapter",
			StoryID:   uuid.New(),
			Number:    factory.Int32Ptr(1),
			SortOrder: factory.Int32Ptr(1),
		}

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(expectedChapter, nil)

		result, err := uc.GetChapter(ctx, chapterID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Chapter", result.Title)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		chapterID := uuid.New()

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(nil, assert.AnError)

		result, err := uc.GetChapter(ctx, chapterID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestChapterUseCase_ListChapters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("lists all chapters for story", func(t *testing.T) {
		storyID := uuid.New()
		params := chapter.ListChaptersParams{
			StoryID: storyID,
		}

		chapters := []sqlc.Chapter{
			{ID: uuid.New(), StoryID: storyID, Title: "Chapter 1", Number: factory.Int32Ptr(1)},
			{ID: uuid.New(), StoryID: storyID, Title: "Chapter 2", Number: factory.Int32Ptr(2)},
		}

		mockRepo.EXPECT().
			GetChaptersByStoryID(ctx, storyID).
			Return(chapters, nil)

		result, err := uc.ListChapters(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("lists published chapters only", func(t *testing.T) {
		storyID := uuid.New()
		params := chapter.ListChaptersParams{
			StoryID:     storyID,
			IsPublished: factory.BoolPtr(true),
		}

		chapters := []sqlc.Chapter{
			{ID: uuid.New(), StoryID: storyID, Title: "Published Chapter", IsPublished: factory.BoolPtr(true)},
		}

		mockRepo.EXPECT().
			GetPublishedChaptersByStoryID(ctx, storyID).
			Return(chapters, nil)

		result, err := uc.ListChapters(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestChapterUseCase_UpdateChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("updates chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Chapter{
			ID:          chapterID,
			Title:       "Old Title",
			Number:      factory.Int32Ptr(1),
			SortOrder:   factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
		}

		req := chapter.UpdateChapterRequest{
			Title: factory.StringPtr("New Title"),
		}

		updated := &sqlc.Chapter{
			ID:          chapterID,
			Title:       "New Title",
			Number:      factory.Int32Ptr(1),
			SortOrder:   factory.Int32Ptr(1),
			IsPublished: factory.BoolPtr(false),
		}

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(existing, nil)

		mockRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(updated, nil)

		result, err := uc.UpdateChapter(ctx, userID, chapterID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Title", result.Title)
	})
}

func TestChapterUseCase_DeleteChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("deletes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Chapter{
			ID:      chapterID,
			StoryID: uuid.New(),
		}

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(existing, nil)

		mockRepo.EXPECT().
			SoftDelete(ctx, chapterID).
			Return(existing, nil)

		err := uc.DeleteChapter(ctx, userID, chapterID)
		require.NoError(t, err)
	})
}

func TestChapterUseCase_PublishChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("publishes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		userID := uuid.New()

		published := &sqlc.Chapter{
			ID:          chapterID,
			IsPublished: factory.BoolPtr(true),
			PublishedAt: factory.TimeAt(2026, 3, 4, 0, 0, 0),
		}

		mockRepo.EXPECT().
			Publish(ctx, chapterID).
			Return(published, nil)

		result, err := uc.PublishChapter(ctx, userID, chapterID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.IsPublished)
	})
}

func TestChapterUseCase_UnpublishChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("unpublishes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		userID := uuid.New()

		unpublished := &sqlc.Chapter{
			ID:          chapterID,
			IsPublished: factory.BoolPtr(false),
		}

		mockRepo.EXPECT().
			Unpublish(ctx, chapterID).
			Return(unpublished, nil)

		result, err := uc.UnpublishChapter(ctx, userID, chapterID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsPublished)
	})
}

func TestChapterUseCase_ReorderChapter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("reorders chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		storyID := uuid.New()
		existing := &sqlc.Chapter{
			ID:        chapterID,
			StoryID:   storyID,
			SortOrder: factory.Int32Ptr(1),
		}

		req := chapter.ReorderChapterRequest{
			StoryID:   storyID,
			SortOrder: 5,
		}

		updated := &sqlc.Chapter{
			ID:        chapterID,
			StoryID:   storyID,
			SortOrder: factory.Int32Ptr(5),
		}

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(existing, nil)

		mockRepo.EXPECT().
			Reorder(ctx, chapterID, storyID, int32(5)).
			Return(nil)

		mockRepo.EXPECT().
			GetByID(ctx, chapterID).
			Return(updated, nil)

		result, err := uc.ReorderChapter(ctx, chapterID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(5), *result.SortOrder)
	})
}

func TestChapterUseCase_IncrementView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := chapterMocks.NewMockRepository(ctrl)
	uc := chapter.NewUseCase(mockRepo)

	t.Run("increments view count", func(t *testing.T) {
		chapterID := uuid.New()

		incremented := &sqlc.Chapter{
			ID:         chapterID,
			TotalViews: factory.Int32Ptr(1),
		}

		mockRepo.EXPECT().
			IncrementViews(ctx, chapterID).
			Return(incremented, nil)

		result, err := uc.IncrementView(ctx, chapterID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.TotalViews)
	})
}

package chapter

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Basic CRUD
	Create(ctx context.Context, arg sqlc.CreateChapterParams) (*sqlc.Chapter, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error)
	GetByStoryAndNumber(ctx context.Context, storyID uuid.UUID, number int32) (*sqlc.Chapter, error)
	GetChaptersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Chapter, error)
	GetPublishedChaptersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Chapter, error)
	Update(ctx context.Context, arg sqlc.UpdateChapterParams) (*sqlc.Chapter, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error)

	// Publishing
	Publish(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error)
	Unpublish(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error)

	// Stats
	IncrementViews(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error)
	UpdateStats(ctx context.Context, id uuid.UUID, words, votes, favorites, bookmarks, flags, reports int32) (*sqlc.Chapter, error)

	// Ordering
	GetNextSortOrder(ctx context.Context, storyID uuid.UUID) (int32, error)
	Reorder(ctx context.Context, id uuid.UUID, storyID uuid.UUID, sortOrder int32) error
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateChapterParams) (*sqlc.Chapter, error) {
	c, err := r.q.CreateChapter(ctx, arg)
	return &c, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error) {
	c, err := r.q.GetChapterByID(ctx, id)
	return &c, err
}

func (r *repository) GetByStoryAndNumber(ctx context.Context, storyID uuid.UUID, number int32) (*sqlc.Chapter, error) {
	c, err := r.q.GetChapterByStoryAndNumber(ctx, sqlc.GetChapterByStoryAndNumberParams{
		StoryID: storyID,
		Number:  &number,
	})
	return &c, err
}

func (r *repository) GetChaptersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Chapter, error) {
	return r.q.GetChaptersByStoryID(ctx, storyID)
}

func (r *repository) GetPublishedChaptersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Chapter, error) {
	return r.q.GetPublishedChaptersByStoryID(ctx, storyID)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateChapterParams) (*sqlc.Chapter, error) {
	c, err := r.q.UpdateChapter(ctx, arg)
	return &c, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteChapter(ctx, id)
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error) {
	c, err := r.q.SoftDeleteChapter(ctx, id)
	return &c, err
}

func (r *repository) Publish(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error) {
	c, err := r.q.PublishChapter(ctx, id)
	return &c, err
}

func (r *repository) Unpublish(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error) {
	c, err := r.q.UnpublishChapter(ctx, id)
	return &c, err
}

func (r *repository) IncrementViews(ctx context.Context, id uuid.UUID) (*sqlc.Chapter, error) {
	c, err := r.q.IncrementChapterViews(ctx, id)
	return &c, err
}

func (r *repository) UpdateStats(ctx context.Context, id uuid.UUID, words, votes, favorites, bookmarks, flags, reports int32) (*sqlc.Chapter, error) {
	c, err := r.q.UpdateChapterStats(ctx, sqlc.UpdateChapterStatsParams{
		ID:             id,
		TotalWords:     &words,
		TotalVotes:     &votes,
		TotalFavorites: &favorites,
		TotalBookmarks: &bookmarks,
		TotalFlags:     &flags,
		TotalReports:   &reports,
	})
	return &c, err
}

func (r *repository) GetNextSortOrder(ctx context.Context, storyID uuid.UUID) (int32, error) {
	return r.q.GetNextChapterSortOrder(ctx, storyID)
}

func (r *repository) Reorder(ctx context.Context, id uuid.UUID, storyID uuid.UUID, sortOrder int32) error {
	return r.q.ReorderChapters(ctx, sqlc.ReorderChaptersParams{
		ID:        id,
		StoryID:   storyID,
		SortOrder: &sortOrder,
	})
}

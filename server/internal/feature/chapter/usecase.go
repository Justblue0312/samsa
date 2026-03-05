package chapter

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Basic CRUD
	CreateChapter(ctx context.Context, storyID uuid.UUID, req CreateChapterRequest) (*ChapterResponse, error)
	GetChapter(ctx context.Context, id uuid.UUID) (*ChapterResponse, error)
	ListChapters(ctx context.Context, params ListChaptersParams) ([]ChapterResponse, error)
	UpdateChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID, req UpdateChapterRequest) (*ChapterResponse, error)
	DeleteChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) error

	// Publishing
	PublishChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) (*ChapterResponse, error)
	UnpublishChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) (*ChapterResponse, error)

	// Reordering
	ReorderChapter(ctx context.Context, chapterID uuid.UUID, req ReorderChapterRequest) (*ChapterResponse, error)

	// Stats
	IncrementView(ctx context.Context, chapterID uuid.UUID) (*ChapterResponse, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateChapter(ctx context.Context, storyID uuid.UUID, req CreateChapterRequest) (*ChapterResponse, error) {
	// Get next sort order if not provided
	sortOrder := req.SortOrder
	if sortOrder == nil {
		nextOrder, err := uc.repo.GetNextSortOrder(ctx, storyID)
		if err != nil {
			return nil, err
		}
		sortOrder = &nextOrder
	}

	// Set published status
	isPublished := false
	var publishedAt *time.Time
	if req.IsPublished != nil && *req.IsPublished {
		isPublished = true
		now := time.Now()
		publishedAt = &now
	}

	arg := sqlc.CreateChapterParams{
		StoryID:     storyID,
		Title:       req.Title,
		Number:      req.Number,
		SortOrder:   sortOrder,
		Summary:     req.Summary,
		IsPublished: &isPublished,
		PublishedAt: publishedAt,
		TotalWords:  req.TotalWords,
		TotalViews:  ptr(int32(0)),
		CreatedAt:   ptr(time.Now()),
		UpdatedAt:   ptr(time.Now()),
	}

	chapter, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(chapter), nil
}

func (uc *usecase) GetChapter(ctx context.Context, id uuid.UUID) (*ChapterResponse, error) {
	chapter, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToChapterResponse(chapter), nil
}

func (uc *usecase) ListChapters(ctx context.Context, params ListChaptersParams) ([]ChapterResponse, error) {
	var chapters []sqlc.Chapter
	var err error

	if params.IsPublished != nil {
		if *params.IsPublished {
			chapters, err = uc.repo.GetPublishedChaptersByStoryID(ctx, params.StoryID)
		} else {
			chapters, err = uc.repo.GetChaptersByStoryID(ctx, params.StoryID)
		}
	} else {
		chapters, err = uc.repo.GetChaptersByStoryID(ctx, params.StoryID)
	}

	if err != nil {
		return nil, err
	}

	return ToChapterListResponse(chapters), nil
}

func (uc *usecase) UpdateChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID, req UpdateChapterRequest) (*ChapterResponse, error) {
	// Get existing chapter
	existing, err := uc.repo.GetByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// Build update params
	arg := sqlc.UpdateChapterParams{
		ID:          chapterID,
		Title:       existing.Title,
		Number:      existing.Number,
		SortOrder:   existing.SortOrder,
		Summary:     existing.Summary,
		IsPublished: existing.IsPublished,
		PublishedAt: existing.PublishedAt,
		TotalWords:  existing.TotalWords,
		TotalViews:  existing.TotalViews,
		UpdatedAt:   ptr(time.Now()),
	}

	// Apply updates
	if req.Title != nil {
		arg.Title = *req.Title
	}
	if req.Number != nil {
		arg.Number = req.Number
	}
	if req.SortOrder != nil {
		arg.SortOrder = req.SortOrder
	}
	if req.Summary != nil {
		arg.Summary = req.Summary
	}
	if req.IsPublished != nil {
		arg.IsPublished = req.IsPublished
		if *req.IsPublished && existing.PublishedAt == nil {
			now := time.Now()
			arg.PublishedAt = &now
		} else if !*req.IsPublished {
			arg.PublishedAt = nil
		}
	}
	if req.TotalWords != nil {
		arg.TotalWords = req.TotalWords
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(updated), nil
}

func (uc *usecase) DeleteChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) error {
	// Check ownership via story
	_, err := uc.repo.GetByID(ctx, chapterID)
	if err != nil {
		return err
	}

	// TODO: Verify user owns the story that this chapter belongs to
	_ = userID // Remove this line when ownership check is implemented

	_, err = uc.repo.SoftDelete(ctx, chapterID)
	return err
}

func (uc *usecase) PublishChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) (*ChapterResponse, error) {
	// TODO: Verify user owns the story
	_ = userID

	chapter, err := uc.repo.Publish(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(chapter), nil
}

func (uc *usecase) UnpublishChapter(ctx context.Context, userID uuid.UUID, chapterID uuid.UUID) (*ChapterResponse, error) {
	// TODO: Verify user owns the story
	_ = userID

	chapter, err := uc.repo.Unpublish(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(chapter), nil
}

func (uc *usecase) ReorderChapter(ctx context.Context, chapterID uuid.UUID, req ReorderChapterRequest) (*ChapterResponse, error) {
	// Get existing chapter to verify story ownership
	chapter, err := uc.repo.GetByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// TODO: Verify user owns the story
	_ = chapter

	err = uc.repo.Reorder(ctx, chapterID, req.StoryID, req.SortOrder)
	if err != nil {
		return nil, err
	}

	// Fetch updated chapter
	updated, err := uc.repo.GetByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(updated), nil
}

func (uc *usecase) IncrementView(ctx context.Context, chapterID uuid.UUID) (*ChapterResponse, error) {
	chapter, err := uc.repo.IncrementViews(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	return ToChapterResponse(chapter), nil
}

// Helper functions
func ptr[T any](v T) *T {
	return &v
}

// Errors
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrChapterNotFound  = errors.New("chapter not found")
)

package story_status_history

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Create creates a new status history entry
	CreateStatusHistory(ctx context.Context, storyID, userID uuid.UUID, status sqlc.StoryStatus, content, reason string) (*StatusHistoryResponse, error)

	// GetStatusHistory retrieves a status history entry by ID
	GetStatusHistory(ctx context.Context, id uuid.UUID) (*StatusHistoryResponse, error)

	// ListStoryStatusHistory retrieves all status history for a story
	ListStoryStatusHistory(ctx context.Context, storyID uuid.UUID) ([]StatusHistoryResponse, error)

	// ListStoryStatusHistoryPaginated retrieves paginated status history for a story
	ListStoryStatusHistoryPaginated(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]StatusHistoryResponse, int64, error)

	// DeleteStatusHistory deletes a status history entry
	DeleteStatusHistory(ctx context.Context, id uuid.UUID) error
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

// CreateStatusHistory creates a new status history entry
func (uc *usecase) CreateStatusHistory(ctx context.Context, storyID, userID uuid.UUID, status sqlc.StoryStatus, content, reason string) (*StatusHistoryResponse, error) {
	now := time.Now()

	history := &sqlc.StoryStatusHistory{
		ID:          uuid.New(),
		StoryID:     storyID,
		SetStatusBy: userID,
		Content:     content,
		Status:      status,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	// If reason is provided, include it in content
	if reason != "" && content == "" {
		history.Content = reason
	}

	created, err := uc.repo.Create(ctx, history)
	if err != nil {
		return nil, fmt.Errorf("usecase.CreateStatusHistory: %w", err)
	}

	return ToStatusHistoryResponse(created), nil
}

// GetStatusHistory retrieves a status history entry by ID
func (uc *usecase) GetStatusHistory(ctx context.Context, id uuid.UUID) (*StatusHistoryResponse, error) {
	history, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetStatusHistory: %w", err)
	}
	return ToStatusHistoryResponse(history), nil
}

// ListStoryStatusHistory retrieves all status history for a story
func (uc *usecase) ListStoryStatusHistory(ctx context.Context, storyID uuid.UUID) ([]StatusHistoryResponse, error) {
	history, err := uc.repo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("usecase.ListStoryStatusHistory: %w", err)
	}

	result := make([]StatusHistoryResponse, len(history))
	for i, h := range history {
		result[i] = *ToStatusHistoryResponse(&h)
	}

	return result, nil
}

// ListStoryStatusHistoryPaginated retrieves paginated status history for a story
func (uc *usecase) ListStoryStatusHistoryPaginated(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]StatusHistoryResponse, int64, error) {
	if limit <= 0 {
		limit = 20
	}

	history, total, err := uc.repo.ListByStoryPaginated(ctx, storyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListStoryStatusHistoryPaginated: %w", err)
	}

	result := make([]StatusHistoryResponse, len(history))
	for i, h := range history {
		result[i] = *ToStatusHistoryResponse(&h)
	}

	return result, total, nil
}

// DeleteStatusHistory deletes a status history entry
func (uc *usecase) DeleteStatusHistory(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("usecase.DeleteStatusHistory: %w", err)
	}
	return nil
}

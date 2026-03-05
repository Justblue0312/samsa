package story_status_history

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Create creates a new status history entry
	Create(ctx context.Context, history *sqlc.StoryStatusHistory) (*sqlc.StoryStatusHistory, error)

	// GetByID retrieves a status history entry by ID
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryStatusHistory, error)

	// ListByStory retrieves all status history entries for a story
	ListByStory(ctx context.Context, storyID uuid.UUID) ([]sqlc.StoryStatusHistory, error)

	// ListByStoryPaginated retrieves paginated status history for a story
	ListByStoryPaginated(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryStatusHistory, int64, error)

	// Delete deletes a status history entry
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByStory deletes all status history entries for a story
	DeleteByStory(ctx context.Context, storyID uuid.UUID) error
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

// Create creates a new status history entry
func (r *repository) Create(ctx context.Context, history *sqlc.StoryStatusHistory) (*sqlc.StoryStatusHistory, error) {
	result, err := r.q.CreateStoryStatusHistory(ctx, sqlc.CreateStoryStatusHistoryParams{
		StoryID:     history.StoryID,
		SetStatusBy: history.SetStatusBy,
		Content:     history.Content,
		Status:      history.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}
	return &result, nil
}

// GetByID retrieves a status history entry by ID
func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryStatusHistory, error) {
	history, err := r.q.GetStoryStatusHistoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return &history, nil
}

// ListByStory retrieves all status history entries for a story
func (r *repository) ListByStory(ctx context.Context, storyID uuid.UUID) ([]sqlc.StoryStatusHistory, error) {
	history, err := r.q.ListStoryStatusHistoryByStory(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListByStory: %w", err)
	}
	return history, nil
}

// ListByStoryPaginated retrieves paginated status history for a story
func (r *repository) ListByStoryPaginated(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.StoryStatusHistory, int64, error) {
	history, err := r.q.ListStoryStatusHistoryByStoryPaginated(ctx, sqlc.ListStoryStatusHistoryByStoryPaginatedParams{
		StoryID: storyID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStoryPaginated: %w", err)
	}

	// Get total count
	count, err := r.q.CountStoryStatusHistoryByStory(ctx, storyID)
	if err != nil {
		return nil, 0, fmt.Errorf("repository.ListByStoryPaginated count: %w", err)
	}

	return history, count, nil
}

// Delete deletes a status history entry
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteStoryStatusHistory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("repository.Delete: %w", err)
	}
	return nil
}

// DeleteByStory deletes all status history entries for a story
func (r *repository) DeleteByStory(ctx context.Context, storyID uuid.UUID) error {
	err := r.q.DeleteStoryStatusHistoryByStory(ctx, storyID)
	if err != nil {
		return fmt.Errorf("repository.DeleteByStory: %w", err)
	}
	return nil
}

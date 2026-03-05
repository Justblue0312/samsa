package story_vote

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Vote CRUD
	CreateVote(ctx context.Context, userID uuid.UUID, req CreateVoteRequest) (*VoteResponse, error)
	GetVote(ctx context.Context, id uuid.UUID) (*VoteResponse, error)
	GetUserVote(ctx context.Context, storyID, userID uuid.UUID) (*VoteResponse, error)
	DeleteUserVote(ctx context.Context, storyID, userID uuid.UUID) error

	// Vote statistics
	GetVoteStats(ctx context.Context, storyID uuid.UUID) (*VoteStatsResponse, error)

	// List operations
	ListStoryVotes(ctx context.Context, storyID uuid.UUID, filter *VoteFilter) ([]VoteResponse, int64, error)
	ListUserVotes(ctx context.Context, userID uuid.UUID, filter *VoteFilter) ([]VoteResponse, int64, error)
}

type usecase struct {
	repo     Repository
	notifier Notifier
}

func NewUseCase(repo Repository, notifier Notifier) UseCase {
	return &usecase{
		repo:     repo,
		notifier: notifier,
	}
}

// CreateVote creates or updates a vote for a story
func (uc *usecase) CreateVote(ctx context.Context, userID uuid.UUID, req CreateVoteRequest) (*VoteResponse, error) {
	arg := sqlc.UpsertStoryVoteParams{
		StoryID: req.StoryID,
		UserID:  userID,
		Rating:  req.Rating,
	}

	vote, err := uc.repo.UpsertVote(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("usecase.CreateVote: %w", err)
	}

	// Notify about vote create/update
	if uc.notifier != nil {
		_ = uc.notifier.NotifyVoteUpdate(ctx, vote.StoryID, userID, vote.Rating)
	}

	return ToVoteResponse(vote), nil
}

// GetVote retrieves a vote by ID
func (uc *usecase) GetVote(ctx context.Context, id uuid.UUID) (*VoteResponse, error) {
	vote, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("usecase.GetVote: %w", err)
	}
	return ToVoteResponse(vote), nil
}

// GetUserVote retrieves a user's vote for a story
func (uc *usecase) GetUserVote(ctx context.Context, storyID, userID uuid.UUID) (*VoteResponse, error) {
	vote, err := uc.repo.GetByStoryAndUser(ctx, storyID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil // No vote found
		}
		return nil, fmt.Errorf("usecase.GetUserVote: %w", err)
	}
	return ToVoteResponse(vote), nil
}

// DeleteUserVote deletes a user's vote for a story
func (uc *usecase) DeleteUserVote(ctx context.Context, storyID, userID uuid.UUID) error {
	if err := uc.repo.DeleteByStoryAndUser(ctx, storyID, userID); err != nil {
		return fmt.Errorf("usecase.DeleteUserVote: %w", err)
	}

	// Notify about vote deletion
	if uc.notifier != nil {
		_ = uc.notifier.NotifyVoteDelete(ctx, storyID, userID)
	}

	return nil
}

// GetVoteStats retrieves vote statistics for a story
func (uc *usecase) GetVoteStats(ctx context.Context, storyID uuid.UUID) (*VoteStatsResponse, error) {
	stats, err := uc.repo.GetStats(ctx, storyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return empty stats
			return &VoteStatsResponse{
				StoryID:       storyID,
				TotalVotes:    0,
				AverageRating: 0,
			}, nil
		}
		return nil, fmt.Errorf("usecase.GetVoteStats: %w", err)
	}

	return ToVoteStatsResponse(storyID, stats.TotalVotes, stats.AverageRating), nil
}

// ListStoryVotes lists votes for a story
func (uc *usecase) ListStoryVotes(ctx context.Context, storyID uuid.UUID, filter *VoteFilter) ([]VoteResponse, int64, error) {
	if filter == nil {
		filter = &VoteFilter{}
	}

	limit := filter.Limit
	offset := filter.GetOffset()

	votes, total, err := uc.repo.ListByStory(ctx, storyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListStoryVotes: %w", err)
	}

	result := make([]VoteResponse, len(votes))
	for i, v := range votes {
		result[i] = *ToVoteResponse(&v)
	}

	return result, total, nil
}

// ListUserVotes lists votes by a user
func (uc *usecase) ListUserVotes(ctx context.Context, userID uuid.UUID, filter *VoteFilter) ([]VoteResponse, int64, error) {
	if filter == nil {
		filter = &VoteFilter{}
	}

	limit := filter.Limit
	offset := filter.GetOffset()

	votes, total, err := uc.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase.ListUserVotes: %w", err)
	}

	result := make([]VoteResponse, len(votes))
	for i, v := range votes {
		result[i] = *ToVoteResponse(&v)
	}

	return result, total, nil
}

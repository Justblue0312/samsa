package story

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	CreateStory(ctx context.Context, ownerID uuid.UUID, req CreateStoryRequest) (*StoryResponse, error)
	GetStory(ctx context.Context, id uuid.UUID) (*StoryResponse, error)
	GetStoryBySlug(ctx context.Context, slug string) (*StoryResponse, error)
	ListUserStories(ctx context.Context, userID uuid.UUID, params ListStoriesParams) ([]StoryResponse, error)
	UpdateStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID, req UpdateStoryRequest) (*StoryResponse, error)
	DeleteStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) error

	// Status management
	PublishStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) (*StoryResponse, error)
	ArchiveStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) (*StoryResponse, error)

	// Voting
	VoteStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID, rating int32) (*sqlc.StoryVote, error)
	RemoveVote(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) error
	GetVoteStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateStory(ctx context.Context, ownerID uuid.UUID, req CreateStoryRequest) (*StoryResponse, error) {
	arg := sqlc.CreateStoryParams{
		OwnerID:  ownerID,
		MediaID:  req.MediaID,
		Name:     req.Name,
		Slug:     req.Slug,
		Synopsis: req.Synopsis,
		Status:   sqlc.StoryStatusDraft,
		Settings: req.Settings,
	}

	s, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Add genres
	for _, gID := range req.Genres {
		if err := uc.repo.AddGenre(ctx, s.ID, gID); err != nil {
			// Log error but continue or return?
			// For now, return error to ensure consistency
			return nil, err
		}
	}

	return ToStoryResponse(s, req.Genres), nil
}

func (uc *usecase) GetStory(ctx context.Context, id uuid.UUID) (*StoryResponse, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	genres, err := uc.repo.GetGenres(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToStoryResponse(s, genres), nil
}

func (uc *usecase) GetStoryBySlug(ctx context.Context, slug string) (*StoryResponse, error) {
	s, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	genres, err := uc.repo.GetGenres(ctx, s.ID)
	if err != nil {
		return nil, err
	}

	return ToStoryResponse(s, genres), nil
}

func (uc *usecase) ListUserStories(ctx context.Context, userID uuid.UUID, params ListStoriesParams) ([]StoryResponse, error) {
	if params.Limit == 0 {
		params.Limit = 10
	}
	arg := sqlc.GetStoriesByOwnerIDParams{
		OwnerID: userID,
		Limit:   params.Limit,
		Offset:  params.Offset,
	}
	stories, err := uc.repo.ListByOwner(ctx, arg)
	if err != nil {
		return nil, err
	}

	res := make([]StoryResponse, len(stories))
	for i, s := range stories {
		genres, err := uc.repo.GetGenres(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		res[i] = *ToStoryResponse(&s, genres)
	}
	return res, nil
}

func (uc *usecase) UpdateStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID, req UpdateStoryRequest) (*StoryResponse, error) {
	s, err := uc.repo.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}

	if s.OwnerID != userID {
		return nil, ErrPermissionDenied
	}

	arg := sqlc.UpdateStoryParams{
		ID:               storyID,
		MediaID:          s.MediaID,
		Name:             s.Name,
		Slug:             s.Slug,
		Synopsis:         s.Synopsis,
		IsVerified:       s.IsVerified,
		IsRecommended:    s.IsRecommended,
		Status:           s.Status,
		FirstPublishedAt: s.FirstPublishedAt,
		LastPublishedAt:  s.LastPublishedAt,
		Settings:         s.Settings,
		DeletedAt:        s.DeletedAt,
	}

	if req.MediaID != nil {
		arg.MediaID = *req.MediaID
	}
	if req.Name != nil {
		arg.Name = *req.Name
	}
	if req.Slug != nil {
		arg.Slug = *req.Slug
	}
	if req.Synopsis != nil {
		arg.Synopsis = req.Synopsis
	}
	if req.IsVerified != nil {
		arg.IsVerified = req.IsVerified
	}
	if req.IsRecommended != nil {
		arg.IsRecommended = req.IsRecommended
	}
	if req.Status != nil {
		arg.Status = *req.Status
	}
	if req.Settings != nil {
		arg.Settings = req.Settings
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Update genres if provided
	if req.Genres != nil {
		if err := uc.repo.ClearGenres(ctx, storyID); err != nil {
			return nil, err
		}
		for _, gID := range req.Genres {
			if err := uc.repo.AddGenre(ctx, storyID, gID); err != nil {
				return nil, err
			}
		}
	}

	genres, err := uc.repo.GetGenres(ctx, storyID)
	if err != nil {
		return nil, err
	}

	return ToStoryResponse(updated, genres), nil
}

func (uc *usecase) DeleteStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) error {
	s, err := uc.repo.GetByID(ctx, storyID)
	if err != nil {
		return err
	}

	if s.OwnerID != userID {
		return ErrPermissionDenied
	}

	_, err = uc.repo.SoftDelete(ctx, storyID)
	return err
}

func (uc *usecase) PublishStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) (*StoryResponse, error) {
	status := sqlc.StoryStatusPublished
	return uc.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{Status: &status})
}

func (uc *usecase) ArchiveStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) (*StoryResponse, error) {
	status := sqlc.StoryStatusArchived
	return uc.UpdateStory(ctx, userID, storyID, UpdateStoryRequest{Status: &status})
}

func (uc *usecase) VoteStory(ctx context.Context, userID uuid.UUID, storyID uuid.UUID, rating int32) (*sqlc.StoryVote, error) {
	return uc.repo.UpsertVote(ctx, sqlc.UpsertStoryVoteParams{
		StoryID: storyID,
		UserID:  userID,
		Rating:  rating,
	})
}

func (uc *usecase) RemoveVote(ctx context.Context, userID uuid.UUID, storyID uuid.UUID) error {
	return uc.repo.DeleteVote(ctx, sqlc.DeleteStoryVoteParams{
		StoryID: storyID,
		UserID:  userID,
	})
}

func (uc *usecase) GetVoteStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error) {
	return uc.repo.GetVoteStats(ctx, storyID)
}

// Errors
var (
	ErrPermissionDenied = &apiError{message: "permission denied"}
)

type apiError struct {
	message string
}

func (e *apiError) Error() string {
	return e.message
}

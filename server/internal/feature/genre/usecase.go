package genre

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	CreateGenre(ctx context.Context, req CreateGenreRequest) (*GenreResponse, error)
	GetGenre(ctx context.Context, id uuid.UUID) (*GenreResponse, error)
	ListGenres(ctx context.Context) ([]GenreResponse, error)
	UpdateGenre(ctx context.Context, id uuid.UUID, req UpdateGenreRequest) (*GenreResponse, error)
	DeleteGenre(ctx context.Context, id uuid.UUID) error

	// Story association
	AddGenreToStory(ctx context.Context, storyID, genreID uuid.UUID) error
	RemoveGenreFromStory(ctx context.Context, storyID, genreID uuid.UUID) error
	GetStoryGenres(ctx context.Context, storyID uuid.UUID) ([]GenreResponse, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateGenre(ctx context.Context, req CreateGenreRequest) (*GenreResponse, error) {
	arg := sqlc.CreateGenreParams{
		Name:        req.Name,
		Description: req.Description,
	}

	g, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToGenreResponse(g), nil
}

func (uc *usecase) GetGenre(ctx context.Context, id uuid.UUID) (*GenreResponse, error) {
	g, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToGenreResponse(g), nil
}

func (uc *usecase) ListGenres(ctx context.Context) ([]GenreResponse, error) {
	genres, err := uc.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]GenreResponse, len(genres))
	for i, g := range genres {
		res[i] = *ToGenreResponse(&g)
	}
	return res, nil
}

func (uc *usecase) UpdateGenre(ctx context.Context, id uuid.UUID, req UpdateGenreRequest) (*GenreResponse, error) {
	g, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	arg := sqlc.UpdateGenreParams{
		ID:          id,
		Name:        g.Name,
		Description: g.Description,
	}

	if req.Name != nil {
		arg.Name = *req.Name
	}
	if req.Description != nil {
		arg.Description = req.Description
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToGenreResponse(updated), nil
}

func (uc *usecase) DeleteGenre(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *usecase) AddGenreToStory(ctx context.Context, storyID, genreID uuid.UUID) error {
	return uc.repo.AddToStory(ctx, storyID, genreID)
}

func (uc *usecase) RemoveGenreFromStory(ctx context.Context, storyID, genreID uuid.UUID) error {
	return uc.repo.RemoveFromStory(ctx, storyID, genreID)
}

func (uc *usecase) GetStoryGenres(ctx context.Context, storyID uuid.UUID) ([]GenreResponse, error) {
	genres, err := uc.repo.GetByStoryID(ctx, storyID)
	if err != nil {
		return nil, err
	}

	res := make([]GenreResponse, len(genres))
	for i, g := range genres {
		res[i] = *ToGenreResponse(&g)
	}
	return res, nil
}

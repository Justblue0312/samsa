package genre

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	Create(ctx context.Context, arg sqlc.CreateGenreParams) (*sqlc.Genre, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Genre, error)
	GetByName(ctx context.Context, name string) (*sqlc.Genre, error)
	List(ctx context.Context) ([]sqlc.Genre, error)
	Update(ctx context.Context, arg sqlc.UpdateGenreParams) (*sqlc.Genre, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// Story association
	AddToStory(ctx context.Context, storyID, genreID uuid.UUID) error
	RemoveFromStory(ctx context.Context, storyID, genreID uuid.UUID) error
	GetByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Genre, error)
	ClearFromStory(ctx context.Context, storyID uuid.UUID) error
}

type repository struct {
	queries *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		queries: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateGenreParams) (*sqlc.Genre, error) {
	g, err := r.queries.CreateGenre(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Genre, error) {
	g, err := r.queries.GetGenreByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repository) GetByName(ctx context.Context, name string) (*sqlc.Genre, error) {
	g, err := r.queries.GetGenreByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repository) List(ctx context.Context) ([]sqlc.Genre, error) {
	return r.queries.ListGenres(ctx)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateGenreParams) (*sqlc.Genre, error) {
	g, err := r.queries.UpdateGenre(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteGenre(ctx, id)
}

func (r *repository) AddToStory(ctx context.Context, storyID, genreID uuid.UUID) error {
	return r.queries.AddGenreToStory(ctx, sqlc.AddGenreToStoryParams{
		StoryID: storyID,
		GenreID: genreID,
	})
}

func (r *repository) RemoveFromStory(ctx context.Context, storyID, genreID uuid.UUID) error {
	return r.queries.RemoveGenreFromStory(ctx, sqlc.RemoveGenreFromStoryParams{
		StoryID: storyID,
		GenreID: genreID,
	})
}

func (r *repository) GetByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.Genre, error) {
	return r.queries.GetGenresByStoryID(ctx, storyID)
}

func (r *repository) ClearFromStory(ctx context.Context, storyID uuid.UUID) error {
	return r.queries.ClearGenresFromStory(ctx, storyID)
}

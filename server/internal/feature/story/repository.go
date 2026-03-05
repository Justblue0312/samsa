package story

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	Create(ctx context.Context, arg sqlc.CreateStoryParams) (*sqlc.Story, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Story, error)
	GetBySlug(ctx context.Context, slug string) (*sqlc.Story, error)
	ListByOwner(ctx context.Context, arg sqlc.GetStoriesByOwnerIDParams) ([]sqlc.Story, error)
	Update(ctx context.Context, arg sqlc.UpdateStoryParams) (*sqlc.Story, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Story, error)

	// Status History
	CreateStatusHistory(ctx context.Context, arg sqlc.CreateStoryStatusHistoryParams) (*sqlc.StoryStatusHistory, error)
	ListStatusHistory(ctx context.Context, storyID uuid.UUID) ([]sqlc.StoryStatusHistory, error)

	// Voting
	UpsertVote(ctx context.Context, arg sqlc.UpsertStoryVoteParams) (*sqlc.StoryVote, error)
	GetVote(ctx context.Context, arg sqlc.GetStoryVoteParams) (*sqlc.StoryVote, error)
	DeleteVote(ctx context.Context, arg sqlc.DeleteStoryVoteParams) error
	GetVoteStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error)

	// Genre association
	AddGenre(ctx context.Context, storyID, genreID uuid.UUID) error
	RemoveGenre(ctx context.Context, storyID, genreID uuid.UUID) error
	GetGenres(ctx context.Context, storyID uuid.UUID) ([]uuid.UUID, error)
	ClearGenres(ctx context.Context, storyID uuid.UUID) error
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateStoryParams) (*sqlc.Story, error) {
	s, err := r.q.CreateStory(ctx, arg)
	return &s, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Story, error) {
	s, err := r.q.GetStoryByID(ctx, id)
	return &s, err
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*sqlc.Story, error) {
	s, err := r.q.GetStoryBySlug(ctx, slug)
	return &s, err
}

func (r *repository) ListByOwner(ctx context.Context, arg sqlc.GetStoriesByOwnerIDParams) ([]sqlc.Story, error) {
	return r.q.GetStoriesByOwnerID(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateStoryParams) (*sqlc.Story, error) {
	s, err := r.q.UpdateStory(ctx, arg)
	return &s, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteStory(ctx, id)
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Story, error) {
	s, err := r.q.SoftDeleteStory(ctx, id)
	return &s, err
}

func (r *repository) CreateStatusHistory(ctx context.Context, arg sqlc.CreateStoryStatusHistoryParams) (*sqlc.StoryStatusHistory, error) {
	h, err := r.q.CreateStoryStatusHistory(ctx, arg)
	return &h, err
}

func (r *repository) ListStatusHistory(ctx context.Context, storyID uuid.UUID) ([]sqlc.StoryStatusHistory, error) {
	return r.q.ListStoryStatusHistoryByStory(ctx, storyID)
}

func (r *repository) UpsertVote(ctx context.Context, arg sqlc.UpsertStoryVoteParams) (*sqlc.StoryVote, error) {
	v, err := r.q.UpsertStoryVote(ctx, arg)
	return &v, err
}

func (r *repository) GetVote(ctx context.Context, arg sqlc.GetStoryVoteParams) (*sqlc.StoryVote, error) {
	v, err := r.q.GetStoryVote(ctx, arg)
	return &v, err
}

func (r *repository) DeleteVote(ctx context.Context, arg sqlc.DeleteStoryVoteParams) error {
	return r.q.DeleteStoryVote(ctx, arg)
}

func (r *repository) GetVoteStats(ctx context.Context, storyID uuid.UUID) (sqlc.GetStoryVoteStatsRow, error) {
	return r.q.GetStoryVoteStats(ctx, storyID)
}

func (r *repository) AddGenre(ctx context.Context, storyID, genreID uuid.UUID) error {
	return r.q.AddGenreToStory(ctx, sqlc.AddGenreToStoryParams{
		StoryID: storyID,
		GenreID: genreID,
	})
}

func (r *repository) RemoveGenre(ctx context.Context, storyID, genreID uuid.UUID) error {
	return r.q.RemoveGenreFromStory(ctx, sqlc.RemoveGenreFromStoryParams{
		StoryID: storyID,
		GenreID: genreID,
	})
}

func (r *repository) GetGenres(ctx context.Context, storyID uuid.UUID) ([]uuid.UUID, error) {
	genres, err := r.q.GetGenresByStoryID(ctx, storyID)
	if err != nil {
		return nil, err
	}

	res := make([]uuid.UUID, len(genres))
	for i, g := range genres {
		res[i] = g.ID
	}
	return res, nil
}

func (r *repository) ClearGenres(ctx context.Context, storyID uuid.UUID) error {
	return r.q.ClearGenresFromStory(ctx, storyID)
}

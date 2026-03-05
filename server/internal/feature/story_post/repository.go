package story_post

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	Create(ctx context.Context, arg sqlc.CreateStoryPostParams) (*sqlc.StoryPost, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryPost, error)
	ListByAuthor(ctx context.Context, arg sqlc.ListStoryPostsByAuthorParams) ([]sqlc.StoryPost, error)
	ListByStory(ctx context.Context, arg sqlc.ListStoryPostsByStoryParams) ([]sqlc.StoryPost, error)
	ListByStoryFiltered(ctx context.Context, arg sqlc.ListStoryPostsByStoryWithFiltersParams) ([]sqlc.StoryPost, error)
	Update(ctx context.Context, arg sqlc.UpdateStoryPostParams) (*sqlc.StoryPost, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*sqlc.StoryPost, error)
	PermanentlyDelete(ctx context.Context, id uuid.UUID) error
	BulkDelete(ctx context.Context, ids []uuid.UUID) error
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.StoryPost, error)
	CountByStory(ctx context.Context, storyID uuid.UUID) (int64, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateStoryPostParams) (*sqlc.StoryPost, error) {
	p, err := r.q.CreateStoryPost(ctx, arg)
	return &p, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.StoryPost, error) {
	p, err := r.q.GetStoryPostByID(ctx, id)
	return &p, err
}

func (r *repository) ListByAuthor(ctx context.Context, arg sqlc.ListStoryPostsByAuthorParams) ([]sqlc.StoryPost, error) {
	return r.q.ListStoryPostsByAuthor(ctx, arg)
}

func (r *repository) ListByStory(ctx context.Context, arg sqlc.ListStoryPostsByStoryParams) ([]sqlc.StoryPost, error) {
	return r.q.ListStoryPostsByStory(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateStoryPostParams) (*sqlc.StoryPost, error) {
	p, err := r.q.UpdateStoryPost(ctx, arg)
	return &p, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteStoryPost(ctx, id)
}

func (r *repository) Restore(ctx context.Context, id uuid.UUID) (*sqlc.StoryPost, error) {
	p, err := r.q.RestoreStoryPost(ctx, id)
	return &p, err
}

func (r *repository) PermanentlyDelete(ctx context.Context, id uuid.UUID) error {
	return r.q.PermanentlyDeleteStoryPost(ctx, id)
}

func (r *repository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	return r.q.BulkDeleteStoryPosts(ctx, ids)
}

func (r *repository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]sqlc.StoryPost, error) {
	return r.q.GetStoryPostsByIDs(ctx, ids)
}

func (r *repository) CountByStory(ctx context.Context, storyID uuid.UUID) (int64, error) {
	return r.q.CountStoryPostsByStory(ctx, &storyID)
}

func (r *repository) ListByStoryFiltered(ctx context.Context, arg sqlc.ListStoryPostsByStoryWithFiltersParams) ([]sqlc.StoryPost, error) {
	return r.q.ListStoryPostsByStoryWithFilters(ctx, arg)
}

package story_post

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	CreatePost(ctx context.Context, req CreateStoryPostRequest) (*StoryPostResponse, error)
	GetPost(ctx context.Context, id uuid.UUID) (*StoryPostResponse, error)
	ListAuthorPosts(ctx context.Context, authorID uuid.UUID, limit, offset int32) ([]StoryPostResponse, error)
	ListStoryPosts(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]StoryPostResponse, error)
	ListStoryPostsFiltered(ctx context.Context, storyID, authorID *uuid.UUID, includeDeleted bool, limit, offset int32) ([]StoryPostResponse, int64, error)
	UpdatePost(ctx context.Context, postID uuid.UUID, req UpdateStoryPostRequest) (*StoryPostResponse, error)
	DeletePost(ctx context.Context, postID uuid.UUID) error
	RestorePost(ctx context.Context, postID uuid.UUID) (*StoryPostResponse, error)
	PermanentlyDeletePost(ctx context.Context, postID uuid.UUID) error
	BulkDeletePosts(ctx context.Context, postIDs []uuid.UUID) error
	GetPostsByIDs(ctx context.Context, ids []uuid.UUID) ([]StoryPostResponse, error)
	CountStoryPosts(ctx context.Context, storyID uuid.UUID) (int64, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreatePost(ctx context.Context, req CreateStoryPostRequest) (*StoryPostResponse, error) {
	arg := sqlc.CreateStoryPostParams{
		AuthorID:          req.AuthorID,
		Content:           req.Content,
		MediaIds:          req.MediaIds,
		StoryID:           req.StoryID,
		ChapterID:         req.ChapterID,
		IsNotifyFollowers: &req.IsNotifyFollowers,
	}

	p, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToStoryPostResponse(p), nil
}

func (uc *usecase) GetPost(ctx context.Context, id uuid.UUID) (*StoryPostResponse, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToStoryPostResponse(p), nil
}

func (uc *usecase) ListAuthorPosts(ctx context.Context, authorID uuid.UUID, limit, offset int32) ([]StoryPostResponse, error) {
	if limit == 0 {
		limit = 10
	}
	arg := sqlc.ListStoryPostsByAuthorParams{
		AuthorID: authorID,
		Limit:    limit,
		Offset:   offset,
	}
	posts, err := uc.repo.ListByAuthor(ctx, arg)
	if err != nil {
		return nil, err
	}

	res := make([]StoryPostResponse, len(posts))
	for i, p := range posts {
		res[i] = *ToStoryPostResponse(&p)
	}
	return res, nil
}

func (uc *usecase) ListStoryPosts(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]StoryPostResponse, error) {
	if limit == 0 {
		limit = 10
	}
	arg := sqlc.ListStoryPostsByStoryParams{
		StoryID: &storyID,
		Limit:   limit,
		Offset:  offset,
	}
	posts, err := uc.repo.ListByStory(ctx, arg)
	if err != nil {
		return nil, err
	}

	res := make([]StoryPostResponse, len(posts))
	for i, p := range posts {
		res[i] = *ToStoryPostResponse(&p)
	}
	return res, nil
}

func (uc *usecase) UpdatePost(ctx context.Context, postID uuid.UUID, req UpdateStoryPostRequest) (*StoryPostResponse, error) {
	p, err := uc.repo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	arg := sqlc.UpdateStoryPostParams{
		ID:                postID,
		Content:           p.Content,
		MediaIds:          p.MediaIds,
		IsNotifyFollowers: p.IsNotifyFollowers,
	}

	if req.Content != nil {
		arg.Content = *req.Content
	}
	if req.MediaIds != nil {
		arg.MediaIds = req.MediaIds
	}
	if req.IsNotifyFollowers != nil {
		arg.IsNotifyFollowers = req.IsNotifyFollowers
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToStoryPostResponse(updated), nil
}

func (uc *usecase) DeletePost(ctx context.Context, postID uuid.UUID) error {
	return uc.repo.Delete(ctx, postID)
}

func (uc *usecase) RestorePost(ctx context.Context, postID uuid.UUID) (*StoryPostResponse, error) {
	p, err := uc.repo.Restore(ctx, postID)
	if err != nil {
		return nil, err
	}
	return ToStoryPostResponse(p), nil
}

func (uc *usecase) PermanentlyDeletePost(ctx context.Context, postID uuid.UUID) error {
	return uc.repo.PermanentlyDelete(ctx, postID)
}

func (uc *usecase) BulkDeletePosts(ctx context.Context, postIDs []uuid.UUID) error {
	return uc.repo.BulkDelete(ctx, postIDs)
}

func (uc *usecase) GetPostsByIDs(ctx context.Context, ids []uuid.UUID) ([]StoryPostResponse, error) {
	posts, err := uc.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := make([]StoryPostResponse, len(posts))
	for i, p := range posts {
		res[i] = *ToStoryPostResponse(&p)
	}
	return res, nil
}

func (uc *usecase) CountStoryPosts(ctx context.Context, storyID uuid.UUID) (int64, error) {
	return uc.repo.CountByStory(ctx, storyID)
}

func (uc *usecase) ListStoryPostsFiltered(ctx context.Context, storyID, authorID *uuid.UUID, includeDeleted bool, limit, offset int32) ([]StoryPostResponse, int64, error) {
	if limit == 0 {
		limit = 20
	}
	arg := sqlc.ListStoryPostsByStoryWithFiltersParams{
		StoryID:        storyID,
		AuthorID:       authorID,
		IncludeDeleted: &includeDeleted,
		Limit:          limit,
		Offset:         offset,
	}
	posts, err := uc.repo.ListByStoryFiltered(ctx, arg)
	if err != nil {
		return nil, 0, err
	}
	// Count total for pagination
	var total int64
	if storyID != nil {
		total, _ = uc.repo.CountByStory(ctx, *storyID)
	}
	res := make([]StoryPostResponse, len(posts))
	for i, p := range posts {
		res[i] = *ToStoryPostResponse(&p)
	}
	return res, total, nil
}

package flag

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type UseCase interface {
	CreateFlag(ctx context.Context, inspectorID uuid.UUID, req CreateFlagRequest) (*FlagResponse, error)
	GetFlag(ctx context.Context, id uuid.UUID) (*FlagResponse, error)
	ListFlags(ctx context.Context, params ListFlagsParams) (*FlagListResponse, error)
	UpdateFlag(ctx context.Context, id uuid.UUID, req UpdateFlagRequest) (*FlagResponse, error)
	DeleteFlag(ctx context.Context, id uuid.UUID) error
	ListFlagsByStory(ctx context.Context, storyID uuid.UUID, page, limit int32) ([]FlagResponse, int64, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateFlag(ctx context.Context, inspectorID uuid.UUID, req CreateFlagRequest) (*FlagResponse, error) {
	arg := sqlc.CreateFlagParams{
		StoryID:     req.StoryID,
		ChapterID:   req.ChapterID,
		InspectorID: &inspectorID,
		Title:       req.Title,
		Description: req.Description,
		FlagType:    sqlc.FlagTypes(req.FlagType),
		FlagRate:    sqlc.FlagRate(req.FlagRate),
		FlagScore:   req.FlagScore,
	}

	f, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToFlagResponse(f), nil
}

func (uc *usecase) GetFlag(ctx context.Context, id uuid.UUID) (*FlagResponse, error) {
	f, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToFlagResponse(f), nil
}

func (uc *usecase) ListFlags(ctx context.Context, params ListFlagsParams) (*FlagListResponse, error) {
	if params.Limit == 0 {
		params.Limit = 10
	}

	flags, err := uc.repo.ListAll(ctx, params)
	if err != nil {
		return nil, err
	}

	count, err := uc.repo.GetCount(ctx, params)
	if err != nil {
		return nil, err
	}

	page := int32(1)
	if params.Offset > 0 {
		page = (params.Offset / params.Limit) + 1
	}

	return ToFlagListResponse(flags, count, page, params.Limit), nil
}

func (uc *usecase) UpdateFlag(ctx context.Context, id uuid.UUID, req UpdateFlagRequest) (*FlagResponse, error) {
	f, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	arg := sqlc.UpdateFlagParams{
		ID:          id,
		Title:       f.Title,
		Description: f.Description,
		FlagType:    f.FlagType,
		FlagRate:    f.FlagRate,
		FlagScore:   f.FlagScore,
	}

	if req.Title != nil {
		arg.Title = *req.Title
	}
	if req.Description != nil {
		arg.Description = req.Description
	}
	if req.FlagType != nil {
		arg.FlagType = sqlc.FlagTypes(*req.FlagType)
	}
	if req.FlagRate != nil {
		arg.FlagRate = sqlc.FlagRate(*req.FlagRate)
	}
	if req.FlagScore != nil {
		arg.FlagScore = *req.FlagScore
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToFlagResponse(updated), nil
}

func (uc *usecase) DeleteFlag(ctx context.Context, id uuid.UUID) error {
	_, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return uc.repo.Delete(ctx, id)
}

func (uc *usecase) ListFlagsByStory(ctx context.Context, storyID uuid.UUID, page, limit int32) ([]FlagResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	params := ListFlagsParams{
		StoryID: &storyID,
		Limit:   limit,
		Offset:  offset,
	}

	flags, err := uc.repo.ListAll(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	count, err := uc.repo.GetCount(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	res := make([]FlagResponse, len(flags))
	for i, f := range flags {
		res[i] = *ToFlagResponse(&f)
	}

	return res, count, nil
}

// Errors
var (
	ErrFlagNotFound     = errors.New("flag not found")
	ErrStoryNotFound    = errors.New("story not found")
	ErrChapterNotFound  = errors.New("chapter not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidFlagScore = errors.New("invalid flag score: must be between 0 and 100")
)

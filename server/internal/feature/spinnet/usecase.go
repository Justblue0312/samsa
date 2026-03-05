package spinnet

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type UseCase interface {
	GetByID(ctx context.Context, id uuid.UUID) (*SpinnetResponse, error)
	GetBySmartSyntax(ctx context.Context, smartSyntax string) (*SpinnetResponse, error)
	List(ctx context.Context, params ListSpinnetsParams) (*SpinnetListResponse, error)
	Create(ctx context.Context, req CreateSpinnetRequest) (*SpinnetResponse, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateSpinnetRequest) (*SpinnetResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) GetByID(ctx context.Context, id uuid.UUID) (*SpinnetResponse, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToSpinnetResponse(s), nil
}

func (uc *usecase) GetBySmartSyntax(ctx context.Context, smartSyntax string) (*SpinnetResponse, error) {
	s, err := uc.repo.GetBySmartSyntax(ctx, smartSyntax)
	if err != nil {
		return nil, err
	}

	return ToSpinnetResponse(s), nil
}

func (uc *usecase) List(ctx context.Context, params ListSpinnetsParams) (*SpinnetListResponse, error) {
	if params.Limit == 0 {
		params.Limit = 10
	}

	spinnets, err := uc.repo.List(ctx, params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}

	count := int64(len(spinnets))
	page := int32(1)
	if params.Offset > 0 {
		page = (params.Offset / params.Limit) + 1
	}

	return ToSpinnetListResponse(spinnets, count, page, params.Limit), nil
}

func (uc *usecase) Create(ctx context.Context, req CreateSpinnetRequest) (*SpinnetResponse, error) {
	now := time.Now()
	id := uuid.New()

	arg := sqlc.CreateSpinnetParams{
		ID:          id,
		OwnerID:     req.OwnerID,
		Name:        req.Name,
		Content:     req.Content,
		Category:    req.Category,
		SmartSyntax: req.SmartSyntax,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	s, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToSpinnetResponse(s), nil
}

func (uc *usecase) Update(ctx context.Context, id uuid.UUID, req UpdateSpinnetRequest) (*SpinnetResponse, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	arg := sqlc.UpdateSpinnetParams{
		ID:          id,
		Name:        req.Name,
		Content:     req.Content,
		Category:    req.Category,
		SmartSyntax: req.SmartSyntax,
		UpdatedAt:   &now,
	}

	// Use existing values if new values are not provided
	if req.Name == "" {
		arg.Name = s.Name
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToSpinnetResponse(updated), nil
}

func (uc *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return uc.repo.Delete(ctx, id)
}

// Errors
var (
	ErrSpinnetNotFound    = errors.New("spinnet not found")
	ErrSpinnetExists      = errors.New("spinnet with this smart syntax already exists")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrInvalidSmartSyntax = errors.New("invalid smart syntax format")
)

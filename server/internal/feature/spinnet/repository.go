package spinnet

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Spinnet, error)
	GetBySmartSyntax(ctx context.Context, smartSyntax string) (*sqlc.Spinnet, error)
	List(ctx context.Context, limit, offset int32) ([]sqlc.Spinnet, error)
	Create(ctx context.Context, arg sqlc.CreateSpinnetParams) (*sqlc.Spinnet, error)
	Update(ctx context.Context, arg sqlc.UpdateSpinnetParams) (*sqlc.Spinnet, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(q *sqlc.Queries) Repository {
	return &repository{
		q: q,
	}
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Spinnet, error) {
	s, err := r.q.GetSpinnetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) GetBySmartSyntax(ctx context.Context, smartSyntax string) (*sqlc.Spinnet, error) {
	s, err := r.q.GetSpinnetBySmartSyntax(ctx, &smartSyntax)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) List(ctx context.Context, limit, offset int32) ([]sqlc.Spinnet, error) {
	return r.q.ListSpinnets(ctx, sqlc.ListSpinnetsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateSpinnetParams) (*sqlc.Spinnet, error) {
	s, err := r.q.CreateSpinnet(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateSpinnetParams) (*sqlc.Spinnet, error) {
	s, err := r.q.UpdateSpinnet(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	deletedAt := time.Now()
	return r.q.DeleteSpinnet(ctx, sqlc.DeleteSpinnetParams{
		ID:        id,
		DeletedAt: &deletedAt,
	})
}

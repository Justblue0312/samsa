package flag

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/justblue/samsa/gen/sqlc"
)

type Repository interface {
	Create(ctx context.Context, arg sqlc.CreateFlagParams) (*sqlc.Flag, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Flag, error)
	ListByStory(ctx context.Context, storyID uuid.UUID) ([]sqlc.Flag, error)
	ListByChapter(ctx context.Context, chapterID *uuid.UUID) ([]sqlc.Flag, error)
	ListByInspector(ctx context.Context, inspectorID *uuid.UUID, limit, offset int32) ([]sqlc.Flag, error)
	ListAll(ctx context.Context, params ListFlagsParams) ([]sqlc.Flag, error)
	GetCount(ctx context.Context, params ListFlagsParams) (int64, error)
	Update(ctx context.Context, arg sqlc.UpdateFlagParams) (*sqlc.Flag, error)
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

func (r *repository) Create(ctx context.Context, arg sqlc.CreateFlagParams) (*sqlc.Flag, error) {
	f, err := r.q.CreateFlag(ctx, arg)
	return &f, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Flag, error) {
	f, err := r.q.GetFlagByID(ctx, id)
	return &f, err
}

func (r *repository) ListByStory(ctx context.Context, storyID uuid.UUID) ([]sqlc.Flag, error) {
	return r.q.ListFlagsByStory(ctx, storyID)
}

func (r *repository) ListByChapter(ctx context.Context, chapterID *uuid.UUID) ([]sqlc.Flag, error) {
	return r.q.ListFlagsByChapter(ctx, chapterID)
}

func (r *repository) ListByInspector(ctx context.Context, inspectorID *uuid.UUID, limit, offset int32) ([]sqlc.Flag, error) {
	arg := sqlc.ListFlagsByInspectorParams{
		InspectorID: inspectorID,
		Limit:       limit,
		Offset:      offset,
	}
	return r.q.ListFlagsByInspector(ctx, arg)
}

func (r *repository) ListAll(ctx context.Context, params ListFlagsParams) ([]sqlc.Flag, error) {
	if params.Limit == 0 {
		params.Limit = 10
	}

	arg := sqlc.ListFlagsParams{
		StoryID:     params.StoryID,
		ChapterID:   params.ChapterID,
		InspectorID: params.InspectorID,
		Limit:       params.Limit,
		Offset:      params.Offset,
	}

	if params.FlagType != nil {
		arg.FlagType = sqlc.NullFlagTypes{
			FlagTypes: sqlc.FlagTypes(*params.FlagType),
			Valid:     true,
		}
	}

	if params.FlagRate != nil {
		arg.FlagRate = sqlc.NullFlagRate{
			FlagRate: sqlc.FlagRate(*params.FlagRate),
			Valid:    true,
		}
	}

	if params.MinScore != nil {
		arg.MinScore = pgtype.Float8{
			Float64: *params.MinScore,
			Valid:   true,
		}
	}

	if params.MaxScore != nil {
		arg.MaxScore = pgtype.Float8{
			Float64: *params.MaxScore,
			Valid:   true,
		}
	}

	return r.q.ListFlags(ctx, arg)
}

func (r *repository) GetCount(ctx context.Context, params ListFlagsParams) (int64, error) {
	arg := sqlc.GetFlagCountParams{}

	if params.StoryID != nil {
		arg.StoryID = *params.StoryID
	}
	if params.ChapterID != nil {
		arg.ChapterID = *params.ChapterID
	}
	if params.InspectorID != nil {
		arg.InspectorID = *params.InspectorID
	}
	if params.FlagType != nil {
		arg.FlagType = sqlc.FlagTypes(*params.FlagType)
	}
	if params.FlagRate != nil {
		arg.FlagRate = sqlc.FlagRate(*params.FlagRate)
	}

	return r.q.GetFlagCount(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateFlagParams) (*sqlc.Flag, error) {
	f, err := r.q.UpdateFlag(ctx, arg)
	return &f, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteFlag(ctx, id)
}

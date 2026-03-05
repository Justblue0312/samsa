package genre

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type CreateGenreRequest struct {
	Name        string  `json:"name" validate:"required,max=255"`
	Description *string `json:"description"`
}

type UpdateGenreRequest struct {
	Name        *string `json:"name" validate:"omitempty,max=255"`
	Description *string `json:"description"`
}

type GenreResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToGenreResponse(g *sqlc.Genre) *GenreResponse {
	if g == nil {
		return nil
	}

	res := &GenreResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
	}

	if g.CreatedAt != nil {
		res.CreatedAt = *g.CreatedAt
	}
	if g.UpdatedAt != nil {
		res.UpdatedAt = *g.UpdatedAt
	}

	return res
}

type StoryGenreRequest struct {
	StoryID uuid.UUID `json:"story_id" validate:"required"`
	GenreID uuid.UUID `json:"genre_id" validate:"required"`
}

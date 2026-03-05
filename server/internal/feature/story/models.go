package story

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type CreateStoryRequest struct {
	MediaID  uuid.UUID   `json:"media_id" validate:"required"`
	Name     string      `json:"name" validate:"required,max=255"`
	Slug     string      `json:"slug" validate:"required,max=255"`
	Synopsis *string     `json:"synopsis"`
	Settings []byte      `json:"settings"`
	Genres   []uuid.UUID `json:"genres"`
}

type UpdateStoryRequest struct {
	MediaID       *uuid.UUID        `json:"media_id"`
	Name          *string           `json:"name" validate:"omitempty,max=255"`
	Slug          *string           `json:"slug" validate:"omitempty,max=255"`
	Synopsis      *string           `json:"synopsis"`
	IsVerified    *bool             `json:"is_verified"`
	IsRecommended *bool             `json:"is_recommended"`
	Status        *sqlc.StoryStatus `json:"status"`
	Settings      []byte            `json:"settings"`
	Genres        []uuid.UUID       `json:"genres"`
}

type StoryResponse struct {
	ID               uuid.UUID        `json:"id"`
	OwnerID          uuid.UUID        `json:"owner_id"`
	MediaID          uuid.UUID        `json:"media_id"`
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	Synopsis         *string          `json:"synopsis"`
	IsVerified       bool             `json:"is_verified"`
	IsRecommended    bool             `json:"is_recommended"`
	Status           sqlc.StoryStatus `json:"status"`
	FirstPublishedAt *time.Time       `json:"first_published_at"`
	LastPublishedAt  *time.Time       `json:"last_published_at"`
	Settings         []byte           `json:"settings"`
	Genres           []uuid.UUID      `json:"genres,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type ListStoriesParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type SearchStoriesRequest struct {
	Query  string `json:"query"`
	Limit  int32  `json:"limit"`
	Offset int32  `json:"offset"`
}

func ToStoryResponse(s *sqlc.Story, genres []uuid.UUID) *StoryResponse {
	if s == nil {
		return nil
	}

	res := &StoryResponse{
		ID:       s.ID,
		OwnerID:  s.OwnerID,
		MediaID:  s.MediaID,
		Name:     s.Name,
		Slug:     s.Slug,
		Synopsis: s.Synopsis,
		Status:   s.Status,
		Settings: s.Settings,
		Genres:   genres,
	}

	if s.IsVerified != nil {
		res.IsVerified = *s.IsVerified
	}
	if s.IsRecommended != nil {
		res.IsRecommended = *s.IsRecommended
	}
	if s.FirstPublishedAt != nil {
		res.FirstPublishedAt = s.FirstPublishedAt
	}
	if s.LastPublishedAt != nil {
		res.LastPublishedAt = s.LastPublishedAt
	}
	if s.CreatedAt != nil {
		res.CreatedAt = *s.CreatedAt
	}
	if s.UpdatedAt != nil {
		res.UpdatedAt = *s.UpdatedAt
	}

	return res
}

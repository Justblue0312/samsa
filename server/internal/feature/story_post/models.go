package story_post

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type CreateStoryPostRequest struct {
	AuthorID          uuid.UUID   `json:"author_id" validate:"required"`
	Content           string      `json:"content" validate:"required"`
	MediaIds          []uuid.UUID `json:"media_ids"`
	StoryID           *uuid.UUID  `json:"story_id"`
	ChapterID         *uuid.UUID  `json:"chapter_id"`
	IsNotifyFollowers bool        `json:"is_notify_followers"`
}

type UpdateStoryPostRequest struct {
	Content           *string     `json:"content"`
	MediaIds          []uuid.UUID `json:"media_ids"`
	IsNotifyFollowers *bool       `json:"is_notify_followers"`
}

type StoryPostResponse struct {
	ID                uuid.UUID   `json:"id"`
	AuthorID          uuid.UUID   `json:"author_id"`
	Content           string      `json:"content"`
	MediaIds          []uuid.UUID `json:"media_ids"`
	StoryID           *uuid.UUID  `json:"story_id"`
	ChapterID         *uuid.UUID  `json:"chapter_id"`
	IsNotifyFollowers bool        `json:"is_notify_followers"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

func ToStoryPostResponse(p *sqlc.StoryPost) *StoryPostResponse {
	if p == nil {
		return nil
	}

	res := &StoryPostResponse{
		ID:        p.ID,
		AuthorID:  p.AuthorID,
		Content:   p.Content,
		MediaIds:  p.MediaIds,
		StoryID:   p.StoryID,
		ChapterID: p.ChapterID,
	}

	if p.IsNotifyFollowers != nil {
		res.IsNotifyFollowers = *p.IsNotifyFollowers
	}
	if p.CreatedAt != nil {
		res.CreatedAt = *p.CreatedAt
	}
	if p.UpdatedAt != nil {
		res.UpdatedAt = *p.UpdatedAt
	}

	return res
}

// StoryPostListResponse represents a paginated list of story posts
type StoryPostListResponse struct {
	Posts []StoryPostResponse       `json:"posts"`
	Meta  queryparam.PaginationMeta `json:"meta"`
}

// BulkDeleteRequest represents a request to delete multiple posts
type BulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1"`
}

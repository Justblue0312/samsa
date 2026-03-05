package chapter

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

// CreateChapterRequest represents a request to create a new chapter.
type CreateChapterRequest struct {
	StoryID     uuid.UUID `json:"story_id" validate:"required"`
	Title       string    `json:"title" validate:"required,max=500"`
	Number      *int32    `json:"number"`
	SortOrder   *int32    `json:"sort_order"`
	Summary     *string   `json:"summary"`
	IsPublished *bool     `json:"is_published"`
	TotalWords  *int32    `json:"total_words"`
}

// UpdateChapterRequest represents a request to update an existing chapter.
type UpdateChapterRequest struct {
	Title       *string `json:"title" validate:"omitempty,max=500"`
	Number      *int32  `json:"number"`
	SortOrder   *int32  `json:"sort_order"`
	Summary     *string `json:"summary"`
	IsPublished *bool   `json:"is_published"`
	TotalWords  *int32  `json:"total_words"`
}

// ChapterResponse represents a chapter response.
type ChapterResponse struct {
	ID             uuid.UUID  `json:"id"`
	StoryID        uuid.UUID  `json:"story_id"`
	Title          string     `json:"title"`
	Number         *int32     `json:"number"`
	SortOrder      *int32     `json:"sort_order"`
	Summary        *string    `json:"summary"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	TotalWords     int32      `json:"total_words"`
	TotalViews     int32      `json:"total_views"`
	TotalVotes     int32      `json:"total_votes"`
	TotalFavorites int32      `json:"total_favorites"`
	TotalBookmarks int32      `json:"total_bookmarks"`
	TotalFlags     int32      `json:"total_flags"`
	TotalReports   int32      `json:"total_reports"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ListChaptersParams represents parameters for listing chapters.
type ListChaptersParams struct {
	StoryID      uuid.UUID `json:"story_id"`
	IsPublished  *bool     `json:"is_published"`
	Limit        int32     `json:"limit"`
	Offset       int32     `json:"offset"`
	IncludeStats *bool     `json:"include_stats"`
}

// ReorderChapterRequest represents a request to reorder a chapter.
type ReorderChapterRequest struct {
	StoryID   uuid.UUID `json:"story_id" validate:"required"`
	SortOrder int32     `json:"sort_order" validate:"required"`
}

// PublishChapterRequest represents a request to publish/unpublish a chapter.
type PublishChapterRequest struct {
	IsPublished bool `json:"is_published"`
}

// ToChapterResponse converts a sqlc.Chapter to a ChapterResponse.
func ToChapterResponse(c *sqlc.Chapter) *ChapterResponse {
	if c == nil {
		return nil
	}

	res := &ChapterResponse{
		ID:        c.ID,
		StoryID:   c.StoryID,
		Title:     c.Title,
		Number:    c.Number,
		SortOrder: c.SortOrder,
		Summary:   c.Summary,
	}

	if c.IsPublished != nil {
		res.IsPublished = *c.IsPublished
	}
	if c.PublishedAt != nil {
		res.PublishedAt = c.PublishedAt
	}
	if c.TotalWords != nil {
		res.TotalWords = *c.TotalWords
	}
	if c.TotalViews != nil {
		res.TotalViews = *c.TotalViews
	}
	if c.TotalVotes != nil {
		res.TotalVotes = *c.TotalVotes
	}
	if c.TotalFavorites != nil {
		res.TotalFavorites = *c.TotalFavorites
	}
	if c.TotalBookmarks != nil {
		res.TotalBookmarks = *c.TotalBookmarks
	}
	if c.TotalFlags != nil {
		res.TotalFlags = *c.TotalFlags
	}
	if c.TotalReports != nil {
		res.TotalReports = *c.TotalReports
	}
	if c.CreatedAt != nil {
		res.CreatedAt = *c.CreatedAt
	}
	if c.UpdatedAt != nil {
		res.UpdatedAt = *c.UpdatedAt
	}

	return res
}

// ToChapterListResponse converts a slice of sqlc.Chapter to a slice of ChapterResponse.
func ToChapterListResponse(chapters []sqlc.Chapter) []ChapterResponse {
	res := make([]ChapterResponse, len(chapters))
	for i, c := range chapters {
		res[i] = *ToChapterResponse(&c)
	}
	return res
}

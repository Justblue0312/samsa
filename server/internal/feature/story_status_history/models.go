package story_status_history

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

// StatusHistoryResponse represents a status change entry in API responses
type StatusHistoryResponse struct {
	ID          uuid.UUID        `json:"id"`
	StoryID     uuid.UUID        `json:"story_id"`
	SetStatusBy uuid.UUID        `json:"set_status_by"`
	Content     string           `json:"content"`
	Status      sqlc.StoryStatus `json:"status"`
	Reason      *string          `json:"reason,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// StatusHistoryListResponse represents a list of status history entries
type StatusHistoryListResponse struct {
	History []StatusHistoryResponse `json:"history"`
	Total   int64                   `json:"total"`
}

// ToStatusHistoryResponse converts a sqlc.StoryStatusHistory to response format
func ToStatusHistoryResponse(h *sqlc.StoryStatusHistory) *StatusHistoryResponse {
	if h == nil {
		return nil
	}

	resp := &StatusHistoryResponse{
		ID:          h.ID,
		StoryID:     h.StoryID,
		SetStatusBy: h.SetStatusBy,
		Content:     h.Content,
		Status:      h.Status,
	}

	if h.CreatedAt != nil {
		resp.CreatedAt = *h.CreatedAt
	}
	if h.UpdatedAt != nil {
		resp.UpdatedAt = *h.UpdatedAt
	}

	return resp
}

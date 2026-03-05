package story_vote

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

// CreateVoteRequest represents a request to create or update a vote
type CreateVoteRequest struct {
	StoryID uuid.UUID `json:"story_id" validate:"required,uuid"`
	Rating  int32     `json:"rating" validate:"required,min=1,max=5"`
}

// UpdateVoteRequest represents a request to update a vote
type UpdateVoteRequest struct {
	Rating *int32 `json:"rating" validate:"omitempty,min=1,max=5"`
}

// VoteResponse represents a single vote in API responses
type VoteResponse struct {
	ID        uuid.UUID `json:"id"`
	StoryID   uuid.UUID `json:"story_id"`
	UserID    uuid.UUID `json:"user_id"`
	Rating    int32     `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VoteStatsResponse represents vote statistics for a story
type VoteStatsResponse struct {
	StoryID       uuid.UUID `json:"story_id"`
	TotalVotes    int64     `json:"total_votes"`
	AverageRating float64   `json:"average_rating"`
}

// VoteListResponse represents a paginated list of votes
type VoteListResponse struct {
	Votes []VoteResponse            `json:"votes"`
	Meta  queryparam.PaginationMeta `json:"meta"`
}

// ToVoteResponse converts a sqlc.StoryVote to VoteResponse
func ToVoteResponse(v *sqlc.StoryVote) *VoteResponse {
	if v == nil {
		return nil
	}

	resp := &VoteResponse{
		ID:      v.ID,
		StoryID: v.StoryID,
		UserID:  v.UserID,
		Rating:  v.Rating,
	}

	if v.CreatedAt != nil {
		resp.CreatedAt = *v.CreatedAt
	}
	if v.UpdatedAt != nil {
		resp.UpdatedAt = *v.UpdatedAt
	}

	return resp
}

// ToVoteStatsResponse converts sqlc vote stats to response format
func ToVoteStatsResponse(storyID uuid.UUID, total int64, avg float32) *VoteStatsResponse {
	return &VoteStatsResponse{
		StoryID:       storyID,
		TotalVotes:    total,
		AverageRating: float64(avg),
	}
}

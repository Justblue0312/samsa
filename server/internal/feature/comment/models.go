package comment

import (
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type CreateCommentRequest struct {
	EntityType string     `json:"entity_type" validate:"required"`
	EntityID   uuid.UUID  `json:"entity_id" validate:"required"`
	ParentID   *uuid.UUID `json:"parent_id"`
	Content    string     `json:"content" validate:"required"`
	Source     string     `json:"source"`
}

type UpdateCommentRequest struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

type CommentResponse struct {
	sqlc.Comment    `json:"comment"`
	ReactionCount   int32                       `json:"reaction_count"`
	ReactionDetails map[sqlc.ReactionType]int64 `json:"reaction_details"`
	VoteCount       int32                       `json:"vote_count"`
	VoteDetails     map[sqlc.VoteType]int64     `json:"vote_details"`
}

type CommentListResponse struct {
	Comments []CommentResponse         `json:"comments"`
	Meta     queryparam.PaginationMeta `json:"meta"`
}

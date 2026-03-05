package comment

import (
	"github.com/google/uuid"
	"github.com/justblue/samsa/pkg/queryparam"
)

type CommentFilter struct {
	queryparam.PaginationParams

	ID         []uuid.UUID `query:"id"`
	EntityType string      `query:"entity_type" validate:"required"`
	EntityID   []uuid.UUID `query:"entity_id"`
	IsPinned   *bool       `query:"is_pinned"`
	IsReported *bool       `query:"is_reported"`
	IsArchived *bool       `query:"is_archived"`
	IsResolved *bool       `query:"is_resolved"`
	IsDeleted  *bool       `query:"is_deleted"`
}

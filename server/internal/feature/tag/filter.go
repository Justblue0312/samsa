package tag

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type TagFilter struct {
	*queryparam.PaginationParams

	OwnerID       *uuid.UUID      `query:"owner_id"`
	EntityID      *uuid.UUID      `query:"entity_id"`
	EntityType    sqlc.EntityType `query:"entity_type"`
	IsHidden      *bool           `query:"is_hidden"`
	IsSystem      *bool           `query:"is_system"`
	IsRecommended *bool           `query:"is_recommended"`
	SearchQuery   *string         `query:"search_query"`
}

func Validate(r *http.Request) (*TagFilter, error) {
	var filter TagFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.PaginationParams = &queryparam.PaginationParams{
		Page:    filter.Page,
		Limit:   filter.Limit,
		OrderBy: filter.OrderBy,
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.WithDefaultOrderBy("name_asc"),
		queryparam.AllowOrderWithSQLC([]string{
			"name_asc",
			"name_desc",
			"created_at_asc",
			"created_at_desc",
		}),
	)

	return &filter, nil
}

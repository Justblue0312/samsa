package author

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/pkg/queryparam"
)

type AuthorFilter struct {
	*queryparam.PaginationParams

	UserID        *uuid.UUID `json:"user_id" query:"user_id"`
	IsRecommended *bool      `json:"is_recommended" query:"is_recommended"`
	SearchQuery   *string    `json:"search_query" query:"search_query"`
}

func Validate(r *http.Request) (*AuthorFilter, error) {
	var filter AuthorFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.PaginationParams = &queryparam.PaginationParams{
		Page:  filter.Page,
		Limit: filter.Limit,
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.AllowOrderWith(map[string]string{
			"created_at": "created_at",
			"updated_at": "updated_at",
			"stage_name": "stage_name",
		}),
	)

	return &filter, nil
}

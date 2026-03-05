package notification

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type NotificationFilter struct {
	*queryparam.PaginationParams

	UserID uuid.UUID               `query:"user_id"`
	Level  *sqlc.NotificationLevel `query:"level"`
	IsRead *bool                   `query:"is_read"`
	Type   *string                 `query:"type"`
}

func Validate(r *http.Request) (*NotificationFilter, error) {
	var filter NotificationFilter
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
		queryparam.WithDefaultOrderBy("created_at:desc"),
		queryparam.AllowOrderWithSQLC([]string{
			"created_at_asc",
			"created_at_desc",
			"updated_at_asc",
			"updated_at_desc",
		}),
	)

	return &filter, nil
}

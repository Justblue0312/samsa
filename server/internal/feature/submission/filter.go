package submission

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type SubmissionFilter struct {
	queryparam.PaginationParams

	IDs          []uuid.UUID                  `query:"id"`
	RequesterIDs []uuid.UUID                  `query:"requester_id"`
	ApproverIDs  []uuid.UUID                  `query:"approver_id"`
	Title        string                       `query:"title" ops:"like,ilike"`
	Type         sqlc.SubmissionType          `query:"type" ops:"eq,in"`
	Status       sqlc.SubmissionStatus        `query:"status" ops:"eq,in"`
	ExposeID     string                       `query:"expose_id"`    // e.g. "SUB-0001"
	SearchQuery  *string                      `query:"search_query"` // FTS across title + message
	Tags         []string                     `query:"tags" ops:"in"`
	CreatedAt    queryparam.Filter[time.Time] `query:"created_at" ops:"gte,lte"`
}

func Validate(r *http.Request) (*SubmissionFilter, error) {
	var filter SubmissionFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.WithDefaultOrderBy("created_at:desc"),
		queryparam.AllowOrderWith(map[string]string{
			"created_at": "created_at",
			"updated_at": "updated_at",
			"title":      "title",
		}),
	)

	return &filter, nil
}

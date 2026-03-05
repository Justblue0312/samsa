package story_report

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

// ReportFilter defines filters for listing reports
type ReportFilter struct {
	queryparam.PaginationParams

	StoryID    *uuid.UUID         `query:"story_id"`
	ReporterID *uuid.UUID         `query:"reporter_id"`
	Status     *sqlc.ReportStatus `query:"status"`
	IsResolved *bool              `query:"is_resolved"`
}

// Validate implements queryparam.Validatable
func Validate(r *http.Request) (*ReportFilter, error) {
	var filter ReportFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.WithDefaultOrderBy("created_at:desc"),
		queryparam.AllowOrderWith(map[string]string{
			"status":      "status",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
			"resolved_at": "resolved_at",
		}),
	)

	return &filter, nil
}

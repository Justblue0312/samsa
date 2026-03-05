package story_vote

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/pkg/queryparam"
)

// VoteFilter defines filters for listing votes
type VoteFilter struct {
	queryparam.PaginationParams

	StoryID uuid.UUID `query:"story_id"`
	UserID  uuid.UUID `query:"user_id"`
}

// Validate implements queryparam.Validatable
func Validate(r *http.Request) (*VoteFilter, error) {
	var filter VoteFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.WithDefaultOrderBy("created_at:desc"),
		queryparam.AllowOrderWith(map[string]string{
			"rating":     "rating",
			"created_at": "created_at",
			"updated_at": "updated_at",
		}),
	)

	return &filter, nil
}

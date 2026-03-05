package file

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/justblue/samsa/pkg/queryparam"
)

type FileFilter struct {
	*queryparam.PaginationParams

	OwnerID         *uuid.UUID  `query:"owner_id"`
	FileIDs         []uuid.UUID `query:"file_ids"`
	IsArchived      *bool       `query:"is_archived"`
	IncludedDeleted bool        `query:"included_deleted"`
}

func (f *FileFilter) Normalize(opts ...queryparam.PaginationOption) {
	if f.PaginationParams == nil {
		f.PaginationParams = &queryparam.PaginationParams{}
	}
	f.PaginationParams.Normalize(opts...)
}

func Validate(r *http.Request) (*FileFilter, error) {
	var filter FileFilter
	if err := queryparam.DecodeRequest(&filter, r.URL.RawQuery); err != nil {
		return nil, fmt.Errorf("failed to decode query params: %w", err)
	}

	filter.Normalize(
		queryparam.WithDefaultLimit(20),
		queryparam.WithMaxLimit(100),
		queryparam.WithDefaultOrderBy("created_at_desc"),
		queryparam.AllowOrderWithSQLC([]string{
			"created_at_asc",
			"created_at_desc",
			"updated_at_asc",
			"updated_at_desc",
		}),
	)

	return &filter, nil
}

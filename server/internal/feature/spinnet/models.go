package spinnet

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

// SpinnetResponse represents a spinnet in API responses
type SpinnetResponse struct {
	ID          uuid.UUID  `json:"id"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
	Name        string     `json:"name"`
	Content     []byte     `json:"content,omitempty"`
	Category    *string    `json:"category,omitempty"`
	SmartSyntax *string    `json:"smart_syntax,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSpinnetRequest represents a request to create a spinnet
type CreateSpinnetRequest struct {
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
	Name        string     `json:"name" validate:"required,max=255"`
	Content     []byte     `json:"content,omitempty"`
	Category    *string    `json:"category,omitempty" validate:"omitempty,max=100"`
	SmartSyntax *string    `json:"smart_syntax,omitempty" validate:"omitempty,max=100"`
}

// UpdateSpinnetRequest represents a request to update a spinnet
type UpdateSpinnetRequest struct {
	Name        string  `json:"name" validate:"required,max=255"`
	Content     []byte  `json:"content,omitempty"`
	Category    *string `json:"category,omitempty" validate:"omitempty,max=100"`
	SmartSyntax *string `json:"smart_syntax,omitempty" validate:"omitempty,max=100"`
}

// ListSpinnetsParams represents parameters for listing spinnets
type ListSpinnetsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// SpinnetListResponse represents a paginated list of spinnets
type SpinnetListResponse struct {
	Spinnets []SpinnetResponse         `json:"spinnets"`
	Meta     queryparam.PaginationMeta `json:"meta"`
}

// ToSpinnetResponse converts a sqlc.Spinnet to SpinnetResponse
func ToSpinnetResponse(s *sqlc.Spinnet) *SpinnetResponse {
	if s == nil {
		return nil
	}

	resp := &SpinnetResponse{
		ID:          s.ID,
		OwnerID:     s.OwnerID,
		Name:        s.Name,
		Content:     s.Content,
		Category:    s.Category,
		SmartSyntax: s.SmartSyntax,
	}

	if s.CreatedAt != nil {
		resp.CreatedAt = *s.CreatedAt
	}
	if s.UpdatedAt != nil {
		resp.UpdatedAt = *s.UpdatedAt
	}

	return resp
}

// ToSpinnetListResponse converts a slice of sqlc.Spinnet to SpinnetListResponse
func ToSpinnetListResponse(spinnets []sqlc.Spinnet, totalCount int64, page, limit int32) *SpinnetListResponse {
	res := make([]SpinnetResponse, len(spinnets))
	for i, s := range spinnets {
		res[i] = *ToSpinnetResponse(&s)
	}

	meta := queryparam.NewPaginationMeta(page, limit, totalCount)

	return &SpinnetListResponse{
		Spinnets: res,
		Meta:     meta,
	}
}

package flag

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

// FlagType represents the type of content flag
type FlagType string

const (
	FlagTypeSpam           FlagType = "spam"
	FlagTypeInappropriate  FlagType = "inappropriate"
	FlagTypeCopyright      FlagType = "copyright"
	FlagTypePlagiarism     FlagType = "plagiarism"
	FlagTypeHarassment     FlagType = "harassment"
	FlagTypeHateSpeech     FlagType = "hate_speech"
	FlagTypeSelfHarm       FlagType = "self_harm"
	FlagTypeExplicit       FlagType = "explicit"
	FlagTypePrivacy        FlagType = "privacy"
	FlagTypeMisinformation FlagType = "misinformation"
	FlagTypeOther          FlagType = "other"
)

// FlagRate represents the severity rate of a flag
type FlagRate string

const (
	FlagRateLow      FlagRate = "low"
	FlagRateMedium   FlagRate = "medium"
	FlagRateHigh     FlagRate = "high"
	FlagRateCritical FlagRate = "critical"
)

// CreateFlagRequest represents a request to create a flag
type CreateFlagRequest struct {
	StoryID     uuid.UUID  `json:"story_id" validate:"required,uuid"`
	ChapterID   *uuid.UUID `json:"chapter_id,omitempty" validate:"omitempty,uuid"`
	Title       string     `json:"title" validate:"required,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
	FlagType    FlagType   `json:"flag_type" validate:"required,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
	FlagRate    FlagRate   `json:"flag_rate" validate:"required,oneof=low medium high critical"`
	FlagScore   float64    `json:"flag_score" validate:"required,min=0,max=100"`
}

// UpdateFlagRequest represents a request to update a flag
type UpdateFlagRequest struct {
	Title       *string   `json:"title,omitempty" validate:"omitempty,max=255"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
	FlagType    *FlagType `json:"flag_type,omitempty" validate:"omitempty,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
	FlagRate    *FlagRate `json:"flag_rate,omitempty" validate:"omitempty,oneof=low medium high critical"`
	FlagScore   *float64  `json:"flag_score,omitempty" validate:"omitempty,min=0,max=100"`
}

// ListFlagsParams represents parameters for listing flags
type ListFlagsParams struct {
	StoryID     *uuid.UUID `json:"story_id,omitempty"`
	ChapterID   *uuid.UUID `json:"chapter_id,omitempty"`
	InspectorID *uuid.UUID `json:"inspector_id,omitempty"`
	FlagType    *FlagType  `json:"flag_type,omitempty"`
	FlagRate    *FlagRate  `json:"flag_rate,omitempty"`
	MinScore    *float64   `json:"min_score,omitempty"`
	MaxScore    *float64   `json:"max_score,omitempty"`
	Limit       int32      `json:"limit"`
	Offset      int32      `json:"offset"`
}

// FlagResponse represents a flag in API responses
type FlagResponse struct {
	ID          uuid.UUID  `json:"id"`
	StoryID     uuid.UUID  `json:"story_id"`
	ChapterID   *uuid.UUID `json:"chapter_id,omitempty"`
	InspectorID *uuid.UUID `json:"inspector_id,omitempty"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	FlagType    FlagType   `json:"flag_type"`
	FlagRate    FlagRate   `json:"flag_rate"`
	FlagScore   float64    `json:"flag_score"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FlagListResponse represents a paginated list of flags
type FlagListResponse struct {
	Flags []FlagResponse            `json:"flags"`
	Meta  queryparam.PaginationMeta `json:"meta"`
}

// ToFlagResponse converts a sqlc.Flag to FlagResponse
func ToFlagResponse(f *sqlc.Flag) *FlagResponse {
	if f == nil {
		return nil
	}

	resp := &FlagResponse{
		ID:          f.ID,
		StoryID:     f.StoryID,
		ChapterID:   f.ChapterID,
		InspectorID: f.InspectorID,
		Title:       f.Title,
		Description: f.Description,
		FlagType:    FlagType(f.FlagType),
		FlagRate:    FlagRate(f.FlagRate),
		FlagScore:   f.FlagScore,
	}

	if f.CreatedAt != nil {
		resp.CreatedAt = *f.CreatedAt
	}
	if f.UpdatedAt != nil {
		resp.UpdatedAt = *f.UpdatedAt
	}

	return resp
}

// ToFlagListResponse converts a slice of sqlc.Flag to FlagListResponse
func ToFlagListResponse(flags []sqlc.Flag, totalCount int64, page, limit int32) *FlagListResponse {
	res := make([]FlagResponse, len(flags))
	for i, f := range flags {
		res[i] = *ToFlagResponse(&f)
	}

	meta := queryparam.NewPaginationMeta(page, limit, totalCount)

	return &FlagListResponse{
		Flags: res,
		Meta:  meta,
	}
}

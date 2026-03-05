package story_report

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

// ReportReason represents the reason for reporting a story
type ReportReason string

const (
	ReportReasonSpam           ReportReason = "spam"
	ReportReasonHarassment     ReportReason = "harassment"
	ReportReasonHateSpeech     ReportReason = "hate_speech"
	ReportReasonPlagiarism     ReportReason = "plagiarism"
	ReportReasonCopyright      ReportReason = "copyright"
	ReportReasonInappropriate  ReportReason = "inappropriate"
	ReportReasonMisinformation ReportReason = "misinformation"
	ReportReasonOther          ReportReason = "other"
)

// CreateReportRequest represents a request to create a report
type CreateReportRequest struct {
	StoryID     uuid.UUID    `json:"story_id" validate:"required,uuid"`
	ChapterID   *uuid.UUID   `json:"chapter_id,omitempty" validate:"omitempty,uuid"`
	Reason      ReportReason `json:"reason" validate:"required,oneof=spam harassment hate_speech plagiarism copyright inappropriate misinformation other"`
	Title       string       `json:"title" validate:"required,max=255"`
	Description *string      `json:"description" validate:"omitempty,max=2000"`
}

// UpdateReportRequest represents a request to update a report (moderator only)
type UpdateReportRequest struct {
	Status      *sqlc.ReportStatus `json:"status,omitempty" validate:"omitempty,oneof=pending resolved rejected archived"`
	Description *string            `json:"description,omitempty" validate:"omitempty,max=2000"`
}

// ResolveReportRequest represents a request to resolve a report
type ResolveReportRequest struct {
	ResolutionNotes *string `json:"resolution_notes,omitempty" validate:"omitempty,max=2000"`
}

// RejectReportRequest represents a request to reject a report
type RejectReportRequest struct {
	RejectionReason *string `json:"rejection_reason,omitempty" validate:"omitempty,max=2000"`
}

// ReportResponse represents a report in API responses
type ReportResponse struct {
	ID          uuid.UUID         `json:"id"`
	StoryID     uuid.UUID         `json:"story_id"`
	ChapterID   *uuid.UUID        `json:"chapter_id,omitempty"`
	ReporterID  uuid.UUID         `json:"reporter_id"`
	Reason      ReportReason      `json:"reason"`
	Title       string            `json:"title"`
	Description *string           `json:"description,omitempty"`
	Status      sqlc.ReportStatus `json:"status"`
	IsResolved  bool              `json:"is_resolved"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	ResolvedBy  *uuid.UUID        `json:"resolved_by,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ReportListResponse represents a paginated list of reports
type ReportListResponse struct {
	Reports []ReportResponse          `json:"reports"`
	Meta    queryparam.PaginationMeta `json:"meta"`
}

// ToReportResponse converts a sqlc.StoryReport to ReportResponse
func ToReportResponse(r *sqlc.StoryReport) *ReportResponse {
	if r == nil {
		return nil
	}

	resp := &ReportResponse{
		ID:          r.ID,
		StoryID:     r.StoryID,
		ChapterID:   r.ChapterID,
		ReporterID:  r.ReporterID,
		Title:       r.Title,
		Description: r.Description,
		IsResolved:  false,
	}

	if r.Status.Valid {
		resp.Status = r.Status.ReportStatus
	}

	if r.IsResolved != nil {
		resp.IsResolved = *r.IsResolved
	}

	if r.ResolvedAt != nil {
		resp.ResolvedAt = r.ResolvedAt
	}

	if r.ResolvedBy != nil {
		resp.ResolvedBy = r.ResolvedBy
	}

	if r.CreatedAt != nil {
		resp.CreatedAt = *r.CreatedAt
	}

	if r.UpdatedAt != nil {
		resp.UpdatedAt = *r.UpdatedAt
	}

	return resp
}

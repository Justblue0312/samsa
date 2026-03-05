package document

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

// DocumentStatus represents the status of a document in the approval workflow.
type DocumentStatus string

const (
	DocumentStatusDraft         DocumentStatus = "draft"
	DocumentStatusPendingReview DocumentStatus = "pending_review"
	DocumentStatusIsReviewed    DocumentStatus = "is_reviewed"
	DocumentStatusIsApproved    DocumentStatus = "is_approved"
	DocumentStatusRejected      DocumentStatus = "rejected"
	DocumentStatusArchived      DocumentStatus = "archived"
	DocumentStatusDeleted       DocumentStatus = "deleted"
)

// CreateDocumentRequest represents a request to create a new document.
type CreateDocumentRequest struct {
	StoryID      uuid.UUID  `json:"story_id" validate:"required"`
	FolderID     *uuid.UUID `json:"folder_id"`
	Language     string     `json:"language" validate:"required,len=3"`
	BranchName   string     `json:"branch_name" validate:"required,max=100"`
	Title        string     `json:"title" validate:"required,max=500"`
	Slug         string     `json:"slug" validate:"required,max=255"`
	Summary      *string    `json:"summary"`
	Content      []byte     `json:"content"`
	DocumentType *string    `json:"document_type"`
	IsTemplate   *bool      `json:"is_template"`
}

// UpdateDocumentRequest represents a request to update an existing document.
type UpdateDocumentRequest struct {
	FolderID     *uuid.UUID `json:"folder_id"`
	Language     *string    `json:"language" validate:"omitempty,len=3"`
	BranchName   *string    `json:"branch_name" validate:"omitempty,max=100"`
	Title        *string    `json:"title" validate:"omitempty,max=500"`
	Slug         *string    `json:"slug" validate:"omitempty,max=255"`
	Summary      *string    `json:"summary"`
	Content      []byte     `json:"content"`
	DocumentType *string    `json:"document_type"`
	IsLocked     *bool      `json:"is_locked"`
	IsTemplate   *bool      `json:"is_template"`
}

// DocumentResponse represents a document response.
type DocumentResponse struct {
	ID                uuid.UUID      `json:"id"`
	StoryID           uuid.UUID      `json:"story_id"`
	CreatedBy         uuid.UUID      `json:"created_by"`
	FolderID          *uuid.UUID     `json:"folder_id"`
	Language          string         `json:"language"`
	BranchName        string         `json:"branch_name"`
	VersionNumber     int32          `json:"version_number"`
	Title             *string        `json:"title"`
	Slug              *string        `json:"slug"`
	Summary           *string        `json:"summary"`
	DocumentType      *string        `json:"document_type"`
	Status            DocumentStatus `json:"status"`
	IsLocked          bool           `json:"is_locked"`
	IsTemplate        bool           `json:"is_template"`
	PreviousVersionID *uuid.UUID     `json:"previous_version_id"`
	TotalWords        int32          `json:"total_words"`
	TotalViews        int32          `json:"total_views"`
	TotalDownloads    int32          `json:"total_downloads"`
	TotalShares       int32          `json:"total_shares"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// ListDocumentsParams represents parameters for listing documents.
type ListDocumentsParams struct {
	StoryID  *uuid.UUID      `json:"story_id"`
	FolderID *uuid.UUID      `json:"folder_id"`
	Status   *DocumentStatus `json:"status"`
	Limit    int32           `json:"limit"`
	Offset   int32           `json:"offset"`
}

// SubmitForReviewRequest represents a request to submit a document for review.
type SubmitForReviewRequest struct{}

// ReviewDocumentRequest represents a request to review a document.
type ReviewDocumentRequest struct {
	Status   DocumentStatus `json:"status" validate:"required,oneof=is_approved rejected is_reviewed"`
	IsLocked *bool          `json:"is_locked"`
	Comments *string        `json:"comments"`
}

// ToDocumentResponse converts a sqlc.Document to a DocumentResponse.
func ToDocumentResponse(d *sqlc.Document) *DocumentResponse {
	if d == nil {
		return nil
	}

	res := &DocumentResponse{
		ID:                d.ID,
		StoryID:           d.StoryID,
		CreatedBy:         d.CreatedBy,
		FolderID:          d.FolderID,
		Language:          d.Language,
		BranchName:        d.BranchName,
		VersionNumber:     d.VersionNumber,
		Title:             d.Title,
		Slug:              d.Slug,
		Summary:           d.Summary,
		DocumentType:      d.DocumentType,
		Status:            DocumentStatus(d.Status),
		PreviousVersionID: d.PreviousVersionID,
	}

	if d.IsLocked != nil {
		res.IsLocked = *d.IsLocked
	}
	if d.IsTemplate != nil {
		res.IsTemplate = *d.IsTemplate
	}
	if d.TotalWords != nil {
		res.TotalWords = *d.TotalWords
	}
	if d.TotalViews != nil {
		res.TotalViews = *d.TotalViews
	}
	if d.TotalDownloads != nil {
		res.TotalDownloads = *d.TotalDownloads
	}
	if d.TotalShares != nil {
		res.TotalShares = *d.TotalShares
	}
	if d.CreatedAt != nil {
		res.CreatedAt = *d.CreatedAt
	}
	if d.UpdatedAt != nil {
		res.UpdatedAt = *d.UpdatedAt
	}

	return res
}

// ToDocumentListResponse converts a slice of sqlc.Document to a slice of DocumentResponse.
func ToDocumentListResponse(documents []sqlc.Document) []DocumentResponse {
	res := make([]DocumentResponse, len(documents))
	for i, d := range documents {
		res[i] = *ToDocumentResponse(&d)
	}
	return res
}

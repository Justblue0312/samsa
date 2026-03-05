package document_folder

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

// CreateDocumentFolderRequest represents a request to create a new document folder.
type CreateDocumentFolderRequest struct {
	StoryID  uuid.UUID  `json:"story_id" validate:"required"`
	OwnerID  uuid.UUID  `json:"owner_id" validate:"required"`
	Name     string     `json:"name" validate:"required,max=255"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// UpdateDocumentFolderRequest represents a request to update an existing document folder.
type UpdateDocumentFolderRequest struct {
	Name     *string    `json:"name" validate:"omitempty,max=255"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// DocumentFolderResponse represents a document folder response.
type DocumentFolderResponse struct {
	ID              uuid.UUID  `json:"id"`
	StoryID         uuid.UUID  `json:"story_id"`
	OwnerID         uuid.UUID  `json:"owner_id"`
	Name            string     `json:"name"`
	ParentID        *uuid.UUID `json:"parent_id"`
	Depth           int32      `json:"depth"`
	DocumentsCount  int32      `json:"documents_count"`
	SubfoldersCount int32      `json:"subfolders_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ListDocumentFoldersParams represents parameters for listing document folders.
type ListDocumentFoldersParams struct {
	StoryID  *uuid.UUID `json:"story_id"`
	ParentID *uuid.UUID `json:"parent_id"`
	OwnerID  *uuid.UUID `json:"owner_id"`
	Limit    int32      `json:"limit"`
	Offset   int32      `json:"offset"`
}

// MoveDocumentFolderRequest represents a request to move a folder.
type MoveDocumentFolderRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
}

// ToDocumentFolderResponse converts a sqlc.DocumentFolder to a DocumentFolderResponse.
func ToDocumentFolderResponse(f *sqlc.DocumentFolder) *DocumentFolderResponse {
	if f == nil {
		return nil
	}

	res := &DocumentFolderResponse{
		ID:       f.ID,
		StoryID:  f.StoryID,
		OwnerID:  f.OwnerID,
		Name:     f.Name,
		ParentID: f.ParentID,
		Depth:    f.Depth,
	}

	if f.CreatedAt != nil {
		res.CreatedAt = *f.CreatedAt
	}
	if f.UpdatedAt != nil {
		res.UpdatedAt = *f.UpdatedAt
	}

	return res
}

// ToDocumentFolderListResponse converts a slice of sqlc.DocumentFolder to a slice of DocumentFolderResponse.
func ToDocumentFolderListResponse(folders []sqlc.DocumentFolder) []DocumentFolderResponse {
	res := make([]DocumentFolderResponse, len(folders))
	for i, f := range folders {
		res[i] = *ToDocumentFolderResponse(&f)
	}
	return res
}

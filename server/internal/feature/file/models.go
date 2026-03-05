package file

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

type FileResponse struct {
	ID         uuid.UUID             `json:"id"`
	OwnerID    uuid.UUID             `json:"owner_id"`
	Name       string                `json:"name"`
	Path       string                `json:"path"`
	MimeType   *string               `json:"mime_type"`
	Size       int64                 `json:"size"`
	Reference  string                `json:"reference"`
	Payload    string                `json:"payload,omitempty"`
	Service    *string               `json:"service"`
	Source     sqlc.FileUploadSource `json:"source"`
	IsArchived bool                  `json:"is_archived"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

func ToResponse(f *sqlc.File) *FileResponse {
	if f == nil {
		return nil
	}
	return &FileResponse{
		ID:         f.ID,
		OwnerID:    f.OwnerID,
		Name:       f.Name,
		Path:       f.Path,
		MimeType:   f.MimeType,
		Size:       f.Size,
		Reference:  f.Reference,
		Payload:    f.Payload,
		Service:    f.Service,
		Source:     f.Source,
		IsArchived: f.IsArchived,
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}
}

type CreateRequest struct {
	Name    string                `json:"name" validate:"required"`
	Path    string                `json:"path" validate:"required"`
	Service *string               `json:"service"`
	Source  sqlc.FileUploadSource `json:"source" validate:"required"`
}

type UpdateRequest struct {
	Name       *string `json:"name"`
	IsArchived *bool   `json:"is_archived"`
}

type ListResponse struct {
	Files     []*FileResponse `json:"files"`
	Total     int64           `json:"total"`
	Page      int             `json:"page"`
	Limit     int             `json:"limit"`
	TotalPage int             `json:"total_page"`
}

type DownloadURLResponse struct {
	URL         string    `json:"url"`
	ExpiresAt   time.Time `json:"expires_at"`
	ContentType string    `json:"content_type"`
}

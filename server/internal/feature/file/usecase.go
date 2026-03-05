package file

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	s3infra "github.com/justblue/samsa/internal/infras/aws/s3"
)

var (
	ErrNotFound      = errors.New("file not found")
	ErrAlreadyExists = errors.New("file already exists")
)

const (
	DefaultPayloadSizeThreshold int64 = 1024 * 1024 // 1MB
)

type UseCase interface {
	Create(ctx context.Context, ownerID uuid.UUID, req *CreateRequest, content io.Reader) (*FileResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*FileResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateRequest) (*FileResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *FileFilter) (*ListResponse, error)
	GetDownloadURL(ctx context.Context, id uuid.UUID) (*DownloadURLResponse, error)
	// File sharing
	ShareFile(ctx context.Context, id uuid.UUID) (*FileResponse, error)
	UnshareFile(ctx context.Context, id uuid.UUID) (*FileResponse, error)
	ListSharedFiles(ctx context.Context, limit, offset int32) (*ListResponse, error)
	// File validation
	GetFilesByOwnerAndType(ctx context.Context, ownerID uuid.UUID, mimeType string, limit, offset int32) (*ListResponse, error)
	GetFilesByMimeType(ctx context.Context, mimeType string, limit, offset int32) (*ListResponse, error)
	CountFilesByMimeType(ctx context.Context, mimeType string) (int64, error)
	GetTotalSizeByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
	// Soft delete
	SoftDeleteFile(ctx context.Context, id uuid.UUID) (*FileResponse, error)
	RestoreFile(ctx context.Context, id uuid.UUID) (*FileResponse, error)
	// Filtered list
	ListFilesWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool, limit, offset int32) (*ListResponse, error)
}

type usecase struct {
	repo     FileRepository
	s3Client *s3infra.Client
	cfg      *config.Config
}

func NewUseCase(repo FileRepository, s3Client *s3infra.Client, cfg *config.Config) UseCase {
	return &usecase{
		repo:     repo,
		s3Client: s3Client,
		cfg:      cfg,
	}
}

func (u *usecase) Create(ctx context.Context, ownerID uuid.UUID, req *CreateRequest, content io.Reader) (*FileResponse, error) {
	existing, err := u.repo.GetByPath(ctx, req.Path)
	if err == nil && existing != nil {
		return nil, ErrAlreadyExists
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	fileSize := int64(len(data))
	threshold := DefaultPayloadSizeThreshold
	if u.cfg.File.PayloadSizeThreshold > 0 {
		threshold = u.cfg.File.PayloadSizeThreshold
	}

	var payload, reference string
	var source = req.Source

	if fileSize > threshold && u.s3Client != nil && u.s3Client.IsConfigured() {
		key := fmt.Sprintf("files/%s/%s", ownerID.String(), uuid.New().String())
		_, err := u.s3Client.UploadObject(ctx, key, getMimeType(req.Name), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to upload to S3: %w", err)
		}
		reference = key
		source = sqlc.FileUploadSourcePresigned
	} else {
		payload = base64.StdEncoding.EncodeToString(data)
		source = sqlc.FileUploadSourceBase64
	}

	now := time.Now().UTC()
	file := &sqlc.File{
		ID:         uuid.New(),
		OwnerID:    ownerID,
		Name:       req.Name,
		Path:       req.Path,
		MimeType:   ptr(getMimeType(req.Name)),
		Size:       fileSize,
		Reference:  reference,
		Payload:    payload,
		Service:    req.Service,
		Source:     source,
		IsArchived: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	created, err := u.repo.Create(ctx, file)
	if err != nil {
		return nil, err
	}

	return ToResponse(created), nil
}

func (u *usecase) GetByID(ctx context.Context, id uuid.UUID) (*FileResponse, error) {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToResponse(file), nil
}

func (u *usecase) Update(ctx context.Context, id uuid.UUID, req *UpdateRequest) (*FileResponse, error) {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		file.Name = *req.Name
	}
	if req.IsArchived != nil {
		file.IsArchived = *req.IsArchived
	}
	file.UpdatedAt = time.Now().UTC()

	updated, err := u.repo.Update(ctx, file)
	if err != nil {
		return nil, err
	}

	return ToResponse(updated), nil
}

func (u *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if file.Reference != "" && u.s3Client != nil && u.s3Client.IsConfigured() {
		if err := u.s3Client.DeleteObject(ctx, file.Reference); err != nil {
			return fmt.Errorf("failed to delete S3 object: %w", err)
		}
	}

	return u.repo.Delete(ctx, id)
}

func (u *usecase) List(ctx context.Context, filter *FileFilter) (*ListResponse, error) {
	files, total, err := u.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToResponse(f)
	}

	limit := int(filter.GetLimit())
	page := int(filter.PaginationParams.Page)
	totalPage := int(total) / limit
	if int(total)%limit > 0 {
		totalPage++
	}

	return &ListResponse{
		Files:     responses,
		Total:     total,
		Page:      page,
		Limit:     limit,
		TotalPage: totalPage,
	}, nil
}

func (u *usecase) GetDownloadURL(ctx context.Context, id uuid.UUID) (*DownloadURLResponse, error) {
	file, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if file.Reference == "" {
		if file.Payload == "" {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("file stored in database, cannot generate download URL")
	}

	if u.s3Client == nil || !u.s3Client.IsConfigured() {
		return nil, fmt.Errorf("S3 client not configured")
	}

	presignURL, err := u.s3Client.PresignGetObject(ctx, file.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	ttl := u.cfg.AWS.PresignTTL
	if ttl == 0 {
		ttl = 15 * time.Minute
	}

	return &DownloadURLResponse{
		URL:         presignURL,
		ExpiresAt:   time.Now().Add(ttl),
		ContentType: getMimeType(file.Name),
	}, nil
}

func getMimeType(filename string) string {
	ext := ""
	if i := len(filename) - 1; i >= 0 {
		for ; i >= 0; i-- {
			if filename[i] == '.' {
				ext = filename[i:]
				break
			}
		}
	}

	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".zip":  "application/zip",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".txt":  "text/plain",
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func ptr[T any](v T) *T {
	return &v
}

// ShareFile implements [UseCase].
func (u *usecase) ShareFile(ctx context.Context, id uuid.UUID) (*FileResponse, error) {
	file, err := u.repo.Share(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToResponse(file), nil
}

// UnshareFile implements [UseCase].
func (u *usecase) UnshareFile(ctx context.Context, id uuid.UUID) (*FileResponse, error) {
	file, err := u.repo.Unshare(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToResponse(file), nil
}

// ListSharedFiles implements [UseCase].
func (u *usecase) ListSharedFiles(ctx context.Context, limit, offset int32) (*ListResponse, error) {
	files, err := u.repo.GetSharedFiles(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToResponse(&f)
	}

	return &ListResponse{
		Files:     responses,
		Total:     int64(len(files)),
		Page:      1,
		Limit:     int(limit),
		TotalPage: 1,
	}, nil
}

// GetFilesByOwnerAndType implements [UseCase].
func (u *usecase) GetFilesByOwnerAndType(ctx context.Context, ownerID uuid.UUID, mimeType string, limit, offset int32) (*ListResponse, error) {
	files, err := u.repo.GetByOwnerAndType(ctx, ownerID, mimeType, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToResponse(&f)
	}

	return &ListResponse{
		Files:     responses,
		Total:     int64(len(files)),
		Page:      1,
		Limit:     int(limit),
		TotalPage: 1,
	}, nil
}

// GetFilesByMimeType implements [UseCase].
func (u *usecase) GetFilesByMimeType(ctx context.Context, mimeType string, limit, offset int32) (*ListResponse, error) {
	files, err := u.repo.GetByMimeType(ctx, mimeType, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToResponse(&f)
	}

	return &ListResponse{
		Files:     responses,
		Total:     int64(len(files)),
		Page:      1,
		Limit:     int(limit),
		TotalPage: 1,
	}, nil
}

// CountFilesByMimeType implements [UseCase].
func (u *usecase) CountFilesByMimeType(ctx context.Context, mimeType string) (int64, error) {
	return u.repo.CountByMimeType(ctx, mimeType)
}

// GetTotalSizeByOwner implements [UseCase].
func (u *usecase) GetTotalSizeByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return u.repo.GetTotalSizeByOwner(ctx, ownerID)
}

// SoftDeleteFile implements [UseCase].
func (u *usecase) SoftDeleteFile(ctx context.Context, id uuid.UUID) (*FileResponse, error) {
	file, err := u.repo.SoftDelete(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToResponse(file), nil
}

// RestoreFile implements [UseCase].
func (u *usecase) RestoreFile(ctx context.Context, id uuid.UUID) (*FileResponse, error) {
	file, err := u.repo.Restore(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToResponse(file), nil
}

// ListFilesWithFilters implements [UseCase].
func (u *usecase) ListFilesWithFilters(ctx context.Context, ownerID, mimeType, reference *string, isArchived *bool, limit, offset int32) (*ListResponse, error) {
	files, err := u.repo.ListWithFilters(ctx, ownerID, mimeType, reference, isArchived, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*FileResponse, len(files))
	for i, f := range files {
		responses[i] = ToResponse(&f)
	}

	totalCount, err := u.repo.CountWithFilters(ctx, ownerID, mimeType, reference, isArchived)
	if err != nil {
		return nil, err
	}

	limitInt := int(limit)
	page := 1
	totalPage := int(totalCount) / limitInt
	if int(totalCount)%limitInt > 0 {
		totalPage++
	}

	return &ListResponse{
		Files:     responses,
		Total:     totalCount,
		Page:      page,
		Limit:     limitInt,
		TotalPage: totalPage,
	}, nil
}

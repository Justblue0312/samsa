package document

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Basic CRUD
	CreateDocument(ctx context.Context, userID uuid.UUID, req CreateDocumentRequest) (*DocumentResponse, error)
	GetDocument(ctx context.Context, id uuid.UUID) (*DocumentResponse, error)
	GetDocumentBySlug(ctx context.Context, slug string) (*DocumentResponse, error)
	ListDocuments(ctx context.Context, params ListDocumentsParams) ([]DocumentResponse, error)
	UpdateDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, req UpdateDocumentRequest) (*DocumentResponse, error)
	DeleteDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) error

	// Workflow
	SubmitForReview(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*DocumentResponse, error)
	ApproveDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, comments *string) (*DocumentResponse, error)
	RejectDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, comments *string) (*DocumentResponse, error)
	ArchiveDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*DocumentResponse, error)
	ReviewDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, req ReviewDocumentRequest) (*DocumentResponse, error)

	// Stats
	IncrementView(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error)
	IncrementDownload(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error)
	IncrementShare(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error)

	// Versioning
	CreateNewVersion(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, content []byte) (*DocumentResponse, error)
	GetVersionHistory(ctx context.Context, documentID uuid.UUID) ([]DocumentResponse, error)

	// Status History
	GetStatusHistory(ctx context.Context, documentID uuid.UUID) ([]sqlc.DocumentStatusHistory, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateDocument(ctx context.Context, userID uuid.UUID, req CreateDocumentRequest) (*DocumentResponse, error) {
	status := sqlc.DocumentStatusDraft
	now := time.Now()

	arg := sqlc.CreateDocumentParams{
		StoryID:       req.StoryID,
		CreatedBy:     userID,
		FolderID:      req.FolderID,
		Language:      req.Language,
		BranchName:    req.BranchName,
		VersionNumber: 1,
		Content:       req.Content,
		Title:         &req.Title,
		Slug:          &req.Slug,
		Summary:       req.Summary,
		DocumentType:  req.DocumentType,
		Status:        status,
		IsTemplate:    req.IsTemplate,
		TotalWords:    ptr(int32(0)),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	if req.IsTemplate == nil {
		arg.IsTemplate = ptr(false)
	}

	doc, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Create initial status history
	_, _ = uc.repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
		DocumentID:  doc.ID,
		SetStatusBy: userID,
		Content:     "Document created",
		Status:      status,
		CreatedAt:   &now,
	})

	return ToDocumentResponse(doc), nil
}

func (uc *usecase) GetDocument(ctx context.Context, id uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToDocumentResponse(doc), nil
}

func (uc *usecase) GetDocumentBySlug(ctx context.Context, slug string) (*DocumentResponse, error) {
	doc, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return ToDocumentResponse(doc), nil
}

func (uc *usecase) ListDocuments(ctx context.Context, params ListDocumentsParams) ([]DocumentResponse, error) {
	var documents []sqlc.Document
	var err error

	limit := params.Limit
	if limit == 0 {
		limit = 20
	}

	if params.StoryID != nil {
		documents, err = uc.repo.GetDocumentsByStoryID(ctx, sqlc.GetDocumentsByStoryIDParams{
			StoryID: *params.StoryID,
			Limit:   limit,
			Offset:  params.Offset,
		})
	} else if params.FolderID != nil {
		documents, err = uc.repo.GetDocumentsByFolderID(ctx, sqlc.GetDocumentsByFolderIDParams{
			FolderID: params.FolderID,
			Limit:    limit,
			Offset:   params.Offset,
		})
	} else if params.Status != nil {
		documents, err = uc.repo.GetDocumentsByStatus(ctx, sqlc.GetDocumentsByStatusParams{
			Status: sqlc.DocumentStatus(*params.Status),
			Limit:  limit,
			Offset: params.Offset,
		})
	} else {
		documents, err = uc.repo.GetDocumentsByOwnerID(ctx, sqlc.GetDocumentsByOwnerIDParams{
			Limit:  limit,
			Offset: params.Offset,
		})
	}

	if err != nil {
		return nil, err
	}

	return ToDocumentListResponse(documents), nil
}

func (uc *usecase) UpdateDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, req UpdateDocumentRequest) (*DocumentResponse, error) {
	existing, err := uc.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if existing.CreatedBy != userID {
		return nil, ErrPermissionDenied
	}

	arg := sqlc.UpdateDocumentParams{
		ID:                documentID,
		FolderID:          existing.FolderID,
		Language:          existing.Language,
		BranchName:        existing.BranchName,
		VersionNumber:     existing.VersionNumber,
		Content:           existing.Content,
		Title:             existing.Title,
		Slug:              existing.Slug,
		Summary:           existing.Summary,
		DocumentType:      existing.DocumentType,
		Status:            existing.Status,
		IsLocked:          existing.IsLocked,
		IsTemplate:        existing.IsTemplate,
		PreviousVersionID: existing.PreviousVersionID,
		TotalWords:        existing.TotalWords,
		TotalViews:        existing.TotalViews,
		TotalDownloads:    existing.TotalDownloads,
		TotalShares:       existing.TotalShares,
	}

	// Apply updates
	if req.FolderID != nil {
		arg.FolderID = req.FolderID
	}
	if req.Language != nil {
		arg.Language = *req.Language
	}
	if req.BranchName != nil {
		arg.BranchName = *req.BranchName
	}
	if req.Title != nil {
		arg.Title = req.Title
	}
	if req.Slug != nil {
		arg.Slug = req.Slug
	}
	if req.Summary != nil {
		arg.Summary = req.Summary
	}
	if req.Content != nil {
		arg.Content = req.Content
	}
	if req.DocumentType != nil {
		arg.DocumentType = req.DocumentType
	}
	if req.IsLocked != nil {
		arg.IsLocked = req.IsLocked
	}
	if req.IsTemplate != nil {
		arg.IsTemplate = req.IsTemplate
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) DeleteDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) error {
	doc, err := uc.repo.GetByID(ctx, documentID)
	if err != nil {
		return err
	}

	if doc.CreatedBy != userID {
		return ErrPermissionDenied
	}

	_, err = uc.repo.SoftDelete(ctx, documentID)
	return err
}

func (uc *usecase) SubmitForReview(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if doc.CreatedBy != userID {
		return nil, ErrPermissionDenied
	}

	updated, err := uc.repo.SubmitForReview(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Record status change
	now := time.Now()
	_, _ = uc.repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
		DocumentID:  documentID,
		SetStatusBy: userID,
		Content:     "Submitted for review",
		Status:      updated.Status,
		CreatedAt:   &now,
	})

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) ApproveDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, comments *string) (*DocumentResponse, error) {
	// TODO: Verify user has approval permissions
	_ = userID

	updated, err := uc.repo.Approve(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Record status change
	now := time.Now()
	content := "Document approved"
	if comments != nil {
		content = *comments
	}
	_, _ = uc.repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
		DocumentID:  documentID,
		SetStatusBy: userID,
		Content:     content,
		Status:      updated.Status,
		CreatedAt:   &now,
	})

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) RejectDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, comments *string) (*DocumentResponse, error) {
	// TODO: Verify user has approval permissions
	_ = userID

	updated, err := uc.repo.Reject(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Record status change
	now := time.Now()
	content := "Document rejected"
	if comments != nil {
		content = *comments
	}
	_, _ = uc.repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
		DocumentID:  documentID,
		SetStatusBy: userID,
		Content:     content,
		Status:      updated.Status,
		CreatedAt:   &now,
	})

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) ArchiveDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if doc.CreatedBy != userID {
		return nil, ErrPermissionDenied
	}

	updated, err := uc.repo.Archive(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) ReviewDocument(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, req ReviewDocumentRequest) (*DocumentResponse, error) {
	// TODO: Verify user has review permissions
	_ = userID

	isLocked := false
	if req.IsLocked != nil {
		isLocked = *req.IsLocked
	}

	updated, err := uc.repo.Review(ctx, documentID, sqlc.DocumentStatus(req.Status), isLocked)
	if err != nil {
		return nil, err
	}

	// Record status change
	now := time.Now()
	content := "Document reviewed"
	if req.Comments != nil {
		content = *req.Comments
	}
	_, _ = uc.repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
		DocumentID:  documentID,
		SetStatusBy: userID,
		Content:     content,
		Status:      updated.Status,
		CreatedAt:   &now,
	})

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) IncrementView(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.IncrementViews(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(doc), nil
}

func (uc *usecase) IncrementDownload(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.IncrementDownloads(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(doc), nil
}

func (uc *usecase) IncrementShare(ctx context.Context, documentID uuid.UUID) (*DocumentResponse, error) {
	doc, err := uc.repo.IncrementShares(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(doc), nil
}

func (uc *usecase) CreateNewVersion(ctx context.Context, userID uuid.UUID, documentID uuid.UUID, content []byte) (*DocumentResponse, error) {
	existing, err := uc.repo.GetByID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if existing.CreatedBy != userID {
		return nil, ErrPermissionDenied
	}

	// Calculate word count (simple implementation)
	wordCount := int32(len(string(content))) / 5 // Approximate

	newVersion := existing.VersionNumber + 1
	previousVersionID := existing.ID

	updated, err := uc.repo.UpdateVersion(ctx, documentID, newVersion, &previousVersionID, content, wordCount)
	if err != nil {
		return nil, err
	}

	return ToDocumentResponse(updated), nil
}

func (uc *usecase) GetVersionHistory(ctx context.Context, documentID uuid.UUID) ([]DocumentResponse, error) {
	versions, err := uc.repo.GetVersionHistory(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return ToDocumentListResponse(versions), nil
}

func (uc *usecase) GetStatusHistory(ctx context.Context, documentID uuid.UUID) ([]sqlc.DocumentStatusHistory, error) {
	return uc.repo.ListStatusHistory(ctx, documentID)
}

// Helper function
func ptr[T any](v T) *T {
	return &v
}

// Errors
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrDocumentNotFound = errors.New("document not found")
)

package document

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Basic CRUD
	Create(ctx context.Context, arg sqlc.CreateDocumentParams) (*sqlc.Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	GetBySlug(ctx context.Context, slug string) (*sqlc.Document, error)
	GetDocumentsByOwnerID(ctx context.Context, arg sqlc.GetDocumentsByOwnerIDParams) ([]sqlc.Document, error)
	GetDocumentsByFolderID(ctx context.Context, arg sqlc.GetDocumentsByFolderIDParams) ([]sqlc.Document, error)
	GetDocumentsByStatus(ctx context.Context, arg sqlc.GetDocumentsByStatusParams) ([]sqlc.Document, error)
	GetDocumentsByStoryID(ctx context.Context, arg sqlc.GetDocumentsByStoryIDParams) ([]sqlc.Document, error)
	Update(ctx context.Context, arg sqlc.UpdateDocumentParams) (*sqlc.Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)

	// Workflow
	SubmitForReview(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	Approve(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	Reject(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	Archive(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	Review(ctx context.Context, id uuid.UUID, status sqlc.DocumentStatus, isLocked bool) (*sqlc.Document, error)

	// Stats
	IncrementViews(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	IncrementDownloads(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)
	IncrementShares(ctx context.Context, id uuid.UUID) (*sqlc.Document, error)

	// Versioning
	UpdateVersion(ctx context.Context, id uuid.UUID, version int32, previousVersionID *uuid.UUID, content []byte, words int32) (*sqlc.Document, error)
	GetVersionHistory(ctx context.Context, id uuid.UUID) ([]sqlc.Document, error)

	// Status History
	CreateStatusHistory(ctx context.Context, arg sqlc.CreateDocumentStatusHistoryParams) (*sqlc.DocumentStatusHistory, error)
	ListStatusHistory(ctx context.Context, documentID uuid.UUID) ([]sqlc.DocumentStatusHistory, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateDocumentParams) (*sqlc.Document, error) {
	d, err := r.q.CreateDocument(ctx, arg)
	return &d, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.GetDocumentByID(ctx, id)
	return &d, err
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*sqlc.Document, error) {
	d, err := r.q.GetDocumentBySlug(ctx, &slug)
	return &d, err
}

func (r *repository) GetDocumentsByOwnerID(ctx context.Context, arg sqlc.GetDocumentsByOwnerIDParams) ([]sqlc.Document, error) {
	return r.q.GetDocumentsByOwnerID(ctx, arg)
}

func (r *repository) GetDocumentsByFolderID(ctx context.Context, arg sqlc.GetDocumentsByFolderIDParams) ([]sqlc.Document, error) {
	return r.q.GetDocumentsByFolderID(ctx, arg)
}

func (r *repository) GetDocumentsByStatus(ctx context.Context, arg sqlc.GetDocumentsByStatusParams) ([]sqlc.Document, error) {
	return r.q.GetDocumentsByStatus(ctx, arg)
}

func (r *repository) GetDocumentsByStoryID(ctx context.Context, arg sqlc.GetDocumentsByStoryIDParams) ([]sqlc.Document, error) {
	return r.q.GetDocumentsByStoryID(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateDocumentParams) (*sqlc.Document, error) {
	d, err := r.q.UpdateDocument(ctx, arg)
	return &d, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteDocument(ctx, id)
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.SoftDeleteDocument(ctx, id)
	return &d, err
}

func (r *repository) SubmitForReview(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.SubmitDocumentForReview(ctx, id)
	return &d, err
}

func (r *repository) Approve(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.ApproveDocument(ctx, id)
	return &d, err
}

func (r *repository) Reject(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.RejectDocument(ctx, id)
	return &d, err
}

func (r *repository) Archive(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.ArchiveDocument(ctx, id)
	return &d, err
}

func (r *repository) Review(ctx context.Context, id uuid.UUID, status sqlc.DocumentStatus, isLocked bool) (*sqlc.Document, error) {
	d, err := r.q.ReviewDocument(ctx, sqlc.ReviewDocumentParams{
		ID:       id,
		Status:   status,
		IsLocked: &isLocked,
	})
	return &d, err
}

func (r *repository) IncrementViews(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.IncrementDocumentViews(ctx, id)
	return &d, err
}

func (r *repository) IncrementDownloads(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.IncrementDocumentDownloads(ctx, id)
	return &d, err
}

func (r *repository) IncrementShares(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	d, err := r.q.IncrementDocumentShares(ctx, id)
	return &d, err
}

func (r *repository) UpdateVersion(ctx context.Context, id uuid.UUID, version int32, previousVersionID *uuid.UUID, content []byte, words int32) (*sqlc.Document, error) {
	d, err := r.q.UpdateDocumentVersion(ctx, sqlc.UpdateDocumentVersionParams{
		ID:                id,
		VersionNumber:     version,
		PreviousVersionID: previousVersionID,
		Content:           content,
		TotalWords:        &words,
	})
	return &d, err
}

func (r *repository) GetVersionHistory(ctx context.Context, id uuid.UUID) ([]sqlc.Document, error) {
	return r.q.GetDocumentVersionHistory(ctx, id)
}

func (r *repository) CreateStatusHistory(ctx context.Context, arg sqlc.CreateDocumentStatusHistoryParams) (*sqlc.DocumentStatusHistory, error) {
	h, err := r.q.CreateDocumentStatusHistory(ctx, arg)
	return &h, err
}

func (r *repository) ListStatusHistory(ctx context.Context, documentID uuid.UUID) ([]sqlc.DocumentStatusHistory, error) {
	return r.q.ListDocumentStatusHistory(ctx, documentID)
}

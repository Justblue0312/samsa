package document_folder

import (
	"context"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	// Basic CRUD
	Create(ctx context.Context, arg sqlc.CreateDocumentFolderParams) (*sqlc.DocumentFolder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.DocumentFolder, error)
	GetFoldersByParentID(ctx context.Context, parentID uuid.UUID) ([]sqlc.DocumentFolder, error)
	GetRootFolders(ctx context.Context) ([]sqlc.DocumentFolder, error)
	GetFoldersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.DocumentFolder, error)
	GetFoldersByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]sqlc.DocumentFolder, error)
	Update(ctx context.Context, arg sqlc.UpdateDocumentFolderParams) (*sqlc.DocumentFolder, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.DocumentFolder, error)

	// Hierarchy operations
	Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, depth int32) (*sqlc.DocumentFolder, error)
	ValidateDepth(ctx context.Context, parentID *uuid.UUID) (int32, error)
	GetChildCount(ctx context.Context, parentID uuid.UUID) (int32, error)
	GetDocumentsCount(ctx context.Context, folderID uuid.UUID) (int32, error)

	// Tree operations
	GetAncestors(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error)
	GetDescendants(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error)
	GetFolderTree(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error)
	GetSiblings(ctx context.Context, parentID uuid.UUID, excludeID uuid.UUID) ([]sqlc.DocumentFolder, error)

	// Search
	Search(ctx context.Context, query string, limit, offset int32) ([]sqlc.DocumentFolder, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q: sqlc.New(db),
	}
}

func (r *repository) Create(ctx context.Context, arg sqlc.CreateDocumentFolderParams) (*sqlc.DocumentFolder, error) {
	f, err := r.q.CreateDocumentFolder(ctx, arg)
	return &f, err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.DocumentFolder, error) {
	f, err := r.q.GetDocumentFolderByID(ctx, id)
	return &f, err
}

func (r *repository) GetFoldersByParentID(ctx context.Context, parentID uuid.UUID) ([]sqlc.DocumentFolder, error) {
	return r.q.GetDocumentFoldersByParentID(ctx, &parentID)
}

func (r *repository) GetRootFolders(ctx context.Context) ([]sqlc.DocumentFolder, error) {
	return r.q.GetRootDocumentFolders(ctx)
}

func (r *repository) GetFoldersByStoryID(ctx context.Context, storyID uuid.UUID) ([]sqlc.DocumentFolder, error) {
	return r.q.GetDocumentFoldersByStoryID(ctx, storyID)
}

func (r *repository) GetFoldersByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]sqlc.DocumentFolder, error) {
	return r.q.GetDocumentFoldersByOwnerID(ctx, ownerID)
}

func (r *repository) Update(ctx context.Context, arg sqlc.UpdateDocumentFolderParams) (*sqlc.DocumentFolder, error) {
	f, err := r.q.UpdateDocumentFolder(ctx, arg)
	return &f, err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteDocumentFolder(ctx, id)
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.DocumentFolder, error) {
	f, err := r.q.SoftDeleteDocumentFolder(ctx, id)
	return &f, err
}

func (r *repository) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, depth int32) (*sqlc.DocumentFolder, error) {
	f, err := r.q.MoveDocumentFolder(ctx, sqlc.MoveDocumentFolderParams{
		ID:       id,
		ParentID: parentID,
		Depth:    depth,
	})
	return &f, err
}

func (r *repository) ValidateDepth(ctx context.Context, parentID *uuid.UUID) (int32, error) {
	if parentID == nil {
		return 0, nil
	}
	result, err := r.q.ValidateFolderDepth(ctx, *parentID)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (r *repository) GetChildCount(ctx context.Context, parentID uuid.UUID) (int32, error) {
	count, err := r.q.GetChildFoldersCount(ctx, &parentID)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (r *repository) GetDocumentsCount(ctx context.Context, folderID uuid.UUID) (int32, error) {
	count, err := r.q.GetFolderDocumentsCount(ctx, &folderID)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (r *repository) GetAncestors(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error) {
	rows, err := r.q.GetAncestorFolders(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]sqlc.DocumentFolder, len(rows))
	for i, row := range rows {
		result[i] = sqlc.DocumentFolder{
			ID:       row.ID,
			ParentID: row.ParentID,
			Name:     row.Name,
			Depth:    row.Depth,
		}
	}
	return result, nil
}

func (r *repository) GetDescendants(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error) {
	rows, err := r.q.GetDescendantFolders(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]sqlc.DocumentFolder, len(rows))
	for i, row := range rows {
		result[i] = sqlc.DocumentFolder{
			ID:       row.ID,
			ParentID: row.ParentID,
			Name:     row.Name,
			Depth:    row.Depth,
		}
	}
	return result, nil
}

func (r *repository) GetFolderTree(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentFolder, error) {
	rows, err := r.q.GetFolderTree(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]sqlc.DocumentFolder, len(rows))
	for i, row := range rows {
		result[i] = sqlc.DocumentFolder{
			ID:       row.ID,
			ParentID: row.ParentID,
			Name:     row.Name,
			Depth:    row.Depth,
		}
	}
	return result, nil
}

func (r *repository) GetSiblings(ctx context.Context, parentID uuid.UUID, excludeID uuid.UUID) ([]sqlc.DocumentFolder, error) {
	return r.q.GetSiblings(ctx, sqlc.GetSiblingsParams{
		ParentID: &parentID,
		ID:       excludeID,
	})
}

func (r *repository) Search(ctx context.Context, query string, limit, offset int32) ([]sqlc.DocumentFolder, error) {
	return r.q.SearchDocumentFolders(ctx, sqlc.SearchDocumentFoldersParams{
		Column1: &query,
		Limit:   limit,
		Offset:  offset,
	})
}

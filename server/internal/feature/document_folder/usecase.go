package document_folder

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	// Basic CRUD
	CreateFolder(ctx context.Context, req CreateDocumentFolderRequest) (*DocumentFolderResponse, error)
	GetFolder(ctx context.Context, id uuid.UUID) (*DocumentFolderResponse, error)
	ListFolders(ctx context.Context, params ListDocumentFoldersParams) ([]DocumentFolderResponse, error)
	UpdateFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, req UpdateDocumentFolderRequest) (*DocumentFolderResponse, error)
	DeleteFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error

	// Hierarchy operations
	MoveFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, req MoveDocumentFolderRequest) (*DocumentFolderResponse, error)
	GetFolderTree(ctx context.Context, id uuid.UUID) ([]DocumentFolderResponse, error)
	GetAncestors(ctx context.Context, folderID uuid.UUID) ([]DocumentFolderResponse, error)
	GetDescendants(ctx context.Context, folderID uuid.UUID) ([]DocumentFolderResponse, error)

	// Search
	SearchFolders(ctx context.Context, query string, limit, offset int32) ([]DocumentFolderResponse, error)
}

type usecase struct {
	repo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		repo: repo,
	}
}

func (uc *usecase) CreateFolder(ctx context.Context, req CreateDocumentFolderRequest) (*DocumentFolderResponse, error) {
	// Validate depth
	var depth int32 = 0
	if req.ParentID != nil {
		parentDepth, err := uc.repo.ValidateDepth(ctx, req.ParentID)
		if err != nil {
			return nil, err
		}
		depth = parentDepth
	}

	// Check max depth (max 3 levels: 0, 1, 2)
	if depth > 2 {
		return nil, ErrMaxDepthExceeded
	}

	arg := sqlc.CreateDocumentFolderParams{
		StoryID:  req.StoryID,
		OwnerID:  req.OwnerID,
		Name:     req.Name,
		ParentID: req.ParentID,
		Depth:    depth,
	}

	folder, err := uc.repo.Create(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderResponse(folder), nil
}

func (uc *usecase) GetFolder(ctx context.Context, id uuid.UUID) (*DocumentFolderResponse, error) {
	folder, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToDocumentFolderResponse(folder), nil
}

func (uc *usecase) ListFolders(ctx context.Context, params ListDocumentFoldersParams) ([]DocumentFolderResponse, error) {
	var folders []sqlc.DocumentFolder
	var err error

	if params.ParentID != nil {
		folders, err = uc.repo.GetFoldersByParentID(ctx, *params.ParentID)
	} else if params.StoryID != nil {
		folders, err = uc.repo.GetFoldersByStoryID(ctx, *params.StoryID)
	} else if params.OwnerID != nil {
		folders, err = uc.repo.GetFoldersByOwnerID(ctx, *params.OwnerID)
	} else {
		folders, err = uc.repo.GetRootFolders(ctx)
	}

	if err != nil {
		return nil, err
	}

	return ToDocumentFolderListResponse(folders), nil
}

func (uc *usecase) UpdateFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, req UpdateDocumentFolderRequest) (*DocumentFolderResponse, error) {
	existing, err := uc.repo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if existing.OwnerID != userID {
		return nil, ErrPermissionDenied
	}

	arg := sqlc.UpdateDocumentFolderParams{
		ID:       folderID,
		Name:     existing.Name,
		ParentID: existing.ParentID,
		Depth:    existing.Depth,
	}

	if req.Name != nil {
		arg.Name = *req.Name
	}
	if req.ParentID != nil {
		// Validate new parent depth
		var newDepth int32 = 0
		if req.ParentID != nil {
			parentDepth, err := uc.repo.ValidateDepth(ctx, req.ParentID)
			if err != nil {
				return nil, err
			}
			newDepth = parentDepth
		}

		if newDepth > 2 {
			return nil, ErrMaxDepthExceeded
		}

		arg.ParentID = req.ParentID
		arg.Depth = newDepth
	}

	updated, err := uc.repo.Update(ctx, arg)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderResponse(updated), nil
}

func (uc *usecase) DeleteFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error {
	folder, err := uc.repo.GetByID(ctx, folderID)
	if err != nil {
		return err
	}

	// Check ownership
	if folder.OwnerID != userID {
		return ErrPermissionDenied
	}

	// Check if folder has children
	childCount, err := uc.repo.GetChildCount(ctx, folderID)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrFolderNotEmpty
	}

	// Check if folder has documents
	docCount, err := uc.repo.GetDocumentsCount(ctx, folderID)
	if err != nil {
		return err
	}
	if docCount > 0 {
		return ErrFolderNotEmpty
	}

	return uc.repo.Delete(ctx, folderID)
}

func (uc *usecase) MoveFolder(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, req MoveDocumentFolderRequest) (*DocumentFolderResponse, error) {
	folder, err := uc.repo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if folder.OwnerID != userID {
		return nil, ErrPermissionDenied
	}

	// Validate new depth
	var newDepth int32 = 0
	if req.ParentID != nil {
		parentDepth, err := uc.repo.ValidateDepth(ctx, req.ParentID)
		if err != nil {
			return nil, err
		}
		newDepth = parentDepth
	}

	if newDepth > 2 {
		return nil, ErrMaxDepthExceeded
	}

	updated, err := uc.repo.Move(ctx, folderID, req.ParentID, newDepth)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderResponse(updated), nil
}

func (uc *usecase) GetFolderTree(ctx context.Context, id uuid.UUID) ([]DocumentFolderResponse, error) {
	folders, err := uc.repo.GetFolderTree(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderListResponse(folders), nil
}

func (uc *usecase) GetAncestors(ctx context.Context, folderID uuid.UUID) ([]DocumentFolderResponse, error) {
	folders, err := uc.repo.GetAncestors(ctx, folderID)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderListResponse(folders), nil
}

func (uc *usecase) GetDescendants(ctx context.Context, folderID uuid.UUID) ([]DocumentFolderResponse, error) {
	folders, err := uc.repo.GetDescendants(ctx, folderID)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderListResponse(folders), nil
}

func (uc *usecase) SearchFolders(ctx context.Context, query string, limit, offset int32) ([]DocumentFolderResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	folders, err := uc.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return ToDocumentFolderListResponse(folders), nil
}

// Errors
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrFolderNotFound   = errors.New("folder not found")
	ErrMaxDepthExceeded = errors.New("maximum folder depth exceeded (max 3 levels)")
	ErrFolderNotEmpty   = errors.New("folder is not empty (contains subfolders or documents)")
)

package document_folder_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/document_folder"
	folderMocks "github.com/justblue/samsa/internal/feature/document_folder/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentFolderUseCase_CreateFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("creates root folder successfully", func(t *testing.T) {
		req := document_folder.CreateDocumentFolderRequest{
			StoryID: uuid.New(),
			OwnerID: uuid.New(),
			Name:    "Root Folder",
		}

		expectedFolder := &sqlc.DocumentFolder{
			ID:      uuid.New(),
			StoryID: req.StoryID,
			OwnerID: req.OwnerID,
			Name:    "Root Folder",
			Depth:   int32(0),
		}

		mockRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedFolder, nil)

		result, err := uc.CreateFolder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Root Folder", result.Name)
		assert.Equal(t, int32(0), result.Depth)
	})

	t.Run("creates child folder successfully", func(t *testing.T) {
		parentID := uuid.New()
		req := document_folder.CreateDocumentFolderRequest{
			StoryID:  uuid.New(),
			OwnerID:  uuid.New(),
			Name:     "Child Folder",
			ParentID: &parentID,
		}

		expectedFolder := &sqlc.DocumentFolder{
			ID:       uuid.New(),
			StoryID:  req.StoryID,
			OwnerID:  req.OwnerID,
			Name:     "Child Folder",
			ParentID: &parentID,
			Depth:    int32(1),
		}

		mockRepo.EXPECT().
			ValidateDepth(ctx, &parentID).
			Return(int32(0), nil)

		mockRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedFolder, nil)

		result, err := uc.CreateFolder(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.Depth)
	})

	t.Run("returns error when max depth exceeded", func(t *testing.T) {
		parentID := uuid.New()
		req := document_folder.CreateDocumentFolderRequest{
			StoryID:  uuid.New(),
			OwnerID:  uuid.New(),
			Name:     "Deep Folder",
			ParentID: &parentID,
		}

		mockRepo.EXPECT().
			ValidateDepth(ctx, &parentID).
			Return(int32(3), nil)

		result, err := uc.CreateFolder(ctx, req)
		assert.ErrorIs(t, err, document_folder.ErrMaxDepthExceeded)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderUseCase_GetFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("returns folder when found", func(t *testing.T) {
		folderID := uuid.New()
		expectedFolder := &sqlc.DocumentFolder{
			ID:    folderID,
			Name:  "Test Folder",
			Depth: int32(0),
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(expectedFolder, nil)

		result, err := uc.GetFolder(ctx, folderID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Folder", result.Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockRepo.EXPECT().
			GetByID(ctx, uuid.New()).
			Return(nil, assert.AnError)

		result, err := uc.GetFolder(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderUseCase_ListFolders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("lists folders by parent", func(t *testing.T) {
		parentID := uuid.New()
		params := document_folder.ListDocumentFoldersParams{
			ParentID: &parentID,
		}

		folders := []sqlc.DocumentFolder{
			{ID: uuid.New(), ParentID: &parentID, Name: "Child 1"},
			{ID: uuid.New(), ParentID: &parentID, Name: "Child 2"},
		}

		mockRepo.EXPECT().
			GetFoldersByParentID(ctx, parentID).
			Return(folders, nil)

		result, err := uc.ListFolders(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("lists folders by story", func(t *testing.T) {
		storyID := uuid.New()
		params := document_folder.ListDocumentFoldersParams{
			StoryID: &storyID,
		}

		folders := []sqlc.DocumentFolder{
			{ID: uuid.New(), StoryID: storyID},
		}

		mockRepo.EXPECT().
			GetFoldersByStoryID(ctx, storyID).
			Return(folders, nil)

		result, err := uc.ListFolders(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("lists root folders when no params", func(t *testing.T) {
		params := document_folder.ListDocumentFoldersParams{}

		folders := []sqlc.DocumentFolder{
			{ID: uuid.New(), Depth: int32(0)},
		}

		mockRepo.EXPECT().
			GetRootFolders(ctx).
			Return(folders, nil)

		result, err := uc.ListFolders(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestDocumentFolderUseCase_UpdateFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("updates folder name successfully", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			Name:  "Old Name",
			Depth: int32(0),
		}

		req := document_folder.UpdateDocumentFolderRequest{
			Name: factory.StringPtr("New Name"),
		}

		updated := &sqlc.DocumentFolder{
			ID:    folderID,
			Name:  "New Name",
			Depth: int32(0),
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(updated, nil)

		result, err := uc.UpdateFolder(ctx, userID, folderID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Name", result.Name)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			Name:  "Test Folder",
			OwnerID: ownerID,
		}

		req := document_folder.UpdateDocumentFolderRequest{
			Name: factory.StringPtr("New Name"),
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		result, err := uc.UpdateFolder(ctx, userID, folderID, req)
		assert.ErrorIs(t, err, document_folder.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderUseCase_DeleteFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("deletes empty folder successfully", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: userID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			GetChildCount(ctx, folderID).
			Return(int32(0), nil)

		mockRepo.EXPECT().
			GetDocumentsCount(ctx, folderID).
			Return(int32(0), nil)

		mockRepo.EXPECT().
			Delete(ctx, folderID).
			Return(nil)

		err := uc.DeleteFolder(ctx, userID, folderID)
		require.NoError(t, err)
	})

	t.Run("returns error when folder has children", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: userID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			GetChildCount(ctx, folderID).
			Return(int32(1), nil)

		err := uc.DeleteFolder(ctx, userID, folderID)
		assert.ErrorIs(t, err, document_folder.ErrFolderNotEmpty)
	})

	t.Run("returns error when folder has documents", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: userID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			GetChildCount(ctx, folderID).
			Return(int32(0), nil)

		mockRepo.EXPECT().
			GetDocumentsCount(ctx, folderID).
			Return(int32(1), nil)

		err := uc.DeleteFolder(ctx, userID, folderID)
		assert.ErrorIs(t, err, document_folder.ErrFolderNotEmpty)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: ownerID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		err := uc.DeleteFolder(ctx, userID, folderID)
		assert.ErrorIs(t, err, document_folder.ErrPermissionDenied)
	})
}

func TestDocumentFolderUseCase_MoveFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("moves folder successfully", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		newParentID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: userID,
			Depth: int32(0),
		}

		req := document_folder.MoveDocumentFolderRequest{
			ParentID: &newParentID,
		}

		moved := &sqlc.DocumentFolder{
			ID:       folderID,
			OwnerID:  userID,
			ParentID: &newParentID,
			Depth:    int32(1),
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			ValidateDepth(ctx, &newParentID).
			Return(int32(0), nil)

		mockRepo.EXPECT().
			Move(ctx, folderID, &newParentID, int32(1)).
			Return(moved, nil)

		result, err := uc.MoveFolder(ctx, userID, folderID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.Depth)
	})

	t.Run("returns error when max depth exceeded", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		newParentID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: userID,
		}

		req := document_folder.MoveDocumentFolderRequest{
			ParentID: &newParentID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		mockRepo.EXPECT().
			ValidateDepth(ctx, &newParentID).
			Return(int32(3), nil)

		result, err := uc.MoveFolder(ctx, userID, folderID, req)
		assert.ErrorIs(t, err, document_folder.ErrMaxDepthExceeded)
		assert.Nil(t, result)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		folderID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.DocumentFolder{
			ID:    folderID,
			OwnerID: ownerID,
		}

		req := document_folder.MoveDocumentFolderRequest{}

		mockRepo.EXPECT().
			GetByID(ctx, folderID).
			Return(existing, nil)

		result, err := uc.MoveFolder(ctx, userID, folderID, req)
		assert.ErrorIs(t, err, document_folder.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderUseCase_GetFolderTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("returns folder tree", func(t *testing.T) {
		folderID := uuid.New()
		folders := []sqlc.DocumentFolder{
			{ID: folderID, Name: "Root", Depth: int32(0)},
			{ID: uuid.New(), Name: "Child", Depth: int32(1)},
		}

		mockRepo.EXPECT().
			GetFolderTree(ctx, folderID).
			Return(folders, nil)

		result, err := uc.GetFolderTree(ctx, folderID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestDocumentFolderUseCase_GetAncestors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("returns ancestor folders", func(t *testing.T) {
		folderID := uuid.New()
		ancestors := []sqlc.DocumentFolder{
			{ID: uuid.New(), Name: "Parent", Depth: int32(1)},
			{ID: uuid.New(), Name: "Grandparent", Depth: int32(0)},
		}

		mockRepo.EXPECT().
			GetAncestors(ctx, folderID).
			Return(ancestors, nil)

		result, err := uc.GetAncestors(ctx, folderID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestDocumentFolderUseCase_GetDescendants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("returns descendant folders", func(t *testing.T) {
		folderID := uuid.New()
		descendants := []sqlc.DocumentFolder{
			{ID: uuid.New(), Name: "Child 1", Depth: int32(1)},
			{ID: uuid.New(), Name: "Child 2", Depth: int32(1)},
		}

		mockRepo.EXPECT().
			GetDescendants(ctx, folderID).
			Return(descendants, nil)

		result, err := uc.GetDescendants(ctx, folderID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestDocumentFolderUseCase_SearchFolders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := folderMocks.NewMockRepository(ctrl)
	uc := document_folder.NewUseCase(mockRepo)

	t.Run("searches folders successfully", func(t *testing.T) {
		query := "test"
		folders := []sqlc.DocumentFolder{
			{ID: uuid.New(), Name: "Test Folder 1"},
			{ID: uuid.New(), Name: "Test Folder 2"},
		}

		mockRepo.EXPECT().
			Search(ctx, query, int32(20), int32(0)).
			Return(folders, nil)

		result, err := uc.SearchFolders(ctx, query, 20, 0)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("uses default limit when not provided", func(t *testing.T) {
		query := "test"
		folders := []sqlc.DocumentFolder{}

		mockRepo.EXPECT().
			Search(ctx, query, int32(20), int32(0)).
			Return(folders, nil)

		result, err := uc.SearchFolders(ctx, query, 0, 0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

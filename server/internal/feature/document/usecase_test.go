package document_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/document"
	documentMocks "github.com/justblue/samsa/internal/feature/document/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentUseCase_CreateDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("creates document successfully", func(t *testing.T) {
		userID := uuid.New()
		storyID := uuid.New()
		req := document.CreateDocumentRequest{
			StoryID:    storyID,
			Language:   "eng",
			BranchName: "main",
			Title:      "Test Document",
			Slug:       "test-document",
		}

		expectedDoc := &sqlc.Document{
			ID:            uuid.New(),
			StoryID:       storyID,
			CreatedBy:     userID,
			Language:      "eng",
			BranchName:    "main",
			Title:         factory.StringPtr("Test Document"),
			Slug:          factory.StringPtr("test-document"),
			Status:        sqlc.DocumentStatusDraft,
			IsLocked:      factory.BoolPtr(false),
			IsTemplate:    factory.BoolPtr(false),
			TotalWords:    factory.Int32Ptr(0),
			VersionNumber: 1,
		}

		mockRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(expectedDoc, nil)

		mockRepo.EXPECT().
			CreateStatusHistory(ctx, gomock.Any()).
			Return(&sqlc.DocumentStatusHistory{}, nil)

		result, err := uc.CreateDocument(ctx, userID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Document", *result.Title)
		assert.Equal(t, document.DocumentStatusDraft, result.Status)
	})
}

func TestDocumentUseCase_GetDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("returns document when found", func(t *testing.T) {
		docID := uuid.New()
		expectedDoc := &sqlc.Document{
			ID:         docID,
			Title:      factory.StringPtr("Test Document"),
			StoryID:    uuid.New(),
			Language:   "eng",
			BranchName: "main",
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(expectedDoc, nil)

		result, err := uc.GetDocument(ctx, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Document", *result.Title)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		mockRepo.EXPECT().
			GetByID(ctx, uuid.New()).
			Return(nil, assert.AnError)

		result, err := uc.GetDocument(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentUseCase_GetDocumentBySlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("returns document by slug", func(t *testing.T) {
		expectedDoc := &sqlc.Document{
			ID:    uuid.New(),
			Slug:  factory.StringPtr("test-slug"),
			Title: factory.StringPtr("Test Document"),
		}

		mockRepo.EXPECT().
			GetBySlug(ctx, "test-slug").
			Return(expectedDoc, nil)

		result, err := uc.GetDocumentBySlug(ctx, "test-slug")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-slug", *result.Slug)
	})
}

func TestDocumentUseCase_ListDocuments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("lists documents by story", func(t *testing.T) {
		storyID := uuid.New()
		params := document.ListDocumentsParams{
			StoryID: &storyID,
		}

		docs := []sqlc.Document{
			{ID: uuid.New(), StoryID: storyID, Title: factory.StringPtr("Doc 1")},
			{ID: uuid.New(), StoryID: storyID, Title: factory.StringPtr("Doc 2")},
		}

		mockRepo.EXPECT().
			GetDocumentsByStoryID(ctx, gomock.Any()).
			Return(docs, nil)

		result, err := uc.ListDocuments(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("lists documents by folder", func(t *testing.T) {
		folderID := uuid.New()
		params := document.ListDocumentsParams{
			FolderID: &folderID,
		}

		docs := []sqlc.Document{
			{ID: uuid.New(), FolderID: &folderID},
		}

		mockRepo.EXPECT().
			GetDocumentsByFolderID(ctx, gomock.Any()).
			Return(docs, nil)

		result, err := uc.ListDocuments(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("lists documents by status", func(t *testing.T) {
		status := document.DocumentStatusDraft
		params := document.ListDocumentsParams{
			Status: &status,
		}

		docs := []sqlc.Document{
			{ID: uuid.New(), Status: sqlc.DocumentStatusDraft},
		}

		mockRepo.EXPECT().
			GetDocumentsByStatus(ctx, gomock.Any()).
			Return(docs, nil)

		result, err := uc.ListDocuments(ctx, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestDocumentUseCase_UpdateDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("updates document successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
			Title:     factory.StringPtr("Old Title"),
		}

		req := document.UpdateDocumentRequest{
			Title: factory.StringPtr("New Title"),
		}

		updated := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
			Title:     factory.StringPtr("New Title"),
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		mockRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(updated, nil)

		result, err := uc.UpdateDocument(ctx, userID, docID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Title", *result.Title)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: ownerID,
		}

		req := document.UpdateDocumentRequest{
			Title: factory.StringPtr("New Title"),
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		result, err := uc.UpdateDocument(ctx, userID, docID, req)
		assert.ErrorIs(t, err, document.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentUseCase_DeleteDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("deletes document successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		mockRepo.EXPECT().
			SoftDelete(ctx, docID).
			Return(existing, nil)

		err := uc.DeleteDocument(ctx, userID, docID)
		require.NoError(t, err)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: ownerID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		err := uc.DeleteDocument(ctx, userID, docID)
		assert.ErrorIs(t, err, document.ErrPermissionDenied)
	})
}

func TestDocumentUseCase_SubmitForReview(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("submits document for review", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
			Status:    sqlc.DocumentStatusDraft,
		}

		updated := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
			Status:    sqlc.DocumentStatusPendingReview,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		mockRepo.EXPECT().
			SubmitForReview(ctx, docID).
			Return(updated, nil)

		mockRepo.EXPECT().
			CreateStatusHistory(ctx, gomock.Any()).
			Return(&sqlc.DocumentStatusHistory{}, nil)

		result, err := uc.SubmitForReview(ctx, userID, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.DocumentStatusPendingReview, result.Status)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: ownerID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		result, err := uc.SubmitForReview(ctx, userID, docID)
		assert.ErrorIs(t, err, document.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentUseCase_ApproveDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("approves document successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()

		approved := &sqlc.Document{
			ID:     docID,
			Status: sqlc.DocumentStatusIsApproved,
		}

		mockRepo.EXPECT().
			Approve(ctx, docID).
			Return(approved, nil)

		mockRepo.EXPECT().
			CreateStatusHistory(ctx, gomock.Any()).
			Return(&sqlc.DocumentStatusHistory{}, nil)

		result, err := uc.ApproveDocument(ctx, userID, docID, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.DocumentStatusIsApproved, result.Status)
	})
}

func TestDocumentUseCase_RejectDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("rejects document successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()

		rejected := &sqlc.Document{
			ID:     docID,
			Status: sqlc.DocumentStatusRejected,
		}

		mockRepo.EXPECT().
			Reject(ctx, docID).
			Return(rejected, nil)

		mockRepo.EXPECT().
			CreateStatusHistory(ctx, gomock.Any()).
			Return(&sqlc.DocumentStatusHistory{}, nil)

		result, err := uc.RejectDocument(ctx, userID, docID, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.DocumentStatusRejected, result.Status)
	})
}

func TestDocumentUseCase_ArchiveDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("archives document successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
		}

		archived := &sqlc.Document{
			ID:        docID,
			CreatedBy: userID,
			Status:    sqlc.DocumentStatusArchived,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		mockRepo.EXPECT().
			Archive(ctx, docID).
			Return(archived, nil)

		result, err := uc.ArchiveDocument(ctx, userID, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.DocumentStatusArchived, result.Status)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: ownerID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		result, err := uc.ArchiveDocument(ctx, userID, docID)
		assert.ErrorIs(t, err, document.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentUseCase_ReviewDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("reviews document with status change", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()

		reviewed := &sqlc.Document{
			ID:       docID,
			Status:   sqlc.DocumentStatusIsReviewed,
			IsLocked: factory.BoolPtr(false),
		}

		req := document.ReviewDocumentRequest{
			Status: document.DocumentStatusIsReviewed,
		}

		mockRepo.EXPECT().
			Review(ctx, docID, sqlc.DocumentStatusIsReviewed, false).
			Return(reviewed, nil)

		mockRepo.EXPECT().
			CreateStatusHistory(ctx, gomock.Any()).
			Return(&sqlc.DocumentStatusHistory{}, nil)

		result, err := uc.ReviewDocument(ctx, userID, docID, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, document.DocumentStatusIsReviewed, result.Status)
	})
}

func TestDocumentUseCase_IncrementView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("increments view count", func(t *testing.T) {
		docID := uuid.New()

		incremented := &sqlc.Document{
			ID:         docID,
			TotalViews: factory.Int32Ptr(1),
		}

		mockRepo.EXPECT().
			IncrementViews(ctx, docID).
			Return(incremented, nil)

		result, err := uc.IncrementView(ctx, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.TotalViews)
	})
}

func TestDocumentUseCase_IncrementDownload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("increments download count", func(t *testing.T) {
		docID := uuid.New()

		incremented := &sqlc.Document{
			ID:             docID,
			TotalDownloads: factory.Int32Ptr(1),
		}

		mockRepo.EXPECT().
			IncrementDownloads(ctx, docID).
			Return(incremented, nil)

		result, err := uc.IncrementDownload(ctx, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.TotalDownloads)
	})
}

func TestDocumentUseCase_IncrementShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("increments share count", func(t *testing.T) {
		docID := uuid.New()

		incremented := &sqlc.Document{
			ID:          docID,
			TotalShares: factory.Int32Ptr(1),
		}

		mockRepo.EXPECT().
			IncrementShares(ctx, docID).
			Return(incremented, nil)

		result, err := uc.IncrementShare(ctx, docID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), result.TotalShares)
	})
}

func TestDocumentUseCase_CreateNewVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("creates new version successfully", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		existing := &sqlc.Document{
			ID:            docID,
			CreatedBy:     userID,
			VersionNumber: 1,
			Content:       []byte("Original content"),
		}

		newContent := []byte("Updated content")
		updated := &sqlc.Document{
			ID:                docID,
			CreatedBy:         userID,
			VersionNumber:     2,
			PreviousVersionID: &docID,
			Content:           newContent,
			TotalWords:        factory.Int32Ptr(100),
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		mockRepo.EXPECT().
			UpdateVersion(ctx, docID, int32(2), gomock.Any(), newContent, gomock.Any()).
			Return(updated, nil)

		result, err := uc.CreateNewVersion(ctx, userID, docID, newContent)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(2), result.VersionNumber)
	})

	t.Run("returns permission denied when not owner", func(t *testing.T) {
		docID := uuid.New()
		userID := uuid.New()
		ownerID := uuid.New()
		existing := &sqlc.Document{
			ID:        docID,
			CreatedBy: ownerID,
		}

		mockRepo.EXPECT().
			GetByID(ctx, docID).
			Return(existing, nil)

		result, err := uc.CreateNewVersion(ctx, userID, docID, []byte("New content"))
		assert.ErrorIs(t, err, document.ErrPermissionDenied)
		assert.Nil(t, result)
	})
}

func TestDocumentUseCase_GetVersionHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("returns version history", func(t *testing.T) {
		docID := uuid.New()
		versions := []sqlc.Document{
			{ID: docID, VersionNumber: 1},
			{ID: uuid.New(), VersionNumber: 2, PreviousVersionID: &docID},
		}

		mockRepo.EXPECT().
			GetVersionHistory(ctx, docID).
			Return(versions, nil)

		result, err := uc.GetVersionHistory(ctx, docID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestDocumentUseCase_GetStatusHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := documentMocks.NewMockRepository(ctrl)
	uc := document.NewUseCase(mockRepo)

	t.Run("returns status history", func(t *testing.T) {
		docID := uuid.New()
		history := []sqlc.DocumentStatusHistory{
			{DocumentID: docID, Status: sqlc.DocumentStatusDraft},
			{DocumentID: docID, Status: sqlc.DocumentStatusPendingReview},
		}

		mockRepo.EXPECT().
			ListStatusHistory(ctx, docID).
			Return(history, nil)

		result, err := uc.GetStatusHistory(ctx, docID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

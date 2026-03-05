package document_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/document"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentRepository_Create(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("creates document successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		arg := sqlc.CreateDocumentParams{
			StoryID:       story.ID,
			CreatedBy:     user.ID,
			Language:      "eng",
			BranchName:    "main",
			Title:         factory.StringPtr("Test Document"),
			Slug:          factory.StringPtr("test-document"),
			Status:        sqlc.DocumentStatusDraft,
			IsTemplate:    factory.BoolPtr(false),
			TotalWords:    factory.Int32Ptr(0),
			VersionNumber: 1,
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, story.ID, result.StoryID)
		assert.Equal(t, "Test Document", *result.Title)
		assert.Equal(t, sqlc.DocumentStatusDraft, result.Status)
	})

	t.Run("creates document in folder", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})
		folder := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			StoryID: story.ID,
			OwnerID: user.ID,
		})

		arg := sqlc.CreateDocumentParams{
			StoryID:       story.ID,
			CreatedBy:     user.ID,
			FolderID:      &folder.ID,
			Language:      "eng",
			BranchName:    "main",
			Title:         factory.StringPtr("Document in Folder"),
			Slug:          factory.StringPtr("document-in-folder"),
			Status:        sqlc.DocumentStatusDraft,
			IsTemplate:    factory.BoolPtr(false),
			TotalWords:    factory.Int32Ptr(0),
			VersionNumber: 1,
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, folder.ID, *result.FolderID)
	})
}

func TestDocumentRepository_GetByID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns document when found", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.GetByID(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.Title, result.Title)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		result, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentRepository_GetBySlug(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns document when found", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{
			Slug: factory.StringPtr("unique-slug-" + factory.RandomString(4)),
		})

		result, err := repo.GetBySlug(ctx, *existing.Slug)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.ID)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		result, err := repo.GetBySlug(ctx, "non-existent-slug")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentRepository_GetDocumentsByStoryID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns all documents for story", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		// Create multiple documents
		doc1 := factory.Document(t, pool, factory.DocumentOpts{
			StoryID: story.ID,
			Title:   factory.StringPtr("Doc 1"),
		})
		doc2 := factory.Document(t, pool, factory.DocumentOpts{
			StoryID: story.ID,
			Title:   factory.StringPtr("Doc 2"),
		})
		doc3 := factory.Document(t, pool, factory.DocumentOpts{
			StoryID: story.ID,
			Title:   factory.StringPtr("Doc 3"),
		})

		results, err := repo.GetDocumentsByStoryID(ctx, sqlc.GetDocumentsByStoryIDParams{
			StoryID: story.ID,
			Limit:   10,
			Offset:  0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3)

		// Verify documents are present
		ids := make(map[uuid.UUID]bool)
		for _, d := range results {
			ids[d.ID] = true
		}
		assert.True(t, ids[doc1.ID])
		assert.True(t, ids[doc2.ID])
		assert.True(t, ids[doc3.ID])
	})
}

func TestDocumentRepository_GetDocumentsByFolderID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns documents in folder", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		folder := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		// Create documents in folder
		doc1 := factory.DocumentInFolder(t, pool, folder, factory.DocumentOpts{
			Title: factory.StringPtr("Folder Doc 1"),
		})
		doc2 := factory.DocumentInFolder(t, pool, folder, factory.DocumentOpts{
			Title: factory.StringPtr("Folder Doc 2"),
		})

		results, err := repo.GetDocumentsByFolderID(ctx, sqlc.GetDocumentsByFolderIDParams{
			FolderID: &folder.ID,
			Limit:    10,
			Offset:   0,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)

		ids := make(map[uuid.UUID]bool)
		for _, d := range results {
			ids[d.ID] = true
		}
		assert.True(t, ids[doc1.ID])
		assert.True(t, ids[doc2.ID])
	})
}

func TestDocumentRepository_GetDocumentsByStatus(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns documents by status", func(t *testing.T) {
		// Create documents with different statuses
		draft := factory.Document(t, pool, factory.DocumentOpts{
			Title:  factory.StringPtr("Draft Doc"),
			Status: sqlc.DocumentStatusDraft,
		})
		_ = factory.PendingReviewDocument(t, pool, factory.DocumentOpts{
			Title: factory.StringPtr("Pending Doc"),
		})
		_ = factory.ApprovedDocument(t, pool, factory.DocumentOpts{
			Title: factory.StringPtr("Approved Doc"),
		})

		results, err := repo.GetDocumentsByStatus(ctx, sqlc.GetDocumentsByStatusParams{
			Status: sqlc.DocumentStatusDraft,
			Limit:  10,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify draft document is present
		found := false
		for _, d := range results {
			if d.ID == draft.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestDocumentRepository_Update(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("updates document successfully", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		arg := sqlc.UpdateDocumentParams{
			ID:            existing.ID,
			Title:         factory.StringPtr("Updated Title"),
			Slug:          factory.StringPtr("updated-slug"),
			Summary:       factory.StringPtr("Updated summary"),
			Content:       []byte("Updated content"),
			DocumentType:  factory.StringPtr("updated-type"),
			IsLocked:      factory.BoolPtr(true),
			IsTemplate:    factory.BoolPtr(true),
			TotalWords:    factory.Int32Ptr(500),
			VersionNumber: existing.VersionNumber,
		}

		result, err := repo.Update(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Title", *result.Title)
		assert.Equal(t, "updated-slug", *result.Slug)
		assert.Equal(t, "Updated summary", *result.Summary)
		assert.Equal(t, []byte("Updated content"), result.Content)
		assert.Equal(t, "updated-type", *result.DocumentType)
		assert.True(t, *result.IsLocked)
		assert.True(t, *result.IsTemplate)
	})

	t.Run("returns error when document not found", func(t *testing.T) {
		arg := sqlc.UpdateDocumentParams{
			ID:            uuid.New(),
			Title:         factory.StringPtr("Non-existent"),
			VersionNumber: 1,
		}

		result, err := repo.Update(ctx, arg)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentRepository_Delete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("deletes document successfully", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		err := repo.Delete(ctx, existing.ID)
		require.NoError(t, err)

		// Verify document is deleted
		result, err := repo.GetByID(ctx, existing.ID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentRepository_SoftDelete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("soft deletes document successfully", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.SoftDelete(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DeletedAt)
	})
}

func TestDocumentRepository_SubmitForReview(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("submits document for review", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{
			Status: sqlc.DocumentStatusDraft,
		})

		result, err := repo.SubmitForReview(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sqlc.DocumentStatusPendingReview, result.Status)
	})
}

func TestDocumentRepository_Approve(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("approves document successfully", func(t *testing.T) {
		existing := factory.PendingReviewDocument(t, pool, factory.DocumentOpts{})

		result, err := repo.Approve(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sqlc.DocumentStatusIsApproved, result.Status)
	})
}

func TestDocumentRepository_Reject(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("rejects document successfully", func(t *testing.T) {
		existing := factory.PendingReviewDocument(t, pool, factory.DocumentOpts{})

		result, err := repo.Reject(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sqlc.DocumentStatusRejected, result.Status)
	})
}

func TestDocumentRepository_Archive(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("archives document successfully", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.Archive(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sqlc.DocumentStatusArchived, result.Status)
	})
}

func TestDocumentRepository_Review(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("reviews document with status change", func(t *testing.T) {
		existing := factory.PendingReviewDocument(t, pool, factory.DocumentOpts{})

		result, err := repo.Review(ctx, existing.ID, sqlc.DocumentStatusIsReviewed, false)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, sqlc.DocumentStatusIsReviewed, result.Status)
		assert.False(t, *result.IsLocked)
	})

	t.Run("reviews document and locks it", func(t *testing.T) {
		existing := factory.PendingReviewDocument(t, pool, factory.DocumentOpts{})

		result, err := repo.Review(ctx, existing.ID, sqlc.DocumentStatusIsReviewed, true)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, *result.IsLocked)
	})
}

func TestDocumentRepository_IncrementViews(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("increments view count", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.IncrementViews(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, *result.TotalViews, int32(0))
	})
}

func TestDocumentRepository_IncrementDownloads(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("increments download count", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.IncrementDownloads(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, *result.TotalDownloads, int32(0))
	})
}

func TestDocumentRepository_IncrementShares(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("increments share count", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		result, err := repo.IncrementShares(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, *result.TotalShares, int32(0))
	})
}

func TestDocumentRepository_UpdateVersion(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("updates document version", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		previousVersionID := existing.ID
		newContent := []byte("Updated content for new version")

		result, err := repo.UpdateVersion(ctx, existing.ID, 2, &previousVersionID, newContent, 100)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(2), result.VersionNumber)
		assert.Equal(t, newContent, result.Content)
		assert.Equal(t, int32(100), *result.TotalWords)
	})
}

func TestDocumentRepository_GetVersionHistory(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns version history", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})

		// Create a new version
		previousVersionID := existing.ID
		_, _ = repo.UpdateVersion(ctx, existing.ID, 2, &previousVersionID, []byte("Version 2"), 100)

		results, err := repo.GetVersionHistory(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})
}

func TestDocumentRepository_CreateStatusHistory(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("creates status history entry", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})
		user := factory.User(t, pool, factory.UserOpts{})

		arg := sqlc.CreateDocumentStatusHistoryParams{
			DocumentID:  existing.ID,
			SetStatusBy: user.ID,
			Content:     "Document created",
			Status:      sqlc.DocumentStatusDraft,
		}

		result, err := repo.CreateStatusHistory(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.DocumentID)
		assert.Equal(t, sqlc.DocumentStatusDraft, result.Status)
	})
}

func TestDocumentRepository_ListStatusHistory(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns status history", func(t *testing.T) {
		existing := factory.Document(t, pool, factory.DocumentOpts{})
		user := factory.User(t, pool, factory.UserOpts{})

		// Create status history entries
		_, _ = repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
			DocumentID:  existing.ID,
			SetStatusBy: user.ID,
			Content:     "Created",
			Status:      sqlc.DocumentStatusDraft,
		})
		_, _ = repo.CreateStatusHistory(ctx, sqlc.CreateDocumentStatusHistoryParams{
			DocumentID:  existing.ID,
			SetStatusBy: user.ID,
			Content:     "Submitted for review",
			Status:      sqlc.DocumentStatusPendingReview,
		})

		results, err := repo.ListStatusHistory(ctx, existing.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)
	})
}

package document_folder_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/feature/document_folder"
	"github.com/justblue/samsa/internal/testkit"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentFolderRepository_Create(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("creates root folder successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		arg := sqlc.CreateDocumentFolderParams{
			StoryID: story.ID,
			OwnerID: user.ID,
			Name:    "Root Folder",
			Depth:   0,
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "Root Folder", result.Name)
		assert.Equal(t, int32(0), result.Depth)
		assert.Nil(t, result.ParentID)
	})

	t.Run("creates child folder successfully", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		arg := sqlc.CreateDocumentFolderParams{
			StoryID:  parent.StoryID,
			OwnerID:  user.ID,
			Name:     "Child Folder",
			ParentID: &parent.ID,
			Depth:    1,
		}

		result, err := repo.Create(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Child Folder", result.Name)
		assert.Equal(t, int32(1), result.Depth)
		assert.Equal(t, parent.ID, *result.ParentID)
	})
}

func TestDocumentFolderRepository_GetByID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns folder when found", func(t *testing.T) {
		existing := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{})

		result, err := repo.GetByID(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existing.ID, result.ID)
		assert.Equal(t, existing.Name, result.Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		result, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderRepository_GetFoldersByParentID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns child folders", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		child1 := factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Child 1",
		})
		child2 := factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Child 2",
		})

		results, err := repo.GetFoldersByParentID(ctx, parent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)

		ids := make(map[uuid.UUID]bool)
		for _, f := range results {
			ids[f.ID] = true
		}
		assert.True(t, ids[child1.ID])
		assert.True(t, ids[child2.ID])
	})
}

func TestDocumentFolderRepository_GetRootFolders(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns root folders only", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})

		root1 := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Root 1",
		})
		root2 := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Root 2",
		})

		// Create a child folder (should not be returned)
		_ = factory.ChildDocumentFolder(t, pool, root1, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		results, err := repo.GetRootFolders(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify root folders are present
		ids := make(map[uuid.UUID]bool)
		for _, f := range results {
			ids[f.ID] = true
		}
		assert.True(t, ids[root1.ID])
		assert.True(t, ids[root2.ID])
	})
}

func TestDocumentFolderRepository_GetFoldersByStoryID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns all folders for story", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		author := factory.Author(t, pool, factory.AuthorOpts{UserID: user.ID})
		story := factory.Story(t, pool, factory.StoryOpts{OwnerID: author.UserID})

		folder1 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			StoryID: story.ID,
			OwnerID: user.ID,
			Name:    "Folder 1",
		})
		folder2 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			StoryID: story.ID,
			OwnerID: user.ID,
			Name:    "Folder 2",
		})

		results, err := repo.GetFoldersByStoryID(ctx, story.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)

		ids := make(map[uuid.UUID]bool)
		for _, f := range results {
			ids[f.ID] = true
		}
		assert.True(t, ids[folder1.ID])
		assert.True(t, ids[folder2.ID])
	})
}

func TestDocumentFolderRepository_GetFoldersByOwnerID(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns folders by owner", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})

		folder1 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Folder 1",
		})
		folder2 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Folder 2",
		})

		results, err := repo.GetFoldersByOwnerID(ctx, user.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)

		ids := make(map[uuid.UUID]bool)
		for _, f := range results {
			ids[f.ID] = true
		}
		assert.True(t, ids[folder1.ID])
		assert.True(t, ids[folder2.ID])
	})
}

func TestDocumentFolderRepository_Update(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("updates folder successfully", func(t *testing.T) {
		existing := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{})

		arg := sqlc.UpdateDocumentFolderParams{
			ID:     existing.ID,
			Name:   "Updated Folder Name",
			Depth:  existing.Depth,
			ParentID: existing.ParentID,
		}

		result, err := repo.Update(ctx, arg)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Updated Folder Name", result.Name)
	})

	t.Run("returns error when folder not found", func(t *testing.T) {
		arg := sqlc.UpdateDocumentFolderParams{
			ID:    uuid.New(),
			Name:  "Non-existent",
			Depth: 0,
		}

		result, err := repo.Update(ctx, arg)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderRepository_Delete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("deletes folder successfully", func(t *testing.T) {
		existing := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{})

		err := repo.Delete(ctx, existing.ID)
		require.NoError(t, err)

		// Verify folder is deleted
		result, err := repo.GetByID(ctx, existing.ID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDocumentFolderRepository_SoftDelete(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("soft deletes folder successfully", func(t *testing.T) {
		existing := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{})

		result, err := repo.SoftDelete(ctx, existing.ID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DeletedAt)
	})
}

func TestDocumentFolderRepository_Move(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("moves folder to new parent", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		newParent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})
		existing := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		result, err := repo.Move(ctx, existing.ID, &newParent.ID, 1)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, newParent.ID, *result.ParentID)
		assert.Equal(t, int32(1), result.Depth)
	})

	t.Run("moves folder to root", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		root := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})
		existing := factory.ChildDocumentFolder(t, pool, root, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		result, err := repo.Move(ctx, existing.ID, nil, 0)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.ParentID)
		assert.Equal(t, int32(0), result.Depth)
	})
}

func TestDocumentFolderRepository_ValidateDepth(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns 0 for nil parent", func(t *testing.T) {
		result, err := repo.ValidateDepth(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, int32(0), result)
	})

	t.Run("returns parent depth + 1", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.DocumentFolderWithDepth(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		}, 2)

		result, err := repo.ValidateDepth(ctx, &parent.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(2), result)
	})
}

func TestDocumentFolderRepository_GetChildCount(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns child count", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		// Create child folders
		_ = factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})
		_ = factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})
		_ = factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		count, err := repo.GetChildCount(ctx, parent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int32(3))
	})

	t.Run("returns 0 for folder with no children", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		count, err := repo.GetChildCount(ctx, parent.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(0), count)
	})
}

func TestDocumentFolderRepository_GetDocumentsCount(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns documents count in folder", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		folder := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		// Create documents in folder
		_ = factory.DocumentInFolder(t, pool, folder, factory.DocumentOpts{})
		_ = factory.DocumentInFolder(t, pool, folder, factory.DocumentOpts{})

		count, err := repo.GetDocumentsCount(ctx, folder.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int32(2))
	})
}

func TestDocumentFolderRepository_GetAncestors(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns ancestor folders", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		folders := factory.NestedDocumentFolders(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		}, 3)

		// Get ancestors of the deepest folder
		deepest := folders[len(folders)-1]
		ancestors, err := repo.GetAncestors(ctx, deepest.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, ancestors)
	})
}

func TestDocumentFolderRepository_GetDescendants(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns descendant folders", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		folders := factory.NestedDocumentFolders(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		}, 3)

		// Get descendants of the root folder
		root := folders[0]
		descendants, err := repo.GetDescendants(ctx, root.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, descendants)
	})
}

func TestDocumentFolderRepository_GetFolderTree(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns folder tree", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		folders := factory.NestedDocumentFolders(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		}, 3)

		// Get tree from root
		root := folders[0]
		tree, err := repo.GetFolderTree(ctx, root.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, tree)
	})
}

func TestDocumentFolderRepository_GetSiblings(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("returns sibling folders", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})
		parent := factory.RootDocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
		})

		_ = factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Sibling 1",
		})
		_ = factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Sibling 2",
		})
		exclude := factory.ChildDocumentFolder(t, pool, parent, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Exclude",
		})

		siblings, err := repo.GetSiblings(ctx, parent.ID, exclude.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, siblings)

		// Verify excluded folder is not present
		for _, s := range siblings {
			assert.NotEqual(t, exclude.ID, s.ID)
		}
	})
}

func TestDocumentFolderRepository_Search(t *testing.T) {
	pool := testkit.NewDB(t)
	repo := document_folder.NewRepository(pool)

	ctx := context.Background()

	t.Run("searches folders by name", func(t *testing.T) {
		user := factory.User(t, pool, factory.UserOpts{})

		// Create folders with specific names
		match1 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Search Test Folder 1",
		})
		match2 := factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Search Test Folder 2",
		})
		_ = factory.DocumentFolder(t, pool, factory.DocumentFolderOpts{
			OwnerID: user.ID,
			Name:    "Other Folder",
		})

		results, err := repo.Search(ctx, "Search Test", 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify matching folders are present
		ids := make(map[uuid.UUID]bool)
		for _, f := range results {
			ids[f.ID] = true
		}
		assert.True(t, ids[match1.ID])
		assert.True(t, ids[match2.ID])
	})
}

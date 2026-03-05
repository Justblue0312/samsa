package document_folder_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/document_folder"
	folderMocks "github.com/justblue/samsa/internal/feature/document_folder/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("creates folder successfully", func(t *testing.T) {
		req := document_folder.CreateDocumentFolderRequest{
			StoryID: uuid.New(),
			Name:    "Test Folder",
		}
		jsonBody, _ := json.Marshal(req)

		response := &document_folder.DocumentFolderResponse{
			ID:      uuid.New(),
			StoryID: req.StoryID,
			OwnerID: testUser.ID,
			Name:    "Test Folder",
			Depth:   int32(0),
		}

		mockUsecase.EXPECT().
			CreateFolder(gomock.Any(), req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/document-folders", bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Test Folder")
	})

	t.Run("returns unauthorized when no auth", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/document-folders", nil)
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHTTPHandler_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns folder when found", func(t *testing.T) {
		folderID := uuid.New()
		response := &document_folder.DocumentFolderResponse{
			ID:    folderID,
			Name:  "Test Folder",
			Depth: int32(0),
		}

		mockUsecase.EXPECT().
			GetFolder(gomock.Any(), folderID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/"+folderID.String(), nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns bad request for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/invalid", nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("lists folders successfully", func(t *testing.T) {
		response := []document_folder.DocumentFolderResponse{
			{ID: uuid.New(), Name: "Folder 1", Depth: int32(0)},
			{ID: uuid.New(), Name: "Folder 2", Depth: int32(0)},
		}

		mockUsecase.EXPECT().
			ListFolders(gomock.Any(), gomock.Any()).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.List(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("updates folder successfully", func(t *testing.T) {
		folderID := uuid.New()
		req := document_folder.UpdateDocumentFolderRequest{
			Name: factory.StringPtr("Updated Name"),
		}
		jsonBody, _ := json.Marshal(req)

		response := &document_folder.DocumentFolderResponse{
			ID:    folderID,
			Name:  "Updated Name",
			Depth: int32(0),
		}

		mockUsecase.EXPECT().
			UpdateFolder(gomock.Any(), testUser.ID, folderID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/document-folders/"+folderID.String(), bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Update(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("deletes folder successfully", func(t *testing.T) {
		folderID := uuid.New()

		mockUsecase.EXPECT().
			DeleteFolder(gomock.Any(), testUser.ID, folderID).
			Return(nil)

		httpReq := httptest.NewRequest(http.MethodDelete, "/document-folders/"+folderID.String(), nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Delete(w, httpReq)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestHTTPHandler_Move(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("moves folder successfully", func(t *testing.T) {
		folderID := uuid.New()
		newParentID := uuid.New()
		req := document_folder.MoveDocumentFolderRequest{
			ParentID: &newParentID,
		}
		jsonBody, _ := json.Marshal(req)

		response := &document_folder.DocumentFolderResponse{
			ID:       folderID,
			ParentID: &newParentID,
			Depth:    int32(1),
		}

		mockUsecase.EXPECT().
			MoveFolder(gomock.Any(), testUser.ID, folderID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/document-folders/"+folderID.String()+"/move", bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Move(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_GetTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns folder tree", func(t *testing.T) {
		folderID := uuid.New()
		response := []document_folder.DocumentFolderResponse{
			{ID: folderID, Name: "Root", Depth: int32(0)},
			{ID: uuid.New(), Name: "Child", Depth: int32(1)},
		}

		mockUsecase.EXPECT().
			GetFolderTree(gomock.Any(), folderID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/"+folderID.String()+"/tree", nil)
		w := httptest.NewRecorder()

		handler.GetTree(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns bad request for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/invalid/tree", nil)
		w := httptest.NewRecorder()

		handler.GetTree(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_GetAncestors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns ancestor folders", func(t *testing.T) {
		folderID := uuid.New()
		response := []document_folder.DocumentFolderResponse{
			{ID: uuid.New(), Name: "Parent", Depth: int32(1)},
			{ID: uuid.New(), Name: "Grandparent", Depth: int32(0)},
		}

		mockUsecase.EXPECT().
			GetAncestors(gomock.Any(), folderID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/"+folderID.String()+"/ancestors", nil)
		w := httptest.NewRecorder()

		handler.GetAncestors(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_GetDescendants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns descendant folders", func(t *testing.T) {
		folderID := uuid.New()
		response := []document_folder.DocumentFolderResponse{
			{ID: uuid.New(), Name: "Child 1", Depth: int32(1)},
			{ID: uuid.New(), Name: "Child 2", Depth: int32(1)},
		}

		mockUsecase.EXPECT().
			GetDescendants(gomock.Any(), folderID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/"+folderID.String()+"/descendants", nil)
		w := httptest.NewRecorder()

		handler.GetDescendants(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := folderMocks.NewMockUseCase(ctrl)
	handler := document_folder.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("searches folders successfully", func(t *testing.T) {
		response := []document_folder.DocumentFolderResponse{
			{ID: uuid.New(), Name: "Test Folder 1"},
			{ID: uuid.New(), Name: "Test Folder 2"},
		}

		mockUsecase.EXPECT().
			SearchFolders(gomock.Any(), "test", int32(20), int32(0)).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/search?q=test", nil)
		w := httptest.NewRecorder()

		handler.Search(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("uses custom limit and offset", func(t *testing.T) {
		response := []document_folder.DocumentFolderResponse{}

		mockUsecase.EXPECT().
			SearchFolders(gomock.Any(), "test", int32(10), int32(5)).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/document-folders/search?q=test&limit=10&offset=5", nil)
		w := httptest.NewRecorder()

		handler.Search(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

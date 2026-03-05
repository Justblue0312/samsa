package document_test

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
	"github.com/justblue/samsa/internal/feature/document"
	documentMocks "github.com/justblue/samsa/internal/feature/document/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("creates document successfully", func(t *testing.T) {
		req := document.CreateDocumentRequest{
			StoryID:    uuid.New(),
			Language:   "eng",
			BranchName: "main",
			Title:      "Test Document",
			Slug:       "test-document",
		}
		jsonBody, _ := json.Marshal(req)

		response := &document.DocumentResponse{
			ID:        uuid.New(),
			StoryID:   req.StoryID,
			CreatedBy: testUser.ID,
			Title:     factory.StringPtr("Test Document"),
			Status:    document.DocumentStatusDraft,
		}

		mockUsecase.EXPECT().
			CreateDocument(gomock.Any(), testUser.ID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Test Document")
	})

	t.Run("returns unauthorized when no auth", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/documents", nil)
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHTTPHandler_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns document when found", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:     docID,
			Title:  factory.StringPtr("Test Document"),
			Status: document.DocumentStatusDraft,
		}

		mockUsecase.EXPECT().
			GetDocument(gomock.Any(), docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String(), nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns bad request for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodGet, "/documents/invalid", nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_GetBySlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns document by slug", func(t *testing.T) {
		response := &document.DocumentResponse{
			ID:    uuid.New(),
			Slug:  factory.StringPtr("test-slug"),
			Title: factory.StringPtr("Test Document"),
		}

		mockUsecase.EXPECT().
			GetDocumentBySlug(gomock.Any(), "test-slug").
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/documents/slug/test-slug", nil)
		w := httptest.NewRecorder()

		handler.GetBySlug(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("lists documents successfully", func(t *testing.T) {
		response := []document.DocumentResponse{
			{ID: uuid.New(), Title: factory.StringPtr("Doc 1")},
			{ID: uuid.New(), Title: factory.StringPtr("Doc 2")},
		}

		mockUsecase.EXPECT().
			ListDocuments(gomock.Any(), gomock.Any()).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/documents", nil)
		w := httptest.NewRecorder()

		handler.List(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("updates document successfully", func(t *testing.T) {
		docID := uuid.New()
		req := document.UpdateDocumentRequest{
			Title: factory.StringPtr("Updated Title"),
		}
		jsonBody, _ := json.Marshal(req)

		response := &document.DocumentResponse{
			ID:    docID,
			Title: factory.StringPtr("Updated Title"),
		}

		mockUsecase.EXPECT().
			UpdateDocument(gomock.Any(), testUser.ID, docID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/documents/"+docID.String(), bytes.NewReader(jsonBody))
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

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("deletes document successfully", func(t *testing.T) {
		docID := uuid.New()

		mockUsecase.EXPECT().
			DeleteDocument(gomock.Any(), testUser.ID, docID).
			Return(nil)

		httpReq := httptest.NewRequest(http.MethodDelete, "/documents/"+docID.String(), nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Delete(w, httpReq)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestHTTPHandler_SubmitForReview(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("submits document for review", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:     docID,
			Status: document.DocumentStatusPendingReview,
		}

		mockUsecase.EXPECT().
			SubmitForReview(gomock.Any(), testUser.ID, docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/submit", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.SubmitForReview(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Approve(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("approves document successfully", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:     docID,
			Status: document.DocumentStatusIsApproved,
		}

		mockUsecase.EXPECT().
			ApproveDocument(gomock.Any(), testUser.ID, docID, gomock.Any()).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/approve", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Approve(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Reject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("rejects document successfully", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:     docID,
			Status: document.DocumentStatusRejected,
		}

		mockUsecase.EXPECT().
			RejectDocument(gomock.Any(), testUser.ID, docID, gomock.Any()).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/reject", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Reject(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_Archive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("archives document successfully", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:     docID,
			Status: document.DocumentStatusArchived,
		}

		mockUsecase.EXPECT().
			ArchiveDocument(gomock.Any(), testUser.ID, docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/archive", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Archive(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_GetVersionHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns version history", func(t *testing.T) {
		docID := uuid.New()
		response := []document.DocumentResponse{
			{ID: docID, VersionNumber: 1},
			{ID: uuid.New(), VersionNumber: 2},
		}

		mockUsecase.EXPECT().
			GetVersionHistory(gomock.Any(), docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/versions", nil)
		w := httptest.NewRecorder()

		handler.GetVersionHistory(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_GetStatusHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns status history", func(t *testing.T) {
		docID := uuid.New()
		response := []sqlc.DocumentStatusHistory{
			{DocumentID: docID, Status: sqlc.DocumentStatusDraft},
			{DocumentID: docID, Status: sqlc.DocumentStatusPendingReview},
		}

		mockUsecase.EXPECT().
			GetStatusHistory(gomock.Any(), docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/documents/"+docID.String()+"/status-history", nil)
		w := httptest.NewRecorder()

		handler.GetStatusHistory(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_IncrementView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := documentMocks.NewMockUseCase(ctrl)
	handler := document.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("increments view count", func(t *testing.T) {
		docID := uuid.New()
		response := &document.DocumentResponse{
			ID:         docID,
			TotalViews: 1,
		}

		mockUsecase.EXPECT().
			IncrementView(gomock.Any(), docID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/documents/"+docID.String()+"/view", nil)
		w := httptest.NewRecorder()

		handler.IncrementView(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns bad request for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/documents/invalid/view", nil)
		w := httptest.NewRecorder()

		handler.IncrementView(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

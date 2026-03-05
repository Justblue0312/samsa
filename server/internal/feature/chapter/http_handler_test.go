package chapter_test

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
	"github.com/justblue/samsa/internal/feature/chapter"
	chapterMocks "github.com/justblue/samsa/internal/feature/chapter/mocks"
	"github.com/justblue/samsa/internal/testkit/factory"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("creates chapter successfully", func(t *testing.T) {
		storyID := uuid.New()
		req := chapter.CreateChapterRequest{
			StoryID: storyID,
			Title:   "Test Chapter",
		}
		jsonBody, _ := json.Marshal(req)

		response := &chapter.ChapterResponse{
			ID:        uuid.New(),
			StoryID:   storyID,
			Title:     "Test Chapter",
			Number:    factory.Int32Ptr(1),
			IsPublished: false,
		}

		mockUsecase.EXPECT().
			CreateChapter(gomock.Any(), storyID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/chapters", bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Test Chapter")
	})

	t.Run("returns unauthorized when no auth subject", func(t *testing.T) {
		req := chapter.CreateChapterRequest{
			StoryID: uuid.New(),
			Title:   "Test Chapter",
		}
		jsonBody, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/chapters", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns bad request for invalid JSON", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/chapters", bytes.NewReader([]byte("invalid")))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Create(w, httpReq)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestHTTPHandler_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("returns chapter when found", func(t *testing.T) {
		chapterID := uuid.New()
		response := &chapter.ChapterResponse{
			ID:        chapterID,
			Title:     "Test Chapter",
			StoryID:   uuid.New(),
			Number:    factory.Int32Ptr(1),
			IsPublished: false,
		}

		mockUsecase.EXPECT().
			GetChapter(gomock.Any(), chapterID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/chapters/"+chapterID.String(), nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Test Chapter")
	})

	t.Run("returns not found for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodGet, "/chapters/invalid-uuid", nil)
		w := httptest.NewRecorder()

		handler.GetByID(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("lists chapters successfully", func(t *testing.T) {
		storyID := uuid.New()
		response := []chapter.ChapterResponse{
			{ID: uuid.New(), StoryID: storyID, Title: "Chapter 1", Number: factory.Int32Ptr(1)},
			{ID: uuid.New(), StoryID: storyID, Title: "Chapter 2", Number: factory.Int32Ptr(2)},
		}

		mockUsecase.EXPECT().
			ListChapters(gomock.Any(), gomock.Any()).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodGet, "/chapters?story_id="+storyID.String(), nil)
		w := httptest.NewRecorder()

		handler.List(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Chapter 1")
	})
}

func TestHTTPHandler_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("updates chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		req := chapter.UpdateChapterRequest{
			Title: factory.StringPtr("Updated Title"),
		}
		jsonBody, _ := json.Marshal(req)

		response := &chapter.ChapterResponse{
			ID:        chapterID,
			Title:     "Updated Title",
			StoryID:   uuid.New(),
			IsPublished: false,
		}

		mockUsecase.EXPECT().
			UpdateChapter(gomock.Any(), testUser.ID, chapterID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/chapters/"+chapterID.String(), bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Update(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Updated Title")
	})
}

func TestHTTPHandler_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("deletes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()

		mockUsecase.EXPECT().
			DeleteChapter(gomock.Any(), testUser.ID, chapterID).
			Return(nil)

		httpReq := httptest.NewRequest(http.MethodDelete, "/chapters/"+chapterID.String(), nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Delete(w, httpReq)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestHTTPHandler_Publish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("publishes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		response := &chapter.ChapterResponse{
			ID:          chapterID,
			Title:       "Published Chapter",
			StoryID:     uuid.New(),
			IsPublished: true,
		}

		mockUsecase.EXPECT().
			PublishChapter(gomock.Any(), testUser.ID, chapterID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/chapters/"+chapterID.String()+"/publish", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Publish(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Published Chapter")
	})
}

func TestHTTPHandler_Unpublish(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("unpublishes chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		response := &chapter.ChapterResponse{
			ID:          chapterID,
			Title:       "Draft Chapter",
			StoryID:     uuid.New(),
			IsPublished: false,
		}

		mockUsecase.EXPECT().
			UnpublishChapter(gomock.Any(), testUser.ID, chapterID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/chapters/"+chapterID.String()+"/unpublish", nil)
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.Unpublish(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Draft Chapter")
	})
}

func TestHTTPHandler_Reorder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	subj := &subject.AuthSubject{
		User:   testUser,
		Scopes: []subject.Scope{subject.WebReadScope},
	}

	t.Run("reorders chapter successfully", func(t *testing.T) {
		chapterID := uuid.New()
		storyID := uuid.New()
		req := chapter.ReorderChapterRequest{
			StoryID:   storyID,
			SortOrder: 5,
		}
		jsonBody, _ := json.Marshal(req)

		response := &chapter.ChapterResponse{
			ID:        chapterID,
			Title:     "Test Chapter",
			StoryID:   storyID,
			SortOrder: factory.Int32Ptr(5),
		}

		mockUsecase.EXPECT().
			ReorderChapter(gomock.Any(), chapterID, req).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPatch, "/chapters/"+chapterID.String()+"/reorder", bytes.NewReader(jsonBody))
		ctx := context.WithValue(httpReq.Context(), common.AuthSubjectContextKey, subj)
		httpReq = httpReq.WithContext(ctx)
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Reorder(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandler_IncrementView(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := chapterMocks.NewMockUseCase(ctrl)
	handler := chapter.NewHTTPHandler(mockUsecase, nil, nil)

	t.Run("increments view count", func(t *testing.T) {
		chapterID := uuid.New()
		response := &chapter.ChapterResponse{
			ID:        chapterID,
			Title:     "Test Chapter",
			StoryID:   uuid.New(),
			TotalViews: 1,
		}

		mockUsecase.EXPECT().
			IncrementView(gomock.Any(), chapterID).
			Return(response, nil)

		httpReq := httptest.NewRequest(http.MethodPost, "/chapters/"+chapterID.String()+"/view", nil)
		w := httptest.NewRecorder()

		handler.IncrementView(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns bad request for invalid UUID", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodPost, "/chapters/invalid-uuid/view", nil)
		w := httptest.NewRecorder()

		handler.IncrementView(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

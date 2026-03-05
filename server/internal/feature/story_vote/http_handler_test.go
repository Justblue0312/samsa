package story_vote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/story_vote"
	"github.com/justblue/samsa/internal/feature/story_vote/mocks"
	"github.com/justblue/samsa/pkg/subject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHTTPHandler_CreateVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	validator := validator.New()
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, validator)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	testStoryID := uuid.New()

	t.Run("creates vote successfully", func(t *testing.T) {
		reqBody := story_vote.CreateVoteRequest{
			StoryID: testStoryID,
			Rating:  5,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/story-votes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)

		respVote := &story_vote.VoteResponse{
			ID:      uuid.New(),
			StoryID: testStoryID,
			UserID:  testUser.ID,
			Rating:  5,
		}

		mockUsecase.EXPECT().
			CreateVote(gomock.Any(), testUser.ID, reqBody).
			Return(respVote, nil)

		w := httptest.NewRecorder()
		handler.CreateVote(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp story_vote.VoteResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, respVote.ID, resp.ID)
		assert.Equal(t, respVote.Rating, resp.Rating)
	})

	t.Run("returns unprocessable entity on invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/story-votes", bytes.NewReader([]byte("invalid")))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateVote(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("returns unprocessable entity on validation error", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"story_id": "invalid-uuid",
			"rating":   5,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/story-votes", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateVote(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("returns not found when story not found", func(t *testing.T) {
		reqBody := story_vote.CreateVoteRequest{
			StoryID: testStoryID,
			Rating:  5,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/story-votes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)

		mockUsecase.EXPECT().
			CreateVote(gomock.Any(), testUser.ID, reqBody).
			Return(nil, story_vote.ErrNotFound)

		w := httptest.NewRecorder()
		handler.CreateVote(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHTTPHandler_GetVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testVoteID := uuid.New()

	t.Run("retrieves vote successfully", func(t *testing.T) {
		respVote := &story_vote.VoteResponse{
			ID:     testVoteID,
			Rating: 4,
		}

		mockUsecase.EXPECT().
			GetVote(gomock.Any(), testVoteID).
			Return(respVote, nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("vote_id", testVoteID.String())

		req := httptest.NewRequest(http.MethodGet, "/story-votes/"+testVoteID.String(), nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.GetVote(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp story_vote.VoteResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, testVoteID, resp.ID)
	})

	t.Run("returns bad request on invalid uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("vote_id", "invalid")

		req := httptest.NewRequest(http.MethodGet, "/story-votes/invalid", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.GetVote(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns not found when vote not found", func(t *testing.T) {
		mockUsecase.EXPECT().
			GetVote(gomock.Any(), testVoteID).
			Return(nil, story_vote.ErrNotFound)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("vote_id", testVoteID.String())

		req := httptest.NewRequest(http.MethodGet, "/story-votes/"+testVoteID.String(), nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.GetVote(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHTTPHandler_GetUserVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	testStoryID := uuid.New()

	t.Run("retrieves user's vote successfully", func(t *testing.T) {
		respVote := &story_vote.VoteResponse{
			ID:      uuid.New(),
			StoryID: testStoryID,
			UserID:  testUser.ID,
			Rating:  5,
		}

		mockUsecase.EXPECT().
			GetUserVote(gomock.Any(), testStoryID, testUser.ID).
			Return(respVote, nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", testStoryID.String())

		req := httptest.NewRequest(http.MethodGet, "/stories/"+testStoryID.String()+"/my-vote", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebReadScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUserVote(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp story_vote.VoteResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, testStoryID, resp.StoryID)
		assert.Equal(t, int32(5), resp.Rating)
	})

	t.Run("returns not found when no vote exists", func(t *testing.T) {
		mockUsecase.EXPECT().
			GetUserVote(gomock.Any(), testStoryID, testUser.ID).
			Return(nil, nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", testStoryID.String())

		req := httptest.NewRequest(http.MethodGet, "/stories/"+testStoryID.String()+"/my-vote", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebReadScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUserVote(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns bad request on invalid story uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", "invalid")

		req := httptest.NewRequest(http.MethodGet, "/stories/invalid/my-vote", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebReadScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.GetUserVote(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_DeleteUserVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testUser := &sqlc.User{
		ID:    uuid.New(),
		Email: "test@example.com",
	}
	testStoryID := uuid.New()

	t.Run("deletes vote successfully", func(t *testing.T) {
		mockUsecase.EXPECT().
			DeleteUserVote(gomock.Any(), testStoryID, testUser.ID).
			Return(nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", testStoryID.String())

		req := httptest.NewRequest(http.MethodDelete, "/stories/"+testStoryID.String()+"/vote", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteUserVote(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns bad request on invalid story uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", "invalid")

		req := httptest.NewRequest(http.MethodDelete, "/stories/invalid/vote", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		ctx := context.WithValue(req.Context(), common.AuthSubjectContextKey, &subject.AuthSubject{
			User:   testUser,
			Scopes: []subject.Scope{subject.WebWriteScope},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.DeleteUserVote(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_GetVoteStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testStoryID := uuid.New()

	t.Run("retrieves vote stats successfully", func(t *testing.T) {
		respStats := &story_vote.VoteStatsResponse{
			StoryID:       testStoryID,
			TotalVotes:    10,
			AverageRating: 4.5,
		}

		mockUsecase.EXPECT().
			GetVoteStats(gomock.Any(), testStoryID).
			Return(respStats, nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", testStoryID.String())

		req := httptest.NewRequest(http.MethodGet, "/stories/"+testStoryID.String()+"/vote-stats", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.GetVoteStats(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp story_vote.VoteStatsResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, testStoryID, resp.StoryID)
		assert.Equal(t, int64(10), resp.TotalVotes)
		assert.InDelta(t, 4.5, resp.AverageRating, 0.01)
	})

	t.Run("returns bad request on invalid story uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", "invalid")

		req := httptest.NewRequest(http.MethodGet, "/stories/invalid/vote-stats", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.GetVoteStats(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_ListStoryVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testStoryID := uuid.New()

	t.Run("lists votes successfully", func(t *testing.T) {
		votes := []story_vote.VoteResponse{
			{ID: uuid.New(), StoryID: testStoryID, Rating: 5},
			{ID: uuid.New(), StoryID: testStoryID, Rating: 4},
		}

		mockUsecase.EXPECT().
			ListStoryVotes(gomock.Any(), testStoryID, gomock.Any()).
			Return(votes, int64(2), nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", testStoryID.String())

		req := httptest.NewRequest(http.MethodGet, "/stories/"+testStoryID.String()+"/votes", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.ListStoryVotes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "votes")
		assert.Contains(t, resp, "meta")
	})

	t.Run("returns bad request on invalid story uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("story_id", "invalid")

		req := httptest.NewRequest(http.MethodGet, "/stories/invalid/votes", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.ListStoryVotes(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTPHandler_ListUserVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUsecase := mocks.NewMockUseCase(ctrl)
	handler := story_vote.NewHTTPHandler(mockUsecase, nil, nil)

	testUserID := uuid.New()

	t.Run("lists user votes successfully", func(t *testing.T) {
		votes := []story_vote.VoteResponse{
			{ID: uuid.New(), UserID: testUserID, Rating: 5},
			{ID: uuid.New(), UserID: testUserID, Rating: 4},
		}

		mockUsecase.EXPECT().
			ListUserVotes(gomock.Any(), testUserID, gomock.Any()).
			Return(votes, int64(2), nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("user_id", testUserID.String())

		req := httptest.NewRequest(http.MethodGet, "/users/"+testUserID.String()+"/votes", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.ListUserVotes(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "votes")
		assert.Contains(t, resp, "meta")
	})

	t.Run("returns bad request on invalid user uuid", func(t *testing.T) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("user_id", "invalid")

		req := httptest.NewRequest(http.MethodGet, "/users/invalid/votes", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		handler.ListUserVotes(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

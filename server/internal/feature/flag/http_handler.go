package flag

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/pkg/apierror"
	"github.com/justblue/samsa/pkg/queryparam"
	"github.com/justblue/samsa/pkg/respond"
)

//go:generate mockgen -destination=mocks/mock_http_handler.go -source=http_handler.go -package=mocks

type HTTPHandler struct {
	u         UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func NewHTTPHandler(u UseCase, cfg *config.Config, v *validator.Validate) *HTTPHandler {
	return &HTTPHandler{
		u:         u,
		cfg:       cfg,
		validator: v,
	}
}

func mapError(w http.ResponseWriter, err error) {
	switch {
	case err == ErrFlagNotFound:
		respond.Error(w, apierror.NotFound("flag not found"))
	case err == ErrStoryNotFound:
		respond.Error(w, apierror.NotFound("story not found"))
	case err == ErrChapterNotFound:
		respond.Error(w, apierror.NotFound("chapter not found"))
	case err == ErrPermissionDenied:
		respond.Error(w, apierror.Forbidden())
	case err == ErrInvalidFlagScore:
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
	default:
		respond.Error(w, apierror.Internal())
	}
}

// Create creates a new flag on a story or chapter.
//
//	@Summary		Create flag
//	@Description	Creates a new flag for content moderation. Requires moderator/inspector role.
//	@Tags			flags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateFlagRequest	true	"Flag creation request"
//	@Success		201		{object}	FlagResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		403		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Failure		422		{object}	apierror.APIError
//	@Router			/admin/flags [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	var req CreateFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.CreateFlag(ctx, sub.User.ID, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.JSON(w, http.StatusCreated, resp)
}

// List retrieves flags with optional filters.
//
//	@Summary		List flags
//	@Description	Retrieves flags with filtering options. Requires moderator/inspector role.
//	@Tags			flags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id		query		string	false	"Filter by story UUID"
//	@Param			chapter_id		query		string	false	"Filter by chapter UUID"
//	@Param			inspector_id	query		string	false	"Filter by inspector UUID"
//	@Param			flag_type		query		string	false	"Filter by flag type"
//	@Param			flag_rate		query		string	false	"Filter by flag rate"
//	@Param			min_score		query		float64	false	"Minimum flag score"
//	@Param			max_score		query		float64	false	"Maximum flag score"
//	@Param			limit			query		int		false	"Limit"
//	@Param			offset			query		int		false	"Offset"
//	@Success		200				{object}	FlagListResponse
//	@Failure		401				{object}	apierror.APIError
//	@Failure		403				{object}	apierror.APIError
//	@Router			/admin/flags [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	var params ListFlagsParams
	if err := decodeListFlagsParams(r, &params); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.ListFlags(ctx, params)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// GetByID retrieves a flag by ID.
//
//	@Summary		Get flag by ID
//	@Description	Retrieves a flag by its UUID. Requires moderator/inspector role.
//	@Tags			flags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			flag_id	path		string	true	"Flag UUID"
//	@Success		200		{object}	FlagResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Router			/admin/flags/{flag_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "flag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid flag id"))
		return
	}

	resp, err := h.u.GetFlag(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Update updates an existing flag.
//
//	@Summary		Update flag
//	@Description	Updates flag details. Requires moderator/inspector role.
//	@Tags			flags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			flag_id	path		string				true	"Flag UUID"
//	@Param			request	body		UpdateFlagRequest	true	"Flag update request"
//	@Success		200		{object}	FlagResponse
//	@Failure		401		{object}	apierror.APIError
//	@Failure		404		{object}	apierror.APIError
//	@Router			/admin/flags/{flag_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "flag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid flag id"))
		return
	}

	var req UpdateFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respond.Error(w, apierror.UnprocessableEntity(err.Error()))
		return
	}

	resp, err := h.u.UpdateFlag(ctx, id, req)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.OK(w, resp)
}

// Delete deletes a flag.
//
//	@Summary		Delete flag
//	@Description	Deletes a flag. Requires moderator/inspector role.
//	@Tags			flags
//	@Security		BearerAuth
//	@Param			flag_id	path	string	true	"Flag UUID"
//	@Success		204
//	@Failure		401	{object}	apierror.APIError
//	@Failure		404	{object}	apierror.APIError
//	@Router			/admin/flags/{flag_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "flag_id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid flag id"))
		return
	}

	err = h.u.DeleteFlag(ctx, id)
	if err != nil {
		mapError(w, err)
		return
	}

	respond.NoContent(w)
}

func decodeListFlagsParams(r *http.Request, params *ListFlagsParams) error {
	query := r.URL.Query()

	if storyIDStr := query.Get("story_id"); storyIDStr != "" {
		storyID, err := uuid.Parse(storyIDStr)
		if err != nil {
			return err
		}
		params.StoryID = &storyID
	}

	if chapterIDStr := query.Get("chapter_id"); chapterIDStr != "" {
		chapterID, err := uuid.Parse(chapterIDStr)
		if err != nil {
			return err
		}
		params.ChapterID = &chapterID
	}

	if inspectorIDStr := query.Get("inspector_id"); inspectorIDStr != "" {
		inspectorID, err := uuid.Parse(inspectorIDStr)
		if err != nil {
			return err
		}
		params.InspectorID = &inspectorID
	}

	if flagTypeStr := query.Get("flag_type"); flagTypeStr != "" {
		flagType := FlagType(flagTypeStr)
		params.FlagType = &flagType
	}

	if flagRateStr := query.Get("flag_rate"); flagRateStr != "" {
		flagRate := FlagRate(flagRateStr)
		params.FlagRate = &flagRate
	}

	if minScoreStr := query.Get("min_score"); minScoreStr != "" {
		var minScore float64
		if _, err := fmt.Sscanf(minScoreStr, "%f", &minScore); err != nil {
			return err
		}
		params.MinScore = &minScore
	}

	if maxScoreStr := query.Get("max_score"); maxScoreStr != "" {
		var maxScore float64
		if _, err := fmt.Sscanf(maxScoreStr, "%f", &maxScore); err != nil {
			return err
		}
		params.MaxScore = &maxScore
	}

	if pageStr := query.Get("page"); pageStr != "" {
		var page int32
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
			return err
		}
		if page > 0 {
			params.Limit = (page - 1) * 10
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		var limit int32
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			return err
		}
		params.Limit = limit
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		var offset int32
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			return err
		}
		params.Offset = offset
	}

	return nil
}

// ListByStory retrieves flags for a specific story.
//
//	@Summary		List flags for a story
//	@Description	Retrieves all flags for a specific story. Requires moderator/inspector role.
//	@Tags			flags
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			story_id	path		string	true	"Story UUID"
//	@Param			page		query		int		false	"Page number"
//	@Param			limit		query		int		false	"Limit per page"
//	@Success		200			{object}	FlagListResponse
//	@Failure		401			{object}	apierror.APIError
//	@Failure		403			{object}	apierror.APIError
//	@Failure		404			{object}	apierror.APIError
//	@Router			/stories/{story_id}/flags [get]
func (h *HTTPHandler) ListByStory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sub, err := common.GetUserSubject(ctx)
	if err != nil {
		respond.Error(w, apierror.Unauthorized())
		return
	}

	if !sub.IsModerator() {
		respond.Error(w, apierror.Forbidden())
		return
	}

	idStr := chi.URLParam(r, "story_id")
	storyID, err := uuid.Parse(idStr)
	if err != nil {
		respond.Error(w, apierror.BadRequest("invalid story id"))
		return
	}

	// Parse pagination params
	query := r.URL.Query()
	page := int32(1)
	limit := int32(10)

	if pageStr := query.Get("page"); pageStr != "" {
		var p int32
		if _, err := fmt.Sscanf(pageStr, "%d", &p); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		var l int32
		if _, err := fmt.Sscanf(limitStr, "%d", &l); err == nil && l > 0 {
			limit = l
		}
	}

	flags, total, err := h.u.ListFlagsByStory(ctx, storyID, page, limit)
	if err != nil {
		mapError(w, err)
		return
	}

	resp := FlagListResponse{
		Flags: flags,
		Meta:  queryparam.NewPaginationMeta(page, limit, total),
	}

	respond.OK(w, resp)
}

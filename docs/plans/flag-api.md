# Flag API Implementation Plan

**Date:** 2026-03-04  
**Feature:** Content Moderation Flags for Stories/Chapters

## Overview

Implement a Flag API to allow **inspectors/moderators** to place flags on stories or chapters for content moderation purposes. 

**Key distinction from `story_report`:**
- `story_report`: User-submitted reports about violations (pending review)
- `flag`: Inspector/moderator-created flags with severity scoring and rate classification

The existing schema provides:
- **flag_types**: `spam`, `inappropriate`, `copyright`, `plagiarism`, `harassment`, `hate_speech`, `self_harm`, `explicit`, `privacy`, `misinformation`, `other`
- **flag_rate**: `low`, `medium`, `high`, `critical`
- **flag_score**: Float score for prioritization
- **inspector_id**: Links to moderator who created the flag

## Architecture

Follow the existing feature pattern in the codebase:
- `models.go` - Request/Response types and converters
- `repository.go` - Database operations interface and implementation
- `usecase.go` - Business logic interface and implementation
- `http_handler.go` - HTTP handlers with Swagger documentation
- `register.go` - Route registration

## Existing Database Schema

```sql
-- Already exists in 20260301055343_initialize_story_models.sql
CREATE TYPE flag_types AS ENUM ('spam', 'inappropriate', 'copyright', 'plagiarism', 'harassment', 'hate_speech', 'self_harm', 'explicit', 'privacy', 'misinformation', 'other');
CREATE TYPE flag_rate AS ENUM ('low', 'medium', 'high', 'critical');

CREATE TABLE flag (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    inspector_id UUID REFERENCES "user"(id) ON DELETE CASCADE,
    title CHAR(255) NOT NULL,
    description TEXT,
    flag_type flag_types NOT NULL,
    flag_rate flag_rate NOT NULL,
    flag_score FLOAT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Schema Enhancement (Optional Migration)

Consider adding these fields for better flag lifecycle management:

```sql
-- Optional: Add status tracking for flags
ALTER TABLE flag ADD COLUMN status report_status DEFAULT 'pending';
ALTER TABLE flag ADD COLUMN is_resolved BOOLEAN DEFAULT FALSE;
ALTER TABLE flag ADD COLUMN resolved_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE flag ADD COLUMN resolved_by UUID REFERENCES "user"(id) ON DELETE SET NULL;

-- Optional: Add soft delete for audit trail
ALTER TABLE flag ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- New indexes
CREATE INDEX idx_flags_status ON flag(status);
CREATE INDEX idx_flags_chapter_id ON flag(chapter_id) WHERE chapter_id IS NOT NULL;
```

## API Endpoints

All flag operations require **moderator/inspector** authentication.

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/admin/flags` | Create a flag on story/chapter | Yes (moderator) |
| GET | `/admin/flags` | List flags with filters | Yes (moderator) |
| GET | `/admin/flags/{flag_id}` | Get a specific flag | Yes (moderator) |
| PATCH | `/admin/flags/{flag_id}` | Update a flag | Yes (moderator) |
| DELETE | `/admin/flags/{flag_id}` | Remove a flag | Yes (moderator) |

### Optional: Public/Story Owner Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/stories/{story_id}/flags` | Get flags for a story (owner view) | Yes (story owner) |
| GET | `/chapters/{chapter_id}/flags` | Get flags for a chapter (owner view) | Yes (story owner) |

## Implementation Tasks

### 1. Models (`models.go`)

Define:
- `FlagType` enum type (spam, inappropriate, copyright, plagiarism, harassment, hate_speech, self_harm, explicit, privacy, misinformation, other)
- `FlagRate` enum type (low, medium, high, critical)
- `CreateFlagRequest` - with validation
- `UpdateFlagRequest` - partial update
- `FlagResponse` - API response structure
- `FlagListResponse` - paginated list response
- `ToFlagResponse()` converter function

```go
type FlagType string
const (
    FlagTypeSpam          FlagType = "spam"
    FlagTypeInappropriate FlagType = "inappropriate"
    FlagTypeCopyright     FlagType = "copyright"
    FlagTypePlagiarism    FlagType = "plagiarism"
    FlagTypeHarassment    FlagType = "harassment"
    FlagTypeHateSpeech    FlagType = "hate_speech"
    FlagTypeSelfHarm      FlagType = "self_harm"
    FlagTypeExplicit      FlagType = "explicit"
    FlagTypePrivacy       FlagType = "privacy"
    FlagTypeMisinformation FlagType = "misinformation"
    FlagTypeOther         FlagType = "other"
)

type FlagRate string
const (
    FlagRateLow      FlagRate = "low"
    FlagRateMedium   FlagRate = "medium"
    FlagRateHigh     FlagRate = "high"
    FlagRateCritical FlagRate = "critical"
)

type CreateFlagRequest struct {
    StoryID    uuid.UUID  `json:"story_id" validate:"required,uuid"`
    ChapterID  *uuid.UUID `json:"chapter_id,omitempty" validate:"omitempty,uuid"`
    Title      string     `json:"title" validate:"required,max=255"`
    Description *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
    FlagType   FlagType   `json:"flag_type" validate:"required,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
    FlagRate   FlagRate   `json:"flag_rate" validate:"required,oneof=low medium high critical"`
    FlagScore  float64    `json:"flag_score" validate:"required,min=0,max=100"`
}

type UpdateFlagRequest struct {
    Title       *string  `json:"title,omitempty" validate:"omitempty,max=255"`
    Description *string  `json:"description,omitempty" validate:"omitempty,max=2000"`
    FlagType    *FlagType `json:"flag_type,omitempty" validate:"omitempty,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
    FlagRate    *FlagRate `json:"flag_rate,omitempty" validate:"omitempty,oneof=low medium high critical"`
    FlagScore   *float64  `json:"flag_score,omitempty" validate:"omitempty,min=0,max=100"`
}

type FlagResponse struct {
    ID          uuid.UUID  `json:"id"`
    StoryID     uuid.UUID  `json:"story_id"`
    ChapterID   *uuid.UUID `json:"chapter_id,omitempty"`
    InspectorID *uuid.UUID `json:"inspector_id,omitempty"`
    Title       string     `json:"title"`
    Description *string    `json:"description,omitempty"`
    FlagType    FlagType   `json:"flag_type"`
    FlagRate    FlagRate   `json:"flag_rate"`
    FlagScore   float64    `json:"flag_score"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

type FlagListResponse struct {
    Flags []FlagResponse          `json:"flags"`
    Meta  queryparam.PaginationMeta `json:"meta"`
}
```

### 2. Repository (`repository.go`)

Interface methods:

```go
type Repository interface {
    Create(ctx context.Context, arg sqlc.CreateFlagParams) (*sqlc.Flag, error)
    GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Flag, error)
    ListByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]sqlc.Flag, error)
    ListByChapter(ctx context.Context, chapterID uuid.UUID, limit, offset int32) ([]sqlc.Flag, error)
    ListByInspector(ctx context.Context, inspectorID uuid.UUID, limit, offset int32) ([]sqlc.Flag, error)
    ListAll(ctx context.Context, params ListFlagsParams) ([]sqlc.Flag, error)
    Update(ctx context.Context, arg sqlc.UpdateFlagParams) (*sqlc.Flag, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

type ListFlagsParams struct {
    StoryID     *uuid.UUID `json:"story_id,omitempty"`
    ChapterID   *uuid.UUID `json:"chapter_id,omitempty"`
    InspectorID *uuid.UUID `json:"inspector_id,omitempty"`
    FlagType    *FlagType  `json:"flag_type,omitempty"`
    FlagRate    *FlagRate  `json:"flag_rate,omitempty"`
    MinScore    *float64   `json:"min_score,omitempty"`
    MaxScore    *float64   `json:"max_score,omitempty"`
    Limit       int32
    Offset      int32
}
```

### 3. UseCase (`usecase.go`)

Business logic:

```go
type UseCase interface {
    CreateFlag(ctx context.Context, inspectorID uuid.UUID, req CreateFlagRequest) (*FlagResponse, error)
    GetFlag(ctx context.Context, id uuid.UUID) (*FlagResponse, error)
    ListFlags(ctx context.Context, params ListFlagsParams) (*FlagListResponse, error)
    UpdateFlag(ctx context.Context, id uuid.UUID, req UpdateFlagRequest) (*FlagResponse, error)
    DeleteFlag(ctx context.Context, id uuid.UUID) error
}
```

**Business Rules:**
- `CreateFlag`: Verify story/chapter exists, validate flag_score is within range (0-100)
- `UpdateFlag`: Allow partial updates, validate score if provided
- `DeleteFlag`: Soft delete if `deleted_at` column exists, otherwise hard delete
- `ListFlags`: Support filtering by story, chapter, inspector, type, rate, and score range

### 4. HTTP Handler (`http_handler.go`)

Handlers with Swagger docs:

```go
// CreateFlag creates a new flag on a story or chapter.
// @Summary      Create flag
// @Description  Creates a new flag for content moderation. Requires moderator/inspector role.
// @Tags         flags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateFlagRequest  true  "Flag creation request"
// @Success      201      {object}  FlagResponse
// @Failure      401      {object}  apierror.APIError
// @Failure      403      {object}  apierror.APIError
// @Failure      404      {object}  apierror.APIError
// @Failure      422      {object}  apierror.APIError
// @Router       /admin/flags [post]
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request)

// ListFlags retrieves flags with optional filters.
// @Summary      List flags
// @Description  Retrieves flags with filtering options. Requires moderator/inspector role.
// @Tags         flags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        story_id     query  string  false  "Filter by story UUID"
// @Param        chapter_id   query  string  false  "Filter by chapter UUID"
// @Param        inspector_id query  string  false  "Filter by inspector UUID"
// @Param        flag_type    query  string  false  "Filter by flag type"
// @Param        flag_rate    query  string  false  "Filter by flag rate"
// @Param        limit        query  int     false  "Limit"
// @Param        offset       query  int     false  "Offset"
// @Success      200          {object}  FlagListResponse
// @Failure      401          {object}  apierror.APIError
// @Router       /admin/flags [get]
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request)

// GetByID retrieves a flag by ID.
// @Summary      Get flag by ID
// @Description  Retrieves a flag by its UUID. Requires moderator/inspector role.
// @Tags         flags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        flag_id  path  string  true  "Flag UUID"
// @Success      200      {object}  FlagResponse
// @Failure      401      {object}  apierror.APIError
// @Failure      404      {object}  apierror.APIError
// @Router       /admin/flags/{flag_id} [get]
func (h *HTTPHandler) GetByID(w http.ResponseWriter, r *http.Request)

// Update updates an existing flag.
// @Summary      Update flag
// @Description  Updates flag details. Requires moderator/inspector role.
// @Tags         flags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        flag_id  path  string              true  "Flag UUID"
// @Param        request  body  UpdateFlagRequest   true  "Flag update request"
// @Success      200      {object}  FlagResponse
// @Failure      401      {object}  apierror.APIError
// @Failure      404      {object}  apierror.APIError
// @Router       /admin/flags/{flag_id} [patch]
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request)

// Delete soft deletes a flag.
// @Summary      Delete flag
// @Description  Deletes a flag. Requires moderator/inspector role.
// @Tags         flags
// @Security     BearerAuth
// @Param        flag_id  path  string  true  "Flag UUID"
// @Success      204
// @Failure      401      {object}  apierror.APIError
// @Failure      404      {object}  apierror.APIError
// @Router       /admin/flags/{flag_id} [delete]
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request)
```

Error handling:
- `ErrFlagNotFound` - 404
- `ErrStoryNotFound` - 404 (when creating flag)
- `ErrChapterNotFound` - 404 (when creating flag with chapter_id)
- `ErrPermissionDenied` - 403 (non-moderator access)

### 5. Routes (`register.go`)

```go
func RegisterFlagRoutes(router chi.Router, h *HTTPHandler, authMW middleware) {
    // All flag routes require moderator/inspector authentication
    router.Route("/admin/flags", func(r chi.Router) {
        r.Use(authMW.Required)
        r.Use(authMW.RequireModerator) // or RequireInspector based on your role system
        
        r.Get("/", h.List)
        r.Post("/", h.Create)
        
        r.Route("/{flag_id}", func(r chi.Router) {
            r.Get("/", h.GetByID)
            r.Patch("/", h.Update)
            r.Delete("/", h.Delete)
        })
    })
}
```

## Validation Rules

### CreateFlagRequest
```go
type CreateFlagRequest struct {
    StoryID     uuid.UUID  `json:"story_id" validate:"required,uuid"`
    ChapterID   *uuid.UUID `json:"chapter_id,omitempty" validate:"omitempty,uuid"`
    Title       string     `json:"title" validate:"required,max=255"`
    Description *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
    FlagType    FlagType   `json:"flag_type" validate:"required,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
    FlagRate    FlagRate   `json:"flag_rate" validate:"required,oneof=low medium high critical"`
    FlagScore   float64    `json:"flag_score" validate:"required,min=0,max=100"`
}
```

### UpdateFlagRequest
```go
type UpdateFlagRequest struct {
    Title       *string    `json:"title,omitempty" validate:"omitempty,max=255"`
    Description *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
    FlagType    *FlagType  `json:"flag_type,omitempty" validate:"omitempty,oneof=spam inappropriate copyright plagiarism harassment hate_speech self_harm explicit privacy misinformation other"`
    FlagRate    *FlagRate  `json:"flag_rate,omitempty" validate:"omitempty,oneof=low medium high critical"`
    FlagScore   *float64   `json:"flag_score,omitempty" validate:"omitempty,min=0,max=100"`
}
```

## Response Format

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "story_id": "550e8400-e29b-41d4-a716-446655440001",
  "chapter_id": null,
  "inspector_id": "550e8400-e29b-41d4-a716-446655440002",
  "title": "Explicit Content Warning",
  "description": "This story contains explicit violence in chapter 3",
  "flag_type": "explicit",
  "flag_rate": "high",
  "flag_score": 85.5,
  "created_at": "2026-03-04T10:00:00Z",
  "updated_at": "2026-03-04T10:00:00Z"
}
```

## List Response Format

```json
{
  "flags": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "story_id": "550e8400-e29b-41d4-a716-446655440001",
      "flag_type": "explicit",
      "flag_rate": "high",
      "flag_score": 85.5,
      "created_at": "2026-03-04T10:00:00Z",
      "updated_at": "2026-03-04T10:00:00Z"
    }
  ],
  "meta": {
    "limit": 10,
    "offset": 0,
    "total": 1
  }
}
```

## Integration Points

1. **Story/Chapter Display**: Consider exposing flag information to story owners (not public)
2. **Moderation Dashboard**: Flags should appear in admin moderation queue, sorted by `flag_rate` and `flag_score`
3. **User Permissions**: Verify inspector has moderator/inspector role before allowing flag operations
4. **Notifications**: Optionally notify story owner when a flag is placed on their content

## Testing Strategy

1. **Unit Tests**:
   - UseCase logic (validation, score bounds)
   - Repository methods (with mocked DB)
   
2. **Integration Tests**:
   - HTTP handlers with test server
   - End-to-end flag CRUD operations
   
3. **Test Cases**:
   - Create flag on story
   - Create flag on chapter
   - Cannot create flag with invalid score (< 0 or > 100)
   - Cannot flag non-existent story/chapter
   - Moderator can update/delete flags
   - Non-moderator cannot access flag endpoints
   - Filter flags by type, rate, story, chapter
   - Pagination for flag lists

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `server/internal/feature/flag/models.go` | Modify | Add all types and structs |
| `server/internal/feature/flag/repository.go` | Modify | Add interface and implementation |
| `server/internal/feature/flag/usecase.go` | Modify | Add business logic |
| `server/internal/feature/flag/http_handler.go` | Modify | Add HTTP handlers |
| `server/internal/feature/flag/register.go` | Modify | Add route registration |
| `server/internal/feature/flag/errors.go` | Create | Custom error types |
| `server/internal/feature/flag/mocks/` | Generate | Mockgen output |
| `server/db/migrations/` | Optional | Add migration for status/deleted_at columns |
| `server/internal/handler/routes.go` | Modify | Register flag routes |

## Prerequisites

Before starting implementation:

1. ✅ Verify SQLC queries exist for `flag` table operations
2. ✅ Run `sqlc generate` if adding new queries
3. ✅ Ensure middleware for role-based access exists (RequireModerator/Inspector)

## Rollout Plan

1. **Phase 1**: Core CRUD operations (moderators can create/manage flags)
2. **Phase 2**: Filtering and pagination for flag lists
3. **Phase 3**: Integration with moderation dashboard
4. **Phase 4**: Optional schema enhancements (status tracking, soft delete)

## Notes

- Flags are **moderator-only** operations (unlike user-submitted reports)
- `flag_score` (0-100) allows prioritization of moderation queue
- `flag_rate` provides categorical severity (low/medium/high/critical)
- Consider adding `status` column for flag lifecycle tracking (pending → resolved/rejected)
- The `inspector_id` is nullable in schema - allow anonymous system-generated flags if needed

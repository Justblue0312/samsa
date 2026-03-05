# Samsa Writing Platform - API Specification

## Overview

The Samsa Writing Platform API is a RESTful service that provides comprehensive functionality for authors, readers, and administrators to manage written content, social interactions, and community moderation. The API follows resource-oriented design principles with proper HTTP semantics and is organized using feature-based routing.

### API Architecture

- **Base URL**: `/api/v1/`
- **Content-Type**: `application/json`
- **Error Format**: `application/problem+json`
- **Authentication**: Cookie-based sessions with JWT tokens
- **Framework**: Chi v5 router with Go backend
- **Database**: PostgreSQL with proper indexing and constraints
- **Cache**: Redis for session management and caching

### Design Principles

1. **Resource-Oriented**: URLs represent nouns (users, stories, chapters)
2. **HTTP Semantics**: Proper use of HTTP methods and status codes
3. **Consistent Responses**: Standardized response formats across all endpoints
4. **Security First**: Authentication, authorization, and input validation
5. **Scalable Architecture**: Feature-based organization with clear boundaries

---

## Authentication & Authorization

### Authentication Flow

The API uses cookie-based authentication with JWT tokens stored in secure, HttpOnly cookies:

1. **Login**: User provides credentials, receives session cookie
2. **Session Validation**: Middleware validates JWT on each request
3. **Session Refresh**: Automatic token refresh for active sessions
4. **Logout**: Cookie cleared and session invalidated

### Authorization Strategy

#### Scope-Based Access Control

```go
// Core Scopes
const (
    WebReadScope      Scope = "web:read"      // Basic platform access
    WebWriteScope     Scope = "web:write"     // Basic interactions

    // Story Management
    StoryReadScope    Scope = "story:read"    // Read stories
    StoryWriteScope   Scope = "story:write"   // Create/edit stories
    StoryPublishScope Scope = "story:publish" // Publish content

    // Content Management
    FileReadScope     Scope = "file:read"     // Access files
    FileWriteScope    Scope = "file:write"    // Upload/manage files

    // Social Features
    CommentReadScope  Scope = "comment:read"  // Read comments
    CommentWriteScope Scope = "comment:write" // Create comments

    // Moderation
    ModerateScope     Scope = "moderate:content" // Content moderation
    AdminScope        Scope = "admin:system"    // System administration
)
```

#### Role-Based Permissions

| Role          | Default Scopes                     | Additional Permissions  |
| ------------- | ---------------------------------- | ----------------------- |
| **Anonymous** | `web:read`                         | Read published content  |
| **User**      | `web:read`, `web:write`            | Comment, vote, bookmark |
| **Author**    | User + `story:read`, `story:write` | Manage own stories      |
| **Moderator** | Author + `moderate:content`        | Content review          |
| **Admin**     | All scopes                         | Full system access      |

---

## Request & Response Formats

### Standard Response Format

```json
{
    "data": {}, // Response data or null
    "message": "Success message",
    "status": "success",
    "timestamp": "2026-01-14T10:30:00Z"
}
```

### Error Response Format

```json
{
    "message": "Error description",
    "details": {
        "field": "Validation error message"
    },
    "code": "VALIDATION_ERROR",
    "status": "error",
    "timestamp": "2026-01-14T10:30:00Z"
}
```

### Pagination Format

```json
{
    "data": [], // Array of items
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total": 100,
        "total_pages": 5,
        "has_next": true,
        "has_prev": false
    }
}
```

---

## Data Models & Schemas

### User Management

#### User Model

```json
{
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": true,
    "is_active": true,
    "is_admin": false,
    "is_staff": false,
    "is_author": false,
    "is_banned": false,
    "last_login_at": "2026-01-14T10:30:00Z",
    "rate_limit_group": "default",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

#### User Profile Response

```json
{
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": true,
    "profile": {
        "first_name": "John",
        "last_name": "Doe"
    },
    "roles": ["user", "author"],
    "scopes": ["web:read", "web:write", "story:read", "story:write"],
    "statistics": {
        "stories_count": 5,
        "followers_count": 12,
        "following_count": 8
    },
    "created_at": "2026-01-01T00:00:00Z"
}
```

### Author Management

#### Author Model

```json
{
    "id": "uuid",
    "user_id": "uuid",
    "stage_name": "John Doe",
    "slug": "john-doe",
    "first_name": "John",
    "last_name": "Doe",
    "bio": "Fiction writer and poet",
    "description": "Long form author description...",
    "gender": "other",
    "accepted_terms_of_service": true,
    "email_newsletters_and_changelogs": true,
    "email_promotions_and_events": false,
    "is_recommended": false,
    "media_id": "uuid", // Profile image
    "stats": {
        "stories_published": 5,
        "total_words": 50000,
        "total_votes": 150,
        "total_favorites": 75
    },
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

### Story Management

#### Story Model

```json
{
    "id": "uuid",
    "owner_id": "uuid",
    "media_id": "uuid", // Cover image
    "name": "My Great Story",
    "slug": "my-great-story",
    "synopsis": "A compelling story about...",
    "is_verified": false,
    "is_recommended": true,
    "status": "published", // draft, published, archived, deleted
    "first_published_at": "2026-01-10T00:00:00Z",
    "last_published_at": "2026-01-14T10:30:00Z",
    "settings": {
        "allow_comments": true,
        "allow_votes": true
    },
    "author": {
        "id": "uuid",
        "stage_name": "John Doe",
        "slug": "john-doe"
    },
    "genres": ["fiction", "romance"],
    "tags": ["drama", "young-adult"],
    "statistics": {
        "total_chapters": 12,
        "published_chapters": 10,
        "draft_chapters": 2,
        "total_words": 25000,
        "total_views": 1500,
        "total_votes": 45,
        "total_favorites": 23,
        "average_rating": 4.2
    },
    "user_interactions": {
        "has_voted": true,
        "user_vote": 5,
        "has_favorited": true,
        "has_bookmarked": false
    },
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

#### Chapter Model

```json
{
    "id": "uuid",
    "story_id": "uuid",
    "title": "Chapter 1: The Beginning",
    "number": 1,
    "sort_order": 0,
    "summary": "Where it all begins...",
    "is_published": true,
    "published_at": "2026-01-10T00:00:00Z",
    "total_words": 2500,
    "total_views": 150,
    "total_votes": 12,
    "total_favorites": 8,
    "content": {
        "id": "uuid",
        "language": "en",
        "content": {}, // JSON content structure
        "stats": {
            "word_count": 2500,
            "character_count": 12000,
            "paragraph_count": 15
        }
    },
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

### Content Management

#### File Model

```json
{
    "id": "uuid",
    "owner_id": "uuid",
    "name": "cover-image.jpg",
    "version": "1.0",
    "path": "/uploads/2026/01/cover-image.jpg",
    "mime_type": "image/jpeg",
    "size": 1024000,
    "service": "s3",
    "upload_source": "presigned",
    "checksum_etag": "abc123",
    "checksum_sha256_base64": "base64hash",
    "is_uploaded": true,
    "is_archived": false,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

### Social Features

#### Comment Model

```json
{
    "id": "uuid",
    "user_id": "uuid",
    "parent_id": "uuid", // For threaded comments
    "content": {
        "text": "Great story!",
        "mentions": ["@username"],
        "formatting": "markdown"
    },
    "depth": 0, // Thread depth
    "is_resolved": false,
    "is_archived": false,
    "is_reported": false,
    "is_pinned": false,
    "entity_type": "story", // story, chapter, submission
    "entity_id": "uuid",
    "reply_count": 3,
    "reactions": {
        "like": 5,
        "love": 2,
        "wow": 1
    },
    "user_reaction": "like",
    "user": {
        "id": "uuid",
        "username": "reader123",
        "avatar_url": "https://..."
    },
    "created_at": "2026-01-14T10:30:00Z",
    "updated_at": "2026-01-14T10:30:00Z"
}
```

---

## Detailed Endpoint Specifications

### Authentication Endpoints

#### POST /api/v1/auth/login

**Description**: Authenticate user and create session

**Request Body**:

```json
{
    "email": "user@example.com",
    "password": "password123"
}
```

**Response** (200):

```json
{
    "data": {
        "user": {
            "id": "uuid",
            "email": "user@example.com",
            "email_verified": true,
            "roles": ["user", "author"]
        },
        "session": {
            "expires_at": "2026-01-15T10:30:00Z",
            "scopes": ["web:read", "web:write", "story:read", "story:write"]
        }
    },
    "message": "Login successful"
}
```

**Error Responses**:

- 400: Invalid request format
- 401: Invalid credentials
- 403: Account banned or inactive

#### POST /api/v1/auth/register

**Description**: Register new user account

**Request Body**:

```json
{
    "email": "newuser@example.com",
    "password": "password123",
    "confirm_password": "password123"
}
```

**Response** (201):

```json
{
    "data": {
        "user": {
            "id": "uuid",
            "email": "newuser@example.com",
            "email_verified": false
        },
        "verification_required": true
    },
    "message": "Registration successful. Please verify your email."
}
```

### Story Management Endpoints

#### GET /api/v1/stories

**Description**: List stories with filtering and pagination

**Query Parameters**:

- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20, max: 100)
- `sort_by`: Sort field (created_at, updated_at, title, views)
- `sort_dir`: Sort direction (asc, desc)
- `status`: Filter by status (draft, published, archived)
- `genre`: Filter by genre (multiple)
- `tags`: Filter by tags (multiple)
- `author`: Filter by author ID or username
- `search`: Text search in title and synopsis
- `date_from`: Created after date
- `date_to`: Created before date

**Response** (200):

```json
{
    "data": [
        {
            "id": "uuid",
            "name": "My Great Story",
            "slug": "my-great-story",
            "synopsis": "A compelling story...",
            "status": "published",
            "author": {
                "id": "uuid",
                "stage_name": "John Doe",
                "slug": "john-doe"
            },
            "genres": ["fiction", "romance"],
            "tags": ["drama", "young-adult"],
            "statistics": {
                "total_chapters": 12,
                "published_chapters": 10,
                "total_words": 25000,
                "total_views": 1500,
                "average_rating": 4.2
            },
            "user_interactions": {
                "has_voted": false,
                "has_favorited": false,
                "has_bookmarked": false
            },
            "created_at": "2026-01-01T00:00:00Z",
            "updated_at": "2026-01-14T10:30:00Z"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 20,
        "total": 100,
        "total_pages": 5,
        "has_next": true,
        "has_prev": false
    }
}
```

#### POST /api/v1/stories

**Description**: Create new story

**Authentication Required**: Author scope

**Request Body**:

```json
{
    "name": "My New Story",
    "synopsis": "A story about...",
    "media_id": "uuid", // Cover image (optional)
    "genres": ["fiction", "romance"],
    "tags": ["drama", "young-adult"],
    "settings": {
        "allow_comments": true,
        "allow_votes": true
    }
}
```

**Response** (201):

```json
{
    "data": {
        "id": "uuid",
        "name": "My New Story",
        "slug": "my-new-story",
        "synopsis": "A story about...",
        "status": "draft",
        "owner_id": "uuid",
        "genres": ["fiction", "romance"],
        "tags": ["drama", "young-adult"],
        "statistics": {
            "total_chapters": 0,
            "published_chapters": 0,
            "total_words": 0,
            "total_views": 0,
            "total_votes": 0,
            "total_favorites": 0
        },
        "created_at": "2026-01-14T10:30:00Z"
    },
    "message": "Story created successfully"
}
```

### Chapter Management Endpoints

#### POST /api/v1/chapters

**Description**: Create new chapter

**Authentication Required**: Author scope

**Request Body**:

```json
{
    "story_id": "uuid",
    "title": "Chapter 1: The Beginning",
    "summary": "Where it all begins...",
    "content": {
        "language": "en",
        "content": {
            "sections": [
                {
                    "type": "paragraph",
                    "content": "Once upon a time..."
                }
            ]
        }
    },
    "sort_order": 1,
    "number": 1
}
```

**Response** (201):

```json
{
    "data": {
        "id": "uuid",
        "story_id": "uuid",
        "title": "Chapter 1: The Beginning",
        "summary": "Where it all begins...",
        "number": 1,
        "sort_order": 1,
        "is_published": false,
        "total_words": 1500,
        "content": {
            "id": "uuid",
            "language": "en"
        },
        "created_at": "2026-01-14T10:30:00Z"
    },
    "message": "Chapter created successfully"
}
```

### Social Features Endpoints

#### POST /api/v1/social/comments

**Description**: Create comment on entity

**Authentication Required**: User scope

**Request Body**:

```json
{
    "entity_type": "story", // story, chapter, submission
    "entity_id": "uuid",
    "parent_id": "uuid", // Optional for replies
    "content": {
        "text": "Great story! I really enjoyed the character development.",
        "mentions": ["@authorname"],
        "formatting": "markdown"
    }
}
```

**Response** (201):

```json
{
    "data": {
        "id": "uuid",
        "entity_type": "story",
        "entity_id": "uuid",
        "parent_id": null,
        "content": {
            "text": "Great story! I really enjoyed the character development.",
            "mentions": ["@authorname"],
            "formatting": "markdown"
        },
        "depth": 0,
        "reply_count": 0,
        "reactions": {},
        "user": {
            "id": "uuid",
            "username": "reader123"
        },
        "created_at": "2026-01-14T10:30:00Z"
    },
    "message": "Comment created successfully"
}
```

---

## Implementation Guidelines

### Handler Patterns

All handlers should follow this consistent pattern:

```go
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
    // 1. Extract context and subject
    ctx := r.Context()
    subject, err := subject.GetUserSubject(ctx)
    if err != nil {
        respond.Error(w, http.StatusUnauthorized, err)
        return
    }

    // 2. Decode and validate request
    var req RequestStruct
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respond.Error(w, http.StatusBadRequest, err)
        return
    }

    if err := h.validator.Struct(req); err != nil {
        respond.Error(w, http.StatusBadRequest, err)
        return
    }

    // 3. Process business logic
    result, err := h.usecase.Method(ctx, subject, &req)
    if err != nil {
        respond.Error(w, http.StatusInternalServerError, err)
        return
    }

    // 4. Send response
    respond.JSON(w, http.StatusOK, result)
}
```

### Database Considerations

1. **Connection Pooling**: Use pgx/v5 connection pooling
2. **Transaction Management**: Use proper transaction handling
3. **Soft Deletes**: Use `deleted_at` fields where appropriate
4. **Indexing**: Optimize queries with proper indexes
5. **Materialized Views**: Use for story statistics
6. **Partitions**: Use partitioned tables for comments

### Security Best Practices

1. **Input Validation**: Validate all input parameters
2. **SQL Injection**: Use parameterized queries only
3. **XSS Protection**: Sanitize user-generated content
4. **CSRF Protection**: Use secure cookies with SameSite
5. **Rate Limiting**: Implement per-user/IP rate limits
6. **File Uploads**: Validate file types and sizes
7. **Authorization**: Check permissions for every operation

### Performance Considerations

1. **Caching**: Redis for frequently accessed data
2. **Pagination**: Cursor-based for large datasets
3. **Compression**: Gzip responses for large payloads
4. **CDN**: Static assets via CDN
5. **Database Optimization**: Proper indexing and query optimization
6. **Async Operations**: Background jobs for heavy processing

---

## OpenAPI Documentation

### Swagger Annotations Example

```go
// @Summary Create new story
// @Description Creates a new story for the authenticated author
// @Tags stories
// @Accept json
// @Produce json
// @Param request body CreateStoryRequest true "Story creation data"
// @Success 201 {object} CreateStoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/stories [post]
// @Security ApiKeyAuth
```

### Generated Documentation

The API should automatically generate OpenAPI 3.0 documentation at `/swagger/index.html` with:

- Interactive API explorer
- Request/response examples
- Authentication flow documentation
- Schema definitions
- Error response examples

---

## Testing Strategy

### Unit Testing

- Handler function tests with mock use cases
- Request validation tests
- Error handling tests
- Authentication/authorization tests

### Integration Testing

- End-to-end API tests with database
- File upload/download tests
- Authentication flow tests
- Complex workflow tests

### Performance Testing

- Load testing for high-traffic endpoints
- Database query performance
- File upload performance
- Concurrent user testing

---

This specification provides a comprehensive foundation for implementing the Samsa Writing Platform API. It covers all major features while maintaining consistency, security, and scalability considerations.

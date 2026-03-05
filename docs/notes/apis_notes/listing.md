# Samsa Writing Platform - API Endpoints Listing

## Overview

This document provides a comprehensive listing of all planned REST API endpoints for the Samsa writing platform, organized by feature areas. The API follows RESTful principles with proper HTTP semantics and is organized using feature-based routing.

**Base URL**: `/api/v1/`
**Content-Type**: `application/json`
**Authentication**: Cookie-based sessions with JWT tokens

---

## Authentication & Authorization (`/api/v1/auth/`)

### Authentication Endpoints

| Method | Path                              | Description                    | Auth Required |
| ------ | --------------------------------- | ------------------------------ | ------------- |
| POST   | `/auth/login`                     | User login with email/password | No            |
| POST   | `/auth/logout`                    | User logout                    | Yes           |
| POST   | `/auth/register`                  | New user registration          | No            |
| POST   | `/auth/verification-email`        | Send email verification code   | No            |
| POST   | `/auth/verification-email/{code}` | Verify email with code         | No            |
| POST   | `/auth/password/change`           | Change current password        | Yes           |
| POST   | `/auth/password/forgot`           | Request password reset         | No            |
| POST   | `/auth/password/reset/{code}`     | Reset password with code       | No            |

### OAuth Management

| Method | Path                         | Description           | Auth Required |
| ------ | ---------------------------- | --------------------- | ------------- |
| DELETE | `/auth/providers/{provider}` | Remove OAuth provider | Yes           |

---

## User Management (`/api/v1/users/`)

### User Profile

| Method | Path               | Description                       | Auth Required |
| ------ | ------------------ | --------------------------------- | ------------- |
| GET    | `/users/me`        | Get current user profile          | Yes           |
| PUT    | `/users/me`        | Update current user profile       | Yes           |
| GET    | `/users/me/scopes` | Get user permissions/scopes       | Yes           |
| GET    | `/users/{id}`      | Get user profile (public view)    | No            |
| GET    | `/users`           | List users (admin/moderator only) | Admin         |

---

## Author Management (`/api/v1/authors/`)

### Author Profiles

| Method | Path            | Description                       | Auth Required |
| ------ | --------------- | --------------------------------- | ------------- |
| POST   | `/authors`      | Create author profile             | User          |
| GET    | `/authors/me`   | Get my author profile             | Author        |
| PUT    | `/authors/me`   | Update my author profile          | Author        |
| GET    | `/authors/{id}` | Get author profile                | No            |
| GET    | `/authors`      | List authors (with search/filter) | No            |

### Author Verification & Stats

| Method | Path                   | Description               | Auth Required |
| ------ | ---------------------- | ------------------------- | ------------- |
| POST   | `/authors/{id}/verify` | Verify author application | Admin         |
| GET    | `/authors/{id}/stats`  | Get author statistics     | Public/Owner  |

---

## Story Management (`/api/v1/stories/`)

### Story CRUD

| Method | Path            | Description                          | Auth Required |
| ------ | --------------- | ------------------------------------ | ------------- |
| GET    | `/stories`      | List stories (with filtering/search) | No            |
| POST   | `/stories`      | Create new story                     | Author        |
| GET    | `/stories/{id}` | Get story details                    | No            |
| PUT    | `/stories/{id}` | Update story                         | Author        |
| DELETE | `/stories/{id}` | Delete story                         | Author/Admin  |

### Publishing & Workflow

| Method | Path                      | Description     | Auth Required |
| ------ | ------------------------- | --------------- | ------------- |
| POST   | `/stories/{id}/publish`   | Publish story   | Author        |
| POST   | `/stories/{id}/unpublish` | Unpublish story | Author        |

### Story Chapters & Stats

| Method | Path                     | Description          | Auth Required |
| ------ | ------------------------ | -------------------- | ------------- |
| GET    | `/stories/{id}/chapters` | Get story chapters   | No            |
| GET    | `/stories/{id}/stats`    | Get story statistics | Author/Admin  |

### Story Interactions

| Method | Path                     | Description               | Auth Required |
| ------ | ------------------------ | ------------------------- | ------------- |
| POST   | `/stories/{id}/vote`     | Vote on story (1-5 stars) | User          |
| DELETE | `/stories/{id}/vote`     | Remove vote               | User          |
| POST   | `/stories/{id}/favorite` | Favorite story            | User          |
| DELETE | `/stories/{id}/favorite` | Unfavorite story          | User          |
| POST   | `/stories/{id}/bookmark` | Bookmark story            | User          |
| DELETE | `/stories/{id}/bookmark` | Remove bookmark           | User          |

---

## Chapter Management (`/api/v1/chapters/`)

### Chapter CRUD

| Method | Path             | Description         | Auth Required |
| ------ | ---------------- | ------------------- | ------------- |
| POST   | `/chapters`      | Create new chapter  | Author        |
| GET    | `/chapters/{id}` | Get chapter details | No            |
| PUT    | `/chapters/{id}` | Update chapter      | Author        |
| DELETE | `/chapters/{id}` | Delete chapter      | Author        |

### Chapter Publishing

| Method | Path                       | Description       | Auth Required |
| ------ | -------------------------- | ----------------- | ------------- |
| POST   | `/chapters/{id}/publish`   | Publish chapter   | Author        |
| POST   | `/chapters/{id}/unpublish` | Unpublish chapter | Author        |

### Chapter Content

| Method | Path                      | Description            | Auth Required |
| ------ | ------------------------- | ---------------------- | ------------- |
| GET    | `/chapters/{id}/document` | Get chapter content    | No            |
| PUT    | `/chapters/{id}/document` | Update chapter content | Author        |

---

## Content Management (`/api/v1/content/`)

### File Management

| Method | Path                            | Description       | Auth Required |
| ------ | ------------------------------- | ----------------- | ------------- |
| POST   | `/content/files`                | Upload file       | User          |
| GET    | `/content/files/{id}`           | Get file metadata | User          |
| DELETE | `/content/files/{id}`           | Delete file       | Owner/Admin   |
| GET    | `/content/files/{id}/presigned` | Get presigned URL | User          |

### Document Management

| Method | Path                               | Description           | Auth Required |
| ------ | ---------------------------------- | --------------------- | ------------- |
| POST   | `/content/documents`               | Create document       | Author        |
| GET    | `/content/documents/{id}`          | Get document          | Owner/Shared  |
| PUT    | `/content/documents/{id}`          | Update document       | Owner         |
| GET    | `/content/documents/{id}/versions` | Get document versions | Owner         |

### Template Management

| Method | Path                      | Description     | Auth Required |
| ------ | ------------------------- | --------------- | ------------- |
| GET    | `/content/templates`      | List templates  | No            |
| POST   | `/content/templates`      | Create template | User          |
| GET    | `/content/templates/{id}` | Get template    | No            |
| PUT    | `/content/templates/{id}` | Update template | Owner         |
| DELETE | `/content/templates/{id}` | Delete template | Owner/Admin   |

---

## Social Features (`/api/v1/social/`)

### Comment System

| Method | Path                    | Description                | Auth Required |
| ------ | ----------------------- | -------------------------- | ------------- |
| POST   | `/social/comments`      | Create comment             | User          |
| GET    | `/social/comments/{id}` | Get comment                | No            |
| PUT    | `/social/comments/{id}` | Update comment             | Owner         |
| DELETE | `/social/comments/{id}` | Delete comment             | Owner/Admin   |
| GET    | `/social/comments`      | List comments (for entity) | No            |

### Comment Reactions

| Method | Path                          | Description      | Auth Required |
| ------ | ----------------------------- | ---------------- | ------------- |
| POST   | `/social/comments/{id}/react` | React to comment | User          |
| DELETE | `/social/comments/{id}/react` | Remove reaction  | User          |

### User Following

| Method | Path                        | Description        | Auth Required |
| ------ | --------------------------- | ------------------ | ------------- |
| POST   | `/social/follows`           | Follow user        | User          |
| DELETE | `/social/follows/{userId}`  | Unfollow user      | User          |
| GET    | `/social/follows/following` | Get following list | Owner         |
| GET    | `/social/follows/followers` | Get followers list | Owner         |

---

## Content Moderation (`/api/v1/moderation/`)

### Report Management

| Method | Path                               | Description        | Auth Required   |
| ------ | ---------------------------------- | ------------------ | --------------- |
| GET    | `/moderation/reports`              | List reports       | Moderator/Admin |
| POST   | `/moderation/reports`              | Create report      | User            |
| GET    | `/moderation/reports/{id}`         | Get report details | Moderator/Admin |
| PUT    | `/moderation/reports/{id}/resolve` | Resolve report     | Moderator/Admin |

### Content Flagging

| Method | Path                            | Description      | Auth Required   |
| ------ | ------------------------------- | ---------------- | --------------- |
| POST   | `/moderation/flags`             | Flag content     | Moderator       |
| GET    | `/moderation/flags`             | List flags       | Moderator/Admin |
| GET    | `/moderation/flags/{id}`        | Get flag details | Moderator/Admin |
| PUT    | `/moderation/flags/{id}/review` | Review flag      | Moderator/Admin |

---

## Genre Management (`/api/v1/genres/`)

### Genre CRUD

| Method | Path           | Description       | Auth Required |
| ------ | -------------- | ----------------- | ------------- |
| GET    | `/genres`      | List genres       | No            |
| POST   | `/genres`      | Create genre      | Admin         |
| GET    | `/genres/{id}` | Get genre details | No            |
| PUT    | `/genres/{id}` | Update genre      | Admin         |
| DELETE | `/genres/{id}` | Delete genre      | Admin         |

---

## Tag Management (`/api/v1/tags/`)

### Tag CRUD

| Method | Path           | Description     | Auth Required |
| ------ | -------------- | --------------- | ------------- |
| GET    | `/tags`        | List tags       | No            |
| POST   | `/tags`        | Create tag      | User          |
| GET    | `/tags/{id}`   | Get tag details | No            |
| PUT    | `/tags/{id}`   | Update tag      | Owner         |
| DELETE | `/tags/{id}`   | Delete tag      | Owner/Admin   |
| GET    | `/tags/search` | Search tags     | No            |

---

## Notifications (`/api/v1/notifications/`)

### Notification Management

| Method | Path                       | Description                    | Auth Required |
| ------ | -------------------------- | ------------------------------ | ------------- |
| GET    | `/notifications`           | Get user notifications         | User          |
| PUT    | `/notifications/{id}/read` | Mark notification as read      | User          |
| PUT    | `/notifications/read-all`  | Mark all notifications as read | User          |
| DELETE | `/notifications/{id}`      | Delete notification            | User          |
| DELETE | `/notifications`           | Clear all notifications        | User          |

---

## Admin & System (`/api/v1/admin/`)

### User Administration

| Method | Path                    | Description      | Auth Required |
| ------ | ----------------------- | ---------------- | ------------- |
| GET    | `/admin/users`          | List all users   | Admin         |
| GET    | `/admin/users/{id}`     | Get user details | Admin         |
| PUT    | `/admin/users/{id}/ban` | Ban user         | Admin         |
| DELETE | `/admin/users/{id}/ban` | Unban user       | Admin         |

### System Management

| Method | Path                   | Description            | Auth Required |
| ------ | ---------------------- | ---------------------- | ------------- |
| GET    | `/admin/activities`    | List system activities | Admin         |
| GET    | `/admin/configuration` | Get system config      | Admin         |
| PUT    | `/admin/configuration` | Update system config   | Admin         |
| GET    | `/admin/stats`         | Get system statistics  | Admin         |

---

## Health & System (`/api/v1/`)

### System Endpoints

| Method | Path      | Description         | Auth Required |
| ------ | --------- | ------------------- | ------------- |
| GET    | `/health` | System health check | No            |

---

## Summary Statistics

- **Total Endpoints**: 98
- **Feature Areas**: 12
- **Authentication Required**: 71 endpoints
- **Public Access**: 27 endpoints
- **Admin Required**: 16 endpoints
- **Author Required**: 18 endpoints

## Security Notes

- All endpoints use cookie-based authentication with secure, HttpOnly cookies
- Sensitive operations require appropriate scopes
- Rate limiting applies per user/IP
- CORS configured per environment
- Input validation on all endpoints
- SQL injection protection via parameterized queries
- File uploads validated for type and size

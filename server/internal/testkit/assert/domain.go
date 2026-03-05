package assert

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/assert"
)

// Domain-specific assertions for the Samsa application

// ValidUUID checks if a UUID is valid (not nil).
func ValidUUID(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	return assert.NotEqual(t, uuid.Nil, id)
}

// NilUUID checks if a UUID is nil (all zeros).
func NilUUID(t *testing.T, id uuid.UUID) bool {
	t.Helper()
	return assert.Equal(t, uuid.Nil, id)
}

// ValidNullableUUID checks if a nullable UUID is valid (not nil when present).
func ValidNullableUUID(t *testing.T, id *uuid.UUID) bool {
	t.Helper()
	if id == nil {
		return true // nil is acceptable for nullable
	}
	return assert.NotEqual(t, uuid.Nil, *id)
}

// NonEmptyString checks if a string is not empty.
func NonEmptyString(t *testing.T, s string) bool {
	t.Helper()
	return assert.NotEmpty(t, s)
}

// ValidEmail checks if a string looks like a valid email.
func ValidEmail(t *testing.T, email string) bool {
	t.Helper()
	// Simple check - contains @ and has content on both sides
	return assert.Contains(t, email, "@") &&
		assert.NotEmpty(t, email[:indexOf(email, '@')]) &&
		assert.NotEmpty(t, email[indexOf(email, '@')+1:])
}

func indexOf(s string, substr rune) int {
	for i, r := range s {
		if r == substr {
			return i
		}
	}
	return -1
}

// ValidSlug checks if a string is a valid slug (lowercase, hyphens allowed).
func ValidSlug(t *testing.T, slug string) bool {
	t.Helper()
	if !assert.NotEmpty(t, slug) {
		return false
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return assert.Fail(t, "invalid slug character", "character: %c", r)
		}
	}
	return true
}

// ValidTimestamp checks if a timestamp is reasonable (not too far in past/future).
func ValidTimestamp(t *testing.T, ts *time.Time, maxAge time.Duration) bool {
	t.Helper()
	if ts == nil {
		return assert.Fail(t, "timestamp is nil")
	}
	now := time.Now()
	oldest := now.Add(-maxAge)
	future := now.Add(1 * time.Hour) // Allow some clock skew

	if ts.Before(oldest) || ts.After(future) {
		return assert.Fail(t, "timestamp out of reasonable range", "timestamp: %v", ts)
	}
	return true
}

// ValidComment checks if a comment has required fields populated.
func ValidComment(t *testing.T, comment *sqlc.Comment) bool {
	t.Helper()
	if !assert.NotNil(t, comment) {
		return false
	}

	success := true
	if !ValidUUID(t, comment.ID) {
		success = false
	}
	if !ValidUUID(t, comment.UserID) {
		success = false
	}
	if !ValidUUID(t, comment.EntityID) {
		success = false
	}
	if comment.EntityType == "" {
		success = assert.Fail(t, "entity type is empty")
	}
	if comment.Content == nil {
		success = assert.Fail(t, "content is empty")
	}

	return success
}

// ValidFile checks if a file has required fields populated.
func ValidFile(t *testing.T, file *sqlc.File) bool {
	t.Helper()
	if !assert.NotNil(t, file) {
		return false
	}

	success := true
	if !ValidUUID(t, file.ID) {
		success = false
	}
	if !ValidUUID(t, file.OwnerID) {
		success = false
	}
	if !NonEmptyString(t, file.Name) {
		success = false
	}
	if !NonEmptyString(t, file.Path) {
		success = false
	}
	if !NonEmptyString(t, file.Reference) {
		success = false
	}
	if file.Size <= 0 {
		success = assert.Fail(t, "file size should be positive", "size: %d", file.Size)
	}

	return success
}

// ValidSubmission checks if a submission has required fields populated.
func ValidSubmission(t *testing.T, submission *sqlc.Submission) bool {
	t.Helper()
	if !assert.NotNil(t, submission) {
		return false
	}

	success := true
	if !ValidUUID(t, submission.ID) {
		success = false
	}
	if !ValidUUID(t, submission.RequesterID) {
		success = false
	}
	if !NonEmptyString(t, submission.Title) {
		success = false
	}
	if submission.Type == "" {
		success = assert.Fail(t, "submission type is empty")
	}
	if submission.Status == "" {
		success = assert.Fail(t, "submission status is empty")
	}
	if !NonEmptyString(t, submission.ExposeID) {
		success = false
	}

	return success
}

// ValidStoryPost checks if a story post has required fields populated.
func ValidStoryPost(t *testing.T, post *sqlc.StoryPost) bool {
	t.Helper()
	if !assert.NotNil(t, post) {
		return false
	}

	success := true
	if !ValidUUID(t, post.ID) {
		success = false
	}
	if !ValidUUID(t, post.AuthorID) {
		success = false
	}
	if !NonEmptyString(t, post.Content) {
		success = false
	}

	return success
}

// ValidAuthor checks if an author has required fields populated.
func ValidAuthor(t *testing.T, author *sqlc.Author) bool {
	t.Helper()
	if !assert.NotNil(t, author) {
		return false
	}

	success := true
	if !ValidUUID(t, author.ID) {
		success = false
	}
	if !ValidUUID(t, author.UserID) {
		success = false
	}
	if !NonEmptyString(t, author.StageName) {
		success = false
	}
	if !NonEmptyString(t, author.Slug) {
		success = false
	}

	return success
}

// ValidStory checks if a story has required fields populated.
func ValidStory(t *testing.T, story *sqlc.Story) bool {
	t.Helper()
	if !assert.NotNil(t, story) {
		return false
	}

	success := true
	if !ValidUUID(t, story.ID) {
		success = false
	}
	if !ValidUUID(t, story.OwnerID) {
		success = false
	}
	if !NonEmptyString(t, story.Name) {
		success = false
	}
	if !NonEmptyString(t, story.Slug) {
		success = false
	}
	if story.Status == "" {
		success = assert.Fail(t, "story status is empty")
	}

	return success
}

// ValidUser checks if a user has required fields populated.
func ValidUser(t *testing.T, user *sqlc.User) bool {
	t.Helper()
	if !assert.NotNil(t, user) {
		return false
	}

	success := true
	if !ValidUUID(t, user.ID) {
		success = false
	}
	if !NonEmptyString(t, user.Email) {
		success = false
	}
	if !NonEmptyString(t, user.PasswordHash) {
		success = false
	}

	return success
}

// StoryStatus checks if the story status matches expected.
func StoryStatus(t *testing.T, story *sqlc.Story, expected sqlc.StoryStatus) bool {
	t.Helper()
	return assert.Equal(t, expected, story.Status)
}

// SubmissionStatus checks if the submission status matches expected.
func SubmissionStatus(t *testing.T, submission *sqlc.Submission, expected sqlc.SubmissionStatus) bool {
	t.Helper()
	return assert.Equal(t, expected, submission.Status)
}

// CommentNotDeleted checks if a comment is not marked as deleted.
func CommentNotDeleted(t *testing.T, comment *sqlc.Comment) bool {
	t.Helper()
	if comment.IsDeleted == nil {
		return assert.Fail(t, "comment.IsDeleted is nil")
	}
	return assert.False(t, *comment.IsDeleted)
}

// CommentIsDeleted checks if a comment is marked as deleted.
func CommentIsDeleted(t *testing.T, comment *sqlc.Comment) bool {
	t.Helper()
	if comment.IsDeleted == nil {
		return assert.Fail(t, "comment.IsDeleted is nil")
	}
	return assert.True(t, *comment.IsDeleted)
}

// FileNotDeleted checks if a file is not marked as deleted.
func FileNotDeleted(t *testing.T, file *sqlc.File) bool {
	t.Helper()
	return assert.False(t, file.IsDeleted)
}

// FileIsDeleted checks if a file is marked as deleted.
func FileIsDeleted(t *testing.T, file *sqlc.File) bool {
	t.Helper()
	return assert.True(t, file.IsDeleted)
}

// PaginationParams checks if pagination parameters are valid.
func PaginationParams(t *testing.T, limit, offset int32, maxLimit int32) bool {
	t.Helper()
	success := true

	if limit <= 0 {
		success = assert.Fail(t, "limit should be positive", "limit: %d", limit)
	}
	if limit > maxLimit {
		success = assert.Fail(t, "limit exceeds maximum", "limit: %d, max: %d", limit, maxLimit)
	}
	if offset < 0 {
		success = assert.Fail(t, "offset should be non-negative", "offset: %d", offset)
	}

	return success
}

// SliceLength checks if a slice has the expected length.
func SliceLength[T any](t *testing.T, slice []T, expected int) bool {
	t.Helper()
	return assert.Len(t, slice, expected)
}

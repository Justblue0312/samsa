package fixtures

import (
	"context"
	"testing"
	"time"

	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/testkit/factory"
)

// Fixtures provides pre-built test data sets for common testing scenarios.

// TestData is a container for all fixture data.
type TestData struct {
	Users       []*sqlc.User
	Authors     []*sqlc.Author
	Stories     []*sqlc.Story
	Comments    []*sqlc.Comment
	Files       []*sqlc.File
	Submissions []*sqlc.Submission
	StoryPosts  []*sqlc.StoryPost
}

// Empty returns an empty TestData container.
func Empty() *TestData {
	return &TestData{
		Users:       make([]*sqlc.User, 0),
		Authors:     make([]*sqlc.Author, 0),
		Stories:     make([]*sqlc.Story, 0),
		Comments:    make([]*sqlc.Comment, 0),
		Files:       make([]*sqlc.File, 0),
		Submissions: make([]*sqlc.Submission, 0),
		StoryPosts:  make([]*sqlc.StoryPost, 0),
	}
}

// BasicUsers creates a minimal set of users for testing.
// Returns: admin user, regular user, author user.
func BasicUsers(t *testing.T, db sqlc.DBTX) *TestData {
	t.Helper()
	data := Empty()

	// Admin user
	admin := factory.User(t, db, factory.UserOpts{
		Email:   "admin@example.com",
		IsAdmin: true,
	})
	data.Users = append(data.Users, admin)

	// Regular user
	user := factory.User(t, db, factory.UserOpts{
		Email: "user@example.com",
	})
	data.Users = append(data.Users, user)

	// Author user
	authorUser := factory.User(t, db, factory.UserOpts{
		Email: "author@example.com",
	})
	data.Users = append(data.Users, authorUser)

	return data
}

// Authors creates a set of authors for testing.
// Requires: BasicUsers or similar to be created first.
func Authors(t *testing.T, db sqlc.DBTX, users []*sqlc.User) *TestData {
	t.Helper()
	data := Empty()

	for _, user := range users {
		author := factory.Author(t, db, factory.AuthorOpts{
			UserID:    user.ID,
			StageName: "Author " + user.Email,
			Slug:      "author-" + user.ID.String()[:8],
		})
		data.Authors = append(data.Authors, author)
	}

	return data
}

// Stories creates a set of stories for testing.
// Requires: Authors to be created first.
func Stories(t *testing.T, db sqlc.DBTX, authors []*sqlc.Author, count int) *TestData {
	t.Helper()
	data := Empty()

	statuses := []sqlc.StoryStatus{
		sqlc.StoryStatusDraft,
		sqlc.StoryStatusPublished,
		sqlc.StoryStatusArchived,
		sqlc.StoryStatusIsReviewed,
		sqlc.StoryStatusIsApproved,
	}

	for i := 0; i < count; i++ {
		author := authors[i%len(authors)]
		status := statuses[i%len(statuses)]

		story := factory.Story(t, db, factory.StoryOpts{
			OwnerID: author.UserID,
			Name:    "Story " + factory.RandomString(4),
			Slug:    "story-" + factory.RandomString(4),
			Status:  status,
		})
		data.Stories = append(data.Stories, story)
	}

	return data
}

// Comments creates a threaded comment structure for testing.
// Creates: root comments and nested replies.
func Comments(t *testing.T, db sqlc.DBTX, story *sqlc.Story, user *sqlc.User, depth int) *TestData {
	t.Helper()
	data := Empty()

	// Create root comments
	rootCount := 3
	for i := 0; i < rootCount; i++ {
		comment := factory.Comment(t, db, factory.CommentOpts{
			UserID:     user.ID,
			EntityType: sqlc.EntityTypeStory,
			EntityID:   story.ID,
			Content:    []byte("Root comment " + factory.RandomString(4)),
		})
		data.Comments = append(data.Comments, comment)

		// Create nested replies
		if depth > 0 {
			createReplies(t, db, comment, user, depth, &data.Comments)
		}
	}

	return data
}

func createReplies(t *testing.T, db sqlc.DBTX, parent *sqlc.Comment, user *sqlc.User, maxDepth int, comments *[]*sqlc.Comment) {
	t.Helper()

	if *parent.Depth >= int32(maxDepth) {
		return
	}

	replyCount := 2
	for i := 0; i < replyCount; i++ {
		reply := factory.CommentReply(t, db, parent, factory.CommentOpts{
			UserID:  user.ID,
			Content: []byte("Reply " + factory.RandomString(4)),
		})
		*comments = append(*comments, reply)

		if *reply.Depth < int32(maxDepth) {
			createReplies(t, db, reply, user, maxDepth, comments)
		}
	}
}

// Files creates a set of files for testing.
func Files(t *testing.T, db sqlc.DBTX, owner *sqlc.User, count int) *TestData {
	t.Helper()
	data := Empty()

	mimeTypes := []string{
		"text/plain",
		"image/png",
		"image/jpeg",
		"application/pdf",
		"application/json",
	}

	for i := 0; i < count; i++ {
		file := factory.File(t, db, factory.FileOpts{
			OwnerID:  owner.ID,
			Name:     "file-" + factory.RandomString(4) + "." + fileExtension(mimeTypes[i%len(mimeTypes)]),
			MimeType: &mimeTypes[i%len(mimeTypes)],
			Size:     int64(1024 * (i + 1)),
		})
		data.Files = append(data.Files, file)
	}

	return data
}

func fileExtension(mimeType string) string {
	switch mimeType {
	case "text/plain":
		return "txt"
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "application/pdf":
		return "pdf"
	case "application/json":
		return "json"
	default:
		return "bin"
	}
}

// Submissions creates a set of submissions with various statuses.
func Submissions(t *testing.T, db sqlc.DBTX, requester *sqlc.User, count int) *TestData {
	t.Helper()
	data := Empty()

	statuses := []sqlc.SubmissionStatus{
		sqlc.SubmissionStatusPending,
		sqlc.SubmissionStatusAssigned,
		sqlc.SubmissionStatusApproved,
		sqlc.SubmissionStatusRejected,
		sqlc.SubmissionStatusArchived,
	}

	types := []sqlc.SubmissionType{
		sqlc.SubmissionTypeAuthorRequest,
		sqlc.SubmissionTypeStoryApproval,
		sqlc.SubmissionTypeChapterApproval,
		sqlc.SubmissionTypeOther,
	}

	for i := 0; i < count; i++ {
		submission := factory.Submission(t, db, factory.SubmissionOpts{
			RequesterID: requester.ID,
			Title:       "Submission " + factory.RandomString(4),
			Type:        types[i%len(types)],
			Status:      statuses[i%len(statuses)],
		})
		data.Submissions = append(data.Submissions, submission)
	}

	return data
}

// SubmissionsWithSLA creates submissions with specific timestamps for SLA testing.
func SubmissionsWithSLA(t *testing.T, db sqlc.DBTX, requester *sqlc.User) *TestData {
	t.Helper()
	data := Empty()

	// Submission within SLA (approved quickly)
	withinSLA := factory.SubmissionWithSLA(t, db, factory.SubmissionWithSLAOpts{
		RequesterID: requester.ID,
		Title:       "Within SLA",
		Status:      sqlc.SubmissionStatusApproved,
		CreatedAt:   factory.TimeAgo(12 * time.Hour),
		ApprovedAt:  factory.TimeAgo(6 * time.Hour),
	})
	data.Submissions = append(data.Submissions, withinSLA)

	// Submission exceeding SLA (took too long)
	exceedingSLA := factory.SubmissionWithSLA(t, db, factory.SubmissionWithSLAOpts{
		RequesterID: requester.ID,
		Title:       "Exceeding SLA",
		Status:      sqlc.SubmissionStatusApproved,
		CreatedAt:   factory.TimeAgo(48 * time.Hour),
		ApprovedAt:  factory.TimeAgo(24 * time.Hour),
	})
	data.Submissions = append(data.Submissions, exceedingSLA)

	// Pending submission (not yet approved)
	pending := factory.SubmissionWithSLA(t, db, factory.SubmissionWithSLAOpts{
		RequesterID: requester.ID,
		Title:       "Pending",
		Status:      sqlc.SubmissionStatusPending,
		CreatedAt:   factory.TimeAgo(36 * time.Hour),
	})
	data.Submissions = append(data.Submissions, pending)

	return data
}

// StoryPosts creates a set of story posts for testing.
func StoryPosts(t *testing.T, db sqlc.DBTX, author *sqlc.Author, count int) *TestData {
	t.Helper()
	data := Empty()

	for i := 0; i < count; i++ {
		post := factory.StoryPost(t, db, factory.StoryPostOpts{
			AuthorID: author.ID,
			Content:  "Story post content " + factory.RandomString(8),
		})
		data.StoryPosts = append(data.StoryPosts, post)
	}

	return data
}

// SharedFiles creates shared files between users.
func SharedFiles(t *testing.T, db sqlc.DBTX, owner, sharedWith *sqlc.User, count int) *TestData {
	t.Helper()
	data := Empty()

	for i := 0; i < count; i++ {
		file, _, _ := factory.SharedFile(t, db, factory.SharedFileOpts{
			OwnerID:    owner.ID,
			SharedWith: sharedWith.ID,
			Name:       "shared-file-" + factory.RandomString(4) + ".txt",
			Size:       int64(2048 * (i + 1)),
		})
		data.Files = append(data.Files, file)
	}

	return data
}

// CompleteFixture creates a complete test environment with all entity types.
func CompleteFixture(t *testing.T, db sqlc.DBTX) *TestData {
	t.Helper()

	// Create users
	users := BasicUsers(t, db)

	// Create authors from users
	authors := Authors(t, db, users.Users)

	// Create stories
	stories := Stories(t, db, authors.Authors, 5)

	// Create comments on first story
	if len(stories.Stories) > 0 && len(users.Users) > 0 {
		comments := Comments(t, db, stories.Stories[0], users.Users[1], 3)
		data := Empty()
		data.Users = users.Users
		data.Authors = authors.Authors
		data.Stories = stories.Stories
		data.Comments = comments.Comments
		return data
	}

	return users
}

// TruncateAll removes all data from test tables.
// Use with caution - only in test environments!
func TruncateAll(t *testing.T, db sqlc.DBTX) {
	t.Helper()

	tables := []string{
		"submission_status_histories",
		"submission_assignments",
		"submissions",
		"shared_files",
		"files",
		"comments",
		"story_posts",
		"stories",
		"authors",
		"users",
	}

	ctx := context.Background()

	for _, table := range tables {
		query := "TRUNCATE TABLE " + table + " CASCADE"
		_, err := db.Exec(ctx, query)
		if err != nil {
			// Table might not exist or doesn't support TRUNCATE, continue
			continue
		}
	}
}

package fixtures

import (
	"context"
	"testing"
	"time"

	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/testkit/factory"
)

// ScenarioBuilders provides complex test scenario builders for integration tests.

// ScenarioBuilder is a fluent interface for building test scenarios.
type ScenarioBuilder struct {
	t    *testing.T
	db   sqlc.DBTX
	data *TestData
}

// NewScenario creates a new scenario builder.
func NewScenario(t *testing.T, db sqlc.DBTX) *ScenarioBuilder {
	t.Helper()
	return &ScenarioBuilder{
		t:    t,
		db:   db,
		data: Empty(),
	}
}

// WithUsers adds users to the scenario.
func (b *ScenarioBuilder) WithUsers(count int, configure func(int, *factory.UserOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.UserOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		user := factory.User(b.t, b.db, opts)
		b.data.Users = append(b.data.Users, user)
	}

	return b
}

// WithAdmin adds an admin user to the scenario.
func (b *ScenarioBuilder) WithAdmin() *ScenarioBuilder {
	b.t.Helper()

	admin := factory.User(b.t, b.db, factory.UserOpts{
		Email:   "admin-" + factory.RandomString(4) + "@example.com",
		IsAdmin: true,
	})
	b.data.Users = append(b.data.Users, admin)
	return b
}

// WithAuthors adds authors to the scenario.
func (b *ScenarioBuilder) WithAuthors(count int, configure func(int, *factory.AuthorOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.AuthorOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		author := factory.Author(b.t, b.db, opts)
		b.data.Authors = append(b.data.Authors, author)
	}

	return b
}

// WithStories adds stories to the scenario.
func (b *ScenarioBuilder) WithStories(count int, configure func(int, *factory.StoryOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.StoryOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		story := factory.Story(b.t, b.db, opts)
		b.data.Stories = append(b.data.Stories, story)
	}

	return b
}

// WithComments adds comments to the scenario.
func (b *ScenarioBuilder) WithComments(count int, configure func(int, *factory.CommentOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.CommentOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		comment := factory.Comment(b.t, b.db, opts)
		b.data.Comments = append(b.data.Comments, comment)
	}

	return b
}

// WithFiles adds files to the scenario.
func (b *ScenarioBuilder) WithFiles(count int, configure func(int, *factory.FileOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.FileOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		file := factory.File(b.t, b.db, opts)
		b.data.Files = append(b.data.Files, file)
	}

	return b
}

// WithSubmissions adds submissions to the scenario.
func (b *ScenarioBuilder) WithSubmissions(count int, configure func(int, *factory.SubmissionOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.SubmissionOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		submission := factory.Submission(b.t, b.db, opts)
		b.data.Submissions = append(b.data.Submissions, submission)
	}

	return b
}

// WithStoryPosts adds story posts to the scenario.
func (b *ScenarioBuilder) WithStoryPosts(count int, configure func(int, *factory.StoryPostOpts)) *ScenarioBuilder {
	b.t.Helper()

	for i := 0; i < count; i++ {
		opts := factory.StoryPostOpts{}
		if configure != nil {
			configure(i, &opts)
		}
		post := factory.StoryPost(b.t, b.db, opts)
		b.data.StoryPosts = append(b.data.StoryPosts, post)
	}

	return b
}

// Build returns the constructed test data.
func (b *ScenarioBuilder) Build() *TestData {
	return b.data
}

// Pre-built Scenarios

// CommentModerationScenario creates a scenario for testing comment moderation.
// Includes: admin user, multiple users, stories, and comments with various states.
type CommentModerationScenario struct {
	Admin    *sqlc.User
	Users    []*sqlc.User
	Stories  []*sqlc.Story
	Comments []*sqlc.Comment
	Archived []*sqlc.Comment
	Resolved []*sqlc.Comment
	Deleted  []*sqlc.Comment
	Pinned   []*sqlc.Comment
}

// CommentModerationScenario builds a comment moderation test scenario.
func (b *ScenarioBuilder) CommentModerationScenario() *CommentModerationScenario {
	b.t.Helper()

	// Create admin
	admin := factory.User(b.t, b.db, factory.UserOpts{
		Email:   "admin-mod@example.com",
		IsAdmin: true,
	})

	// Create regular users
	var users []*sqlc.User
	for i := 0; i < 5; i++ {
		user := factory.User(b.t, b.db, factory.UserOpts{
			Email: "user-mod-" + factory.RandomString(4) + "@example.com",
		})
		users = append(users, user)
	}

	// Create stories
	var stories []*sqlc.Story
	for _, user := range users[:3] {
		story := factory.Story(b.t, b.db, factory.StoryOpts{
			OwnerID: user.ID,
			Name:    "Moderation Story " + factory.RandomString(4),
			Slug:    "mod-story-" + factory.RandomString(4),
			Status:  sqlc.StoryStatusPublished,
		})
		stories = append(stories, story)
	}

	// Create comments with various states
	var comments, archived, resolved, deleted, pinned []*sqlc.Comment

	for i := 0; i < 10; i++ {
		user := users[i%len(users)]
		story := stories[i%len(stories)]

		comment := factory.Comment(b.t, b.db, factory.CommentOpts{
			UserID:     user.ID,
			EntityType: sqlc.EntityTypeStory,
			EntityID:   story.ID,
			Content:    []byte("Comment " + factory.RandomString(8)),
		})
		comments = append(comments, comment)

		// Set different states for some comments
		if i%4 == 0 {
			// Archive
			archived = append(archived, updateCommentArchive(b.t, b.db, comment, true))
		} else if i%4 == 1 {
			// Resolve
			resolved = append(resolved, updateCommentResolve(b.t, b.db, comment, true))
		} else if i%4 == 2 {
			// Delete
			deleted = append(deleted, softDeleteComment(b.t, b.db, comment))
		} else if i%4 == 3 {
			// Pin
			pinned = append(pinned, updateCommentPin(b.t, b.db, comment, true))
		}
	}

	b.data.Users = append([]*sqlc.User{admin}, users...)
	b.data.Stories = stories
	b.data.Comments = comments

	return &CommentModerationScenario{
		Admin:    admin,
		Users:    users,
		Stories:  stories,
		Comments: comments,
		Archived: archived,
		Resolved: resolved,
		Deleted:  deleted,
		Pinned:   pinned,
	}
}

func updateCommentArchive(t *testing.T, db sqlc.DBTX, comment *sqlc.Comment, archived bool) *sqlc.Comment {
	t.Helper()
	q := sqlc.New(db)
	updated, err := q.UpdateComment(context.Background(), sqlc.UpdateCommentParams{
		EntityType:    comment.EntityType,
		ID:            comment.ID,
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsArchived:    &archived,
		IsResolved:    comment.IsResolved,
		IsPinned:      comment.IsPinned,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	})
	if err != nil {
		t.Fatalf("failed to archive comment: %v", err)
	}
	return &updated
}

func updateCommentResolve(t *testing.T, db sqlc.DBTX, comment *sqlc.Comment, resolved bool) *sqlc.Comment {
	t.Helper()
	q := sqlc.New(db)
	updated, err := q.UpdateComment(context.Background(), sqlc.UpdateCommentParams{
		EntityType:    comment.EntityType,
		ID:            comment.ID,
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsArchived:    comment.IsArchived,
		IsResolved:    &resolved,
		IsPinned:      comment.IsPinned,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	})
	if err != nil {
		t.Fatalf("failed to resolve comment: %v", err)
	}
	return &updated
}

func softDeleteComment(t *testing.T, db sqlc.DBTX, comment *sqlc.Comment) *sqlc.Comment {
	t.Helper()
	q := sqlc.New(db)
	now := time.Now()
	updated, err := q.UpdateComment(context.Background(), sqlc.UpdateCommentParams{
		EntityType:    comment.EntityType,
		ID:            comment.ID,
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsArchived:    comment.IsArchived,
		IsResolved:    comment.IsResolved,
		IsPinned:      comment.IsPinned,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	})
	if err != nil {
		t.Fatalf("failed to delete comment: %v", err)
	}
	// Manually set deleted_at and is_deleted since UpdateComment might not include them
	// This is a test helper, so we can use raw SQL
	_, _ = db.Exec(context.Background(), "UPDATE comments SET deleted_at = $1 WHERE id = $2", now, comment.ID)
	return &updated
}

func updateCommentPin(t *testing.T, db sqlc.DBTX, comment *sqlc.Comment, pinned bool) *sqlc.Comment {
	t.Helper()
	q := sqlc.New(db)
	now := time.Now()
	updated, err := q.UpdateComment(context.Background(), sqlc.UpdateCommentParams{
		EntityType:    comment.EntityType,
		ID:            comment.ID,
		UserID:        comment.UserID,
		ParentID:      comment.ParentID,
		Content:       comment.Content,
		Depth:         comment.Depth,
		Score:         comment.Score,
		IsArchived:    comment.IsArchived,
		IsResolved:    comment.IsResolved,
		IsPinned:      &pinned,
		PinnedAt:      &now,
		ReplyCount:    comment.ReplyCount,
		ReactionCount: comment.ReactionCount,
		Metadata:      comment.Metadata,
		DeletedBy:     comment.DeletedBy,
	})
	if err != nil {
		t.Fatalf("failed to pin comment: %v", err)
	}
	return &updated
}

// FileSharingScenario creates a scenario for testing file sharing.
type FileSharingScenario struct {
	Owner       *sqlc.User
	SharedUsers []*sqlc.User
	Files       []*sqlc.File
	SharedFiles []*sqlc.File
}

// FileSharingScenario builds a file sharing test scenario.
func (b *ScenarioBuilder) FileSharingScenario() *FileSharingScenario {
	b.t.Helper()

	// Create owner
	owner := factory.User(b.t, b.db, factory.UserOpts{
		Email: "owner-" + factory.RandomString(4) + "@example.com",
	})

	// Create users to share with
	var sharedUsers []*sqlc.User
	for i := 0; i < 3; i++ {
		user := factory.User(b.t, b.db, factory.UserOpts{
			Email: "shared-" + factory.RandomString(4) + "@example.com",
		})
		sharedUsers = append(sharedUsers, user)
	}

	// Create files
	var files []*sqlc.File
	for i := 0; i < 5; i++ {
		file := factory.File(b.t, b.db, factory.FileOpts{
			OwnerID: owner.ID,
			Name:    "file-" + factory.RandomString(4) + ".txt",
			Size:    int64(1024 * (i + 1)),
		})
		files = append(files, file)
	}

	// Share some files
	var sharedFiles []*sqlc.File
	q := sqlc.New(b.db)

	for _, file := range files[:3] {
		// Note: ShareFile only takes an ID, not full params
		// The sharing logic should be tested separately
		_, err := q.ShareFile(context.Background(), file.ID)
		if err != nil {
			b.t.Fatalf("failed to share file: %v", err)
		}
		sharedFiles = append(sharedFiles, file)
	}

	b.data.Users = append([]*sqlc.User{owner}, sharedUsers...)
	b.data.Files = files

	return &FileSharingScenario{
		Owner:       owner,
		SharedUsers: sharedUsers,
		Files:       files,
		SharedFiles: sharedFiles,
	}
}

// SLATrackingScenario creates a scenario for testing SLA tracking.
type SLATrackingScenario struct {
	Requester    *sqlc.User
	Approver     *sqlc.User
	WithinSLA    []*sqlc.Submission
	ExceedingSLA []*sqlc.Submission
	Pending      []*sqlc.Submission
	Rejected     []*sqlc.Submission
}

// SLATrackingScenario builds an SLA tracking test scenario.
func (b *ScenarioBuilder) SLATrackingScenario() *SLATrackingScenario {
	b.t.Helper()

	// Create requester and approver
	requester := factory.User(b.t, b.db, factory.UserOpts{
		Email: "requester-" + factory.RandomString(4) + "@example.com",
	})

	approver := factory.User(b.t, b.db, factory.UserOpts{
		Email:   "approver-" + factory.RandomString(4) + "@example.com",
		IsAdmin: true,
	})

	var withinSLA, exceedingSLA, pending, rejected []*sqlc.Submission

	// Submissions within SLA (approved quickly)
	for i := 0; i < 3; i++ {
		created := factory.TimeAgo(time.Duration(i+1) * time.Hour)
		approved := factory.TimeAgo(time.Duration(i) * time.Hour)

		sub := factory.SubmissionWithSLA(b.t, b.db, factory.SubmissionWithSLAOpts{
			RequesterID: requester.ID,
			ApproverID:  &approver.ID,
			Title:       "Within SLA " + factory.RandomString(4),
			Status:      sqlc.SubmissionStatusApproved,
			CreatedAt:   created,
			ApprovedAt:  approved,
		})
		withinSLA = append(withinSLA, sub)
	}

	// Submissions exceeding SLA (took too long)
	for i := 0; i < 2; i++ {
		created := factory.TimeAgo(time.Duration(48+i*12) * time.Hour)
		approved := factory.TimeAgo(time.Duration(24+i*12) * time.Hour)

		sub := factory.SubmissionWithSLA(b.t, b.db, factory.SubmissionWithSLAOpts{
			RequesterID: requester.ID,
			ApproverID:  &approver.ID,
			Title:       "Exceeding SLA " + factory.RandomString(4),
			Status:      sqlc.SubmissionStatusApproved,
			CreatedAt:   created,
			ApprovedAt:  approved,
		})
		exceedingSLA = append(exceedingSLA, sub)
	}

	// Pending submissions (not yet approved)
	for i := 0; i < 2; i++ {
		created := factory.TimeAgo(time.Duration(36+i*12) * time.Hour)

		sub := factory.SubmissionWithSLA(b.t, b.db, factory.SubmissionWithSLAOpts{
			RequesterID: requester.ID,
			Title:       "Pending " + factory.RandomString(4),
			Status:      sqlc.SubmissionStatusPending,
			CreatedAt:   created,
		})
		pending = append(pending, sub)
	}

	// Rejected submissions
	for i := 0; i < 1; i++ {
		created := factory.TimeAgo(24 * time.Hour)
		approved := factory.TimeAgo(6 * time.Hour)

		sub := factory.SubmissionWithSLA(b.t, b.db, factory.SubmissionWithSLAOpts{
			RequesterID: requester.ID,
			ApproverID:  &approver.ID,
			Title:       "Rejected " + factory.RandomString(4),
			Status:      sqlc.SubmissionStatusRejected,
			CreatedAt:   created,
			ApprovedAt:  approved,
		})
		rejected = append(rejected, sub)
	}

	b.data.Users = []*sqlc.User{requester, approver}
	b.data.Submissions = append(append(append(withinSLA, exceedingSLA...), pending...), rejected...)

	return &SLATrackingScenario{
		Requester:    requester,
		Approver:     approver,
		WithinSLA:    withinSLA,
		ExceedingSLA: exceedingSLA,
		Pending:      pending,
		Rejected:     rejected,
	}
}

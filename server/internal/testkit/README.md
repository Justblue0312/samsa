# TestKit - Samsa Testing Toolkit

A comprehensive testing toolkit for the Samsa project, providing factories, assertions, fixtures, and integration test helpers.

## Overview

```
testkit/
├── db.go                    # Database connection and helpers
├── factory/                 # Test data factories
│   ├── helpers.go           # Random data generators
│   ├── user.go              # User factory
│   ├── author.go            # Author factory
│   ├── story.go             # Story factory
│   ├── comment.go           # Comment factory
│   ├── file.go              # File factory
│   ├── submission.go        # Submission factory
│   └── post.go              # Story post factory
├── assert/                  # Assertion helpers
│   ├── database.go          # Database assertions
│   ├── http.go              # HTTP response assertions
│   └── domain.go            # Domain-specific assertions
├── fixtures/                # Pre-built test data
│   ├── fixtures.go          # Common fixtures
│   └── scenario_builders.go # Complex scenario builders
├── integration/             # Integration test helpers
│   ├── suite.go             # Test suite base
│   └── fixtures_loader.go   # SQL fixture loader
└── mocks/                   # Mock generation guides
    └── README.md            # Mock generation documentation
```

## Quick Start

### Basic Test Setup

```go
package comment_test

import (
    "context"
    "testing"

    "github.com/justblue/samsa/internal/feature/comment"
    "github.com/justblue/samsa/internal/testkit"
    "github.com/justblue/samsa/internal/testkit/factory"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCommentRepository_Create(t *testing.T) {
    // Get shared database pool
    pool := testkit.NewDB(t)
    cfg := testkit.SetupConfig()
    
    // Start transaction (auto-rollback)
    tx := testkit.NewTx(t, pool)
    
    // Create repository with transaction
    repo := comment.NewRepository(tx, cfg, nil)
    
    // Create test data using factory
    user := factory.User(t, tx, factory.UserOpts{})
    story := factory.Story(t, tx, factory.StoryOpts{OwnerID: user.ID})
    
    // Test your code
    ctx := context.Background()
    result, err := repo.Create(ctx, &sqlc.Comment{
        UserID:     user.ID,
        EntityID:   story.ID,
        EntityType: sqlc.EntityTypeStory,
        Content:    []byte("Test comment"),
    })
    
    // Assert results
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.NotEqual(t, uuid.Nil, result.ID)
}
```

## Factories

Factories create test data with sensible defaults. Use opts to customize specific fields.

### User Factory

```go
import "github.com/justblue/samsa/internal/testkit/factory"

// Basic user
user := factory.User(t, db, factory.UserOpts{})

// Admin user
admin := factory.User(t, db, factory.UserOpts{
    Email:   "admin@example.com",
    IsAdmin: true,
})

// Banned user
banned := factory.User(t, db, factory.UserOpts{
    Email:    "banned@example.com",
    IsBanned: true,
})
```

### Author Factory

```go
// Basic author (creates user automatically)
author := factory.Author(t, db, factory.AuthorOpts{})

// Author with specific user
author := factory.Author(t, db, factory.AuthorOpts{
    UserID:    existingUser.ID,
    StageName: "My Pen Name",
    Slug:      "my-pen-name",
})
```

### Story Factory

```go
// Draft story
story := factory.Story(t, db, factory.StoryOpts{
    OwnerID: user.ID,
    Status:  sqlc.StoryStatusDraft,
})

// Published story
published := factory.Story(t, db, factory.StoryOpts{
    OwnerID: user.ID,
    Status:  sqlc.StoryStatusPublished,
})
```

### Comment Factory

```go
// Root comment
comment := factory.Comment(t, db, factory.CommentOpts{
    UserID:     user.ID,
    EntityType: sqlc.EntityTypeStory,
    EntityID:   story.ID,
    Content:    []byte("Great chapter!"),
})

// Reply to comment
reply := factory.CommentReply(t, db, parentComment, factory.CommentOpts{
    UserID:  otherUser.ID,
    Content: []byte("I agree!"),
})
```

### File Factory

```go
// Basic file
file := factory.File(t, db, factory.FileOpts{
    OwnerID: user.ID,
    Name:    "document.pdf",
    Size:    1024 * 1024, // 1MB
})

// Shared file
file, shared := factory.SharedFile(t, db, factory.SharedFileOpts{
    OwnerID:    owner.ID,
    SharedWith: recipient.ID,
    Name:       "shared.pdf",
})
```

### Submission Factory

```go
// Basic submission
sub := factory.Submission(t, db, factory.SubmissionOpts{
    RequesterID: user.ID,
    Title:       "Appeal Submission",
    Type:        sqlc.SubmissionTypeAppeal,
    Status:      sqlc.SubmissionStatusPending,
})

// Submission with SLA timestamps
slaSub := factory.SubmissionWithSLA(t, db, factory.SubmissionWithSLAOpts{
    RequesterID: user.ID,
    Title:       "SLA Test",
    Status:      sqlc.SubmissionStatusApproved,
    CreatedAt:   factory.TimeAgo(48 * time.Hour),
    ApprovedAt:  factory.TimeAgo(24 * time.Hour),
})
```

### Story Post Factory

```go
// Basic post
post := factory.StoryPost(t, db, factory.StoryPostOpts{
    AuthorID: author.ID,
    Content:  "Check out my new story!",
})

// Post with story
postWithStory := factory.StoryPostWithStory(t, db, factory.StoryPostOpts{
    AuthorID: author.ID,
    Content:  "New chapter is live!",
})
```

## Assertions

### Database Assertions

```go
import "github.com/justblue/samsa/internal/testkit/assert"

// Check row exists
exists := assert.RowExists(t, pool, "SELECT 1 FROM users WHERE id = $1", userID)

// Count rows
count := assert.CountRows(t, pool, "SELECT COUNT(*) FROM comments WHERE story_id = $1", storyID)

// Check record is deleted
isDeleted := assert.IsDeleted(t, pool, "comments", commentID)

// Compare JSON
assert.JSONEquals(t, expectedMap, actualMap)

// Time comparison with tolerance
assert.TimeEquals(t, expected, actual, 5*time.Second)
```

### HTTP Assertions

```go
import "github.com/justblue/samsa/internal/testkit/assert"

// Status code checks
assert.StatusOK(t, rr)
assert.StatusCreated(t, rr)
assert.StatusNoContent(t, rr)
assert.StatusBadRequest(t, rr)
assert.StatusUnauthorized(t, rr)
assert.StatusNotFound(t, rr)

// Header checks
assert.ContentTypeJSON(t, rr)
assert.HeaderEquals(t, rr, "X-Custom-Header", "value")

// Body checks
assert.BodyContains(t, rr, "error message")
assert.JSONFieldEquals(t, rr, "id", expectedID)
assert.PaginationMeta(t, rr, 100, 10, 0) // total, limit, offset
```

### Domain Assertions

```go
import "github.com/justblue/samsa/internal/testkit/assert"

// UUID checks
assert.ValidUUID(t, id)
assert.UUIDNotNil(t, result.ID)

// Entity validation
assert.ValidComment(t, comment)
assert.ValidFile(t, file)
assert.ValidSubmission(t, submission)
assert.ValidStory(t, story)

// Status checks
assert.StoryStatus(t, story, sqlc.StoryStatusPublished)
assert.SubmissionStatus(t, sub, sqlc.SubmissionStatusApproved)

// Delete state checks
assert.CommentIsDeleted(t, comment)
assert.FileNotDeleted(t, file)
```

## Fixtures

Fixtures provide pre-built test data sets for common scenarios.

### Basic Fixtures

```go
import "github.com/justblue/samsa/internal/testkit/fixtures"

// Create basic users (admin, regular, author, banned)
data := fixtures.BasicUsers(t, db)

// Create stories for authors
stories := fixtures.Stories(t, db, data.Authors, 10)

// Create threaded comments
comments := fixtures.Comments(t, db, stories.Stories[0], data.Users[1], 3)

// Create files
files := fixtures.Files(t, db, data.Users[0], 5)
```

### Scenario Builders

Scenario builders create complex test scenarios with fluent API.

```go
import "github.com/justblue/samsa/internal/testkit/fixtures"

// Build custom scenario
scenario := fixtures.NewScenario(t, db).
    WithUsers(5, func(i int, opts *factory.UserOpts) {
        opts.Email = fmt.Sprintf("user%d@example.com", i)
    }).
    WithAuthors(3, func(i int, opts *factory.AuthorOpts) {
        opts.StageName = "Author " + factory.RandomString(4)
    }).
    WithStories(10, func(i int, opts *factory.StoryOpts) {
        opts.Status = sqlc.StoryStatusPublished
    }).
    Build()

// Pre-built scenarios
modScenario := fixtures.NewScenario(t, db).CommentModerationScenario()
// modScenario.Admin, modScenario.Comments, modScenario.Archived, etc.

fileScenario := fixtures.NewScenario(t, db).FileSharingScenario()
// fileScenario.Owner, fileScenario.SharedFiles, etc.

slaScenario := fixtures.NewScenario(t, db).SLATrackingScenario(24)
// slaScenario.WithinSLA, slaScenario.ExceedingSLA, slaScenario.Pending
```

## Integration Testing

### Test Suite

```go
package comment_test

import (
    "context"
    "net/http"
    "testing"

    "github.com/justblue/samsa/internal/testkit/integration"
)

func TestCommentHandler_Integration(t *testing.T) {
    suite := integration.NewSuite(t)
    suite.Setup()
    defer suite.Teardown()

    // Create handler
    handler := comment.NewHTTPHandler(usecase)

    // Build and execute request
    rr := integration.NewRequest(t, http.MethodPost, "/comments").
        WithJSONBody(jsonBody).
        WithAuth(token).
        Execute(handler.Create)

    // Assert response
    integration.Assert(t, rr).
        Status(http.StatusCreated).
        ContentTypeJSON().
        BodyContains("id")
}
```

### Fixtures Loader

```go
import "github.com/justblue/samsa/internal/testkit/integration"

// Load SQL fixtures
loader := integration.NewFixturesLoader(t, suite.Pool)
err := loader.LoadAll("testdata/fixtures")

// Load single file
err = loader.Load("testdata/fixtures/users.sql")

// Execute raw SQL
err = loader.Execute(`
    INSERT INTO users (email, password_hash) 
    VALUES ('test@example.com', 'hash')
`)

// Truncate tables
err = loader.TruncateTables("comments", "stories")
```

## Helper Utilities

### Random Data Generation

```go
import "github.com/justblue/samsa/internal/testkit/factory"

// Random values
email := factory.RandomEmail("test")           // test-xxxx@example.com
name := factory.RandomName("User")            // User xxxx
slug := factory.RandomSlug("story")           // story-xxxx
phone := factory.RandomPhone()                // +1234567xxx
url := factory.RandomURL("/api/test")         // https://example.com/api/test
str := factory.RandomString(16)               // 16 random chars
```

### Time Helpers

```go
import "github.com/justblue/samsa/internal/testkit/factory"

// Time manipulation
past := factory.TimeAgo(24 * time.Hour)       // 24 hours ago
future := factory.TimeFromNow(1 * time.Hour)  // 1 hour from now
specific := factory.TimeAt(2024, 1, 15, 12, 0, 0)
modified := factory.TimeAdd(&past, 2*time.Hour)
diff := factory.TimeDiff(&t1, &t2)
```

### Pointer Helpers

```go
import "github.com/justblue/samsa/internal/testkit/factory"

// Create pointers easily
boolPtr := factory.BoolPtr(true)
intPtr := factory.IntPtr(42)
int32Ptr := factory.Int32Ptr(100)
int64Ptr := factory.Int64Ptr(1000)
float32Ptr := factory.Float32Ptr(3.14)
stringPtr := factory.StringPtr("hello")
```

## Mock Generation

See [mocks/README.md](mocks/README.md) for detailed mock generation instructions.

```bash
# Generate all mocks
cd server
./scripts/generate_mocks.sh      # Linux/Mac
scripts\generate_mocks.bat       # Windows
```

## Best Practices

1. **Use Transactions**: Always use `testkit.NewTx()` for test isolation
2. **Use Factories**: Don't create test data manually
3. **Clean Assertions**: Use domain-specific assertions for readability
4. **Scenario Builders**: Use for complex multi-entity tests
5. **Table Truncation**: Clean up between test suites if not using transactions
6. **Parallel Tests**: The shared pool supports parallel test execution

## Database Configuration

The testkit connects to the test database using environment variables:

```bash
SAMSA_POSTGRES_USER=samsa
SAMSA_POSTGRES_PWD=samsa
SAMSA_POSTGRES_HOST=localhost
SAMSA_POSTGRES_PORT=5432
SAMSA_POSTGRES_TEST_DATABASE=samsa_test
SAMSA_POSTGRES_SSLMODE=disable
```

## Safety

The testkit includes safety checks to prevent running tests on production databases:

```go
// Check if connected to test database
if !testkit.IsTestDatabase(pool) {
    log.Fatal("Not connected to test database!")
}

// Panic if not test database (use in test setup)
testkit.RequireTestDatabase(t, pool)
```

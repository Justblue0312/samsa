package factory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// AuthorOpts controls which fields are customised when creating a test author.
// Any zero-value field gets a sensible default.
type AuthorOpts struct {
	UserID        uuid.UUID
	StageName     string
	Slug          string
	Gender        string
	IsRecommended *bool
}

// Author inserts an author into the DB and returns the created model.
// If UserID is zero, a new test user is created automatically.
func Author(t *testing.T, db sqlc.DBTX, opts AuthorOpts) *sqlc.Author {
	t.Helper()

	if opts.UserID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.UserID = user.ID
	}
	if opts.StageName == "" {
		opts.StageName = "Test Author " + randID()
	}
	if opts.Slug == "" {
		opts.Slug = "test-author-" + randID()
	}
	if opts.Gender == "" {
		opts.Gender = "other"
	}
	if opts.IsRecommended == nil {
		f := false
		opts.IsRecommended = &f
	}

	acceptedTOS := true
	n := now()

	q := sqlc.New(db)
	isDeleted := false

	firstName := "john"
	lastName := "doe"
	author, err := q.CreateAuthor(context.Background(), sqlc.CreateAuthorParams{
		UserID:                        opts.UserID,
		MediaID:                       nil,
		StageName:                     opts.StageName,
		Gender:                        opts.Gender,
		Slug:                          opts.Slug,
		FirstName:                     &firstName,
		LastName:                      &lastName,
		DOB:                           nil,
		Phone:                         nil,
		Bio:                           nil,
		Description:                   nil,
		AcceptedTermsOfService:        &acceptedTOS,
		EmailNewslettersAndChangelogs: nil,
		EmailPromotionsAndEvents:      nil,
		IsRecommended:                 opts.IsRecommended,
		IsDeleted:                     isDeleted,
		Stats:                         nil,
		CreatedAt:                     n,
		UpdatedAt:                     n,
		DeletedAt:                     nil,
	})
	require.NoError(t, err, "factory: failed to create test author")

	return &author
}

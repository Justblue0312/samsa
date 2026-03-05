package factory

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/stretchr/testify/require"
)

// SubmissionOpts controls which fields are customised when creating a test submission.
// Any zero-value field gets a sensible default.
type SubmissionOpts struct {
	RequesterID uuid.UUID
	ApproverID  *uuid.UUID
	Title       string
	Type        sqlc.SubmissionType
	Status      sqlc.SubmissionStatus
	Message     *string
	Context     map[string]interface{}
	IsDeleted   bool
	ApprovedAt  *time.Time
}

// Submission inserts a submission into the DB and returns the created model.
// If RequesterID is zero, a new test user is created automatically.
func Submission(t *testing.T, db sqlc.DBTX, opts SubmissionOpts) *sqlc.Submission {
	t.Helper()

	if opts.RequesterID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.RequesterID = user.ID
	}
	if opts.Title == "" {
		opts.Title = "Test Submission " + randID()
	}
	if opts.Type == "" {
		opts.Type = sqlc.SubmissionTypeOther
	}
	if opts.Status == "" {
		opts.Status = sqlc.SubmissionStatusPending
	}
	if opts.Context == nil {
		opts.Context = map[string]interface{}{"test": true}
	}

	ctxBytes, err := json.Marshal(opts.Context)
	require.NoError(t, err, "factory: failed to marshal submission context")

	q := sqlc.New(db)
	n := now()

	submission, err := q.CreateSubmission(context.Background(), sqlc.CreateSubmissionParams{
		RequesterID: opts.RequesterID,
		ApproverID:  opts.ApproverID,
		ApprovedAt:  opts.ApprovedAt,
		Message:     opts.Message,
		Title:       opts.Title,
		Type:        opts.Type,
		IsDeleted:   opts.IsDeleted,
		Status:      opts.Status,
		Context:     ctxBytes,
		CreatedAt:   n,
		UpdatedAt:   n,
		DeletedAt:   nil,
	})
	require.NoError(t, err, "factory: failed to create test submission")

	return &submission
}

// SubmissionWithSLAOpts controls which fields are customised when creating a test submission with SLA tracking.
type SubmissionWithSLAOpts struct {
	RequesterID uuid.UUID
	ApproverID  *uuid.UUID
	Title       string
	Type        sqlc.SubmissionType
	Status      sqlc.SubmissionStatus
	CreatedAt   *time.Time
	ApprovedAt  *time.Time
	Message     *string
	Context     map[string]interface{}
}

// SubmissionWithSLA inserts a submission with specific timestamps for SLA testing.
func SubmissionWithSLA(t *testing.T, db sqlc.DBTX, opts SubmissionWithSLAOpts) *sqlc.Submission {
	t.Helper()

	if opts.RequesterID == uuid.Nil {
		user := User(t, db, UserOpts{})
		opts.RequesterID = user.ID
	}
	if opts.Title == "" {
		opts.Title = "Test Submission SLA " + randID()
	}
	if opts.Type == "" {
		opts.Type = sqlc.SubmissionTypeOther
	}
	if opts.Status == "" {
		opts.Status = sqlc.SubmissionStatusPending
	}
	if opts.Context == nil {
		opts.Context = map[string]interface{}{"test": true, "sla": true}
	}

	ctxBytes, err := json.Marshal(opts.Context)
	require.NoError(t, err, "factory: failed to marshal submission context")

	q := sqlc.New(db)

	createdAt := opts.CreatedAt
	if createdAt == nil {
		t := time.Now().Add(-24 * time.Hour).Truncate(time.Second).UTC()
		createdAt = &t
	}

	updatedAt := *createdAt
	if opts.ApprovedAt != nil {
		updatedAt = *opts.ApprovedAt
	}

	submission, err := q.CreateSubmission(context.Background(), sqlc.CreateSubmissionParams{
		RequesterID: opts.RequesterID,
		ApproverID:  opts.ApproverID,
		ApprovedAt:  opts.ApprovedAt,
		Message:     opts.Message,
		Title:       opts.Title,
		Type:        opts.Type,
		IsDeleted:   false,
		Status:      opts.Status,
		Context:     ctxBytes,
		CreatedAt:   createdAt,
		UpdatedAt:   &updatedAt,
		DeletedAt:   nil,
	})
	require.NoError(t, err, "factory: failed to create test submission with SLA")

	return &submission
}

// SubmissionAssignmentOpts controls which fields are customised when creating a test submission assignment.
type SubmissionAssignmentOpts struct {
	SubmissionID uuid.UUID
	AssignedBy   *uuid.UUID
	AssignedTo   *uuid.UUID
}

// SubmissionAssignment inserts a submission assignment into the DB.
func SubmissionAssignment(t *testing.T, db sqlc.DBTX, opts SubmissionAssignmentOpts) *sqlc.SubmissionAssignment {
	t.Helper()

	if opts.SubmissionID == uuid.Nil {
		submission := Submission(t, db, SubmissionOpts{})
		opts.SubmissionID = submission.ID
	}
	if opts.AssignedTo == nil {
		user := User(t, db, UserOpts{})
		opts.AssignedTo = &user.ID
	}
	if opts.AssignedBy == nil {
		user := User(t, db, UserOpts{IsAdmin: true})
		opts.AssignedBy = &user.ID
	}

	q := sqlc.New(db)
	n := *now()

	assignment, err := q.CreateSubmissionAssignment(context.Background(), sqlc.CreateSubmissionAssignmentParams{
		SubmissionID: opts.SubmissionID,
		AssignedBy:   opts.AssignedBy,
		AssignedTo:   opts.AssignedTo,
		AssignedAt:   n,
	})
	require.NoError(t, err, "factory: failed to create test submission assignment")

	return &assignment
}

// SubmissionStatusHistoryOpts controls which fields are customised when creating a test status history.
type SubmissionStatusHistoryOpts struct {
	SubmissionID uuid.UUID
	ChangedBy    *uuid.UUID
	OldStatus    sqlc.SubmissionStatus
	NewStatus    sqlc.SubmissionStatus
	Reason       *string
}

// SubmissionStatusHistory inserts a submission status history record.
func SubmissionStatusHistory(t *testing.T, db sqlc.DBTX, opts SubmissionStatusHistoryOpts) *sqlc.SubmissionStatusHistory {
	t.Helper()

	if opts.SubmissionID == uuid.Nil {
		submission := Submission(t, db, SubmissionOpts{})
		opts.SubmissionID = submission.ID
	}
	if opts.OldStatus == "" {
		opts.OldStatus = sqlc.SubmissionStatusPending
	}
	if opts.NewStatus == "" {
		opts.NewStatus = sqlc.SubmissionStatusApproved
	}
	if opts.ChangedBy == nil {
		user := User(t, db, UserOpts{IsAdmin: true})
		opts.ChangedBy = &user.ID
	}

	q := sqlc.New(db)
	n := *now()

	history, err := q.CreateSubmissionStatusHistory(context.Background(), sqlc.CreateSubmissionStatusHistoryParams{
		SubmissionID: opts.SubmissionID,
		ChangedBy:    opts.ChangedBy,
		OldStatus:    opts.OldStatus,
		NewStatus:    opts.NewStatus,
		Reason:       opts.Reason,
		CreatedAt:    n,
	})
	require.NoError(t, err, "factory: failed to create test submission status history")

	return &history
}

package submission

import (
	"errors"
	"slices"

	"github.com/justblue/samsa/gen/sqlc"
)

// Sentinel errors for the submission domain.
var (
	ErrNotFound          = errors.New("submission not found")
	ErrNotRequester      = errors.New("only requester can perform this action")
	ErrNotApprover       = errors.New("only approver can perform this action")
	ErrNotAssignee       = errors.New("only assignee can perform this action")
	ErrAlreadyApproved   = errors.New("submission already approved")
	ErrAlreadyRejected   = errors.New("submission already rejected")
	ErrAlreadyClaimed    = errors.New("submission already claimed")
	ErrNotPending        = errors.New("submission is not pending")
	ErrUnauthorized      = errors.New("unauthorized to perform this action")
	ErrInvalidContext    = errors.New("invalid submission context")
	ErrInvalidStatus     = errors.New("invalid submission status")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// ValidStatusTransitions defines allowed status transitions.
// Key: current status → Value: allowed next statuses.
var ValidStatusTransitions = map[sqlc.SubmissionStatus][]sqlc.SubmissionStatus{
	sqlc.SubmissionStatusPending:   {sqlc.SubmissionStatusClaimed, sqlc.SubmissionStatusAssigned, sqlc.SubmissionStatusApproved, sqlc.SubmissionStatusRejected, sqlc.SubmissionStatusTimeouted},
	sqlc.SubmissionStatusClaimed:   {sqlc.SubmissionStatusApproved, sqlc.SubmissionStatusRejected, sqlc.SubmissionStatusArchived},
	sqlc.SubmissionStatusAssigned:  {sqlc.SubmissionStatusApproved, sqlc.SubmissionStatusRejected, sqlc.SubmissionStatusArchived},
	sqlc.SubmissionStatusApproved:  {sqlc.SubmissionStatusArchived},
	sqlc.SubmissionStatusRejected:  {sqlc.SubmissionStatusArchived},
	sqlc.SubmissionStatusTimeouted: {sqlc.SubmissionStatusArchived},
	sqlc.SubmissionStatusArchived:  {}, // Terminal state
}

// IsValidTransition reports whether transitioning from → to is permitted.
func IsValidTransition(from, to sqlc.SubmissionStatus) bool {
	allowed, ok := ValidStatusTransitions[from]
	if !ok {
		return false
	}
	return slices.Contains(allowed, to)
}

// IsTerminalStatus reports whether a status cannot transition further.
func IsTerminalStatus(s sqlc.SubmissionStatus) bool {
	transitions, ok := ValidStatusTransitions[s]
	return !ok || len(transitions) == 0
}

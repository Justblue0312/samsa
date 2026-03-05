package submission

import (
	"time"

	"github.com/google/uuid"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

type SubmissionContext struct {
	RequestType   string         `json:"request_type"`
	Justification string         `json:"justification"`
	Priority      string         `json:"priority"`
	Documents     []ContextDoc   `json:"documents"`
	Metadata      map[string]any `json:"metadata"`
	Deadline      *time.Time     `json:"deadline"`
}

type ContextDoc struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

type CreateSubmissionRequest struct {
	RequesterID uuid.UUID           `json:"requester_id" validate:"required"`
	ApproverID  uuid.UUID           `json:"approver_id"`
	Title       string              `json:"title" validate:"required"`
	Type        sqlc.SubmissionType `json:"type" validate:"required"`
	Message     string              `json:"message"`
	Context     SubmissionContext   `json:"context"`
	Tags        []string            `json:"tags" validate:"max=10"`
	AssigneeIDs []uuid.UUID         `json:"assignee_ids"`
}

type ApproveSubmissionRequest struct {
	Reason string `json:"reason"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason"`
}

type AssignSubmissionRequest struct {
	AssignedTo uuid.UUID `json:"assigned_to" validate:"required"`
}

type ClaimSubmissionRequest struct {
	// Currently empty, could add reason in future
}

type UpdateSubmissionRequest struct {
	Title      string              `json:"title"`
	Type       sqlc.SubmissionType `json:"type"`
	Message    string              `json:"message"`
	Context    SubmissionContext   `json:"context"`
	ApproverID *uuid.UUID          `json:"approver_id"`
	Tags       *[]string           `json:"tags" validate:"max=10"`
}

type UpdateContextRequest struct {
	Context SubmissionContext `json:"context" validate:"required"`
}

type BulkUpdateStatusRequest struct {
	IDs    []uuid.UUID           `json:"ids" validate:"required"`
	Status sqlc.SubmissionStatus `json:"status" validate:"required"`
}

// SubmissionResponse is the API response for a submission.
// ExposeID (e.g. "SUB-0001") is included via the embedded sqlc.Submission.
type SubmissionResponse struct {
	sqlc.Submission `json:"submission"`
	StatusHistory   []sqlc.SubmissionStatusHistory `json:"status_history,omitempty"`
}

type SubmissionResponses struct {
	Submissions []SubmissionResponse      `json:"submissions"`
	Meta        queryparam.PaginationMeta `json:"meta"`
}

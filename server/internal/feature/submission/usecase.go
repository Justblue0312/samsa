package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justblue/samsa/config"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/internal/common"
	"github.com/justblue/samsa/internal/feature/author"
	"github.com/justblue/samsa/internal/feature/notification"
	submission_assignment "github.com/justblue/samsa/internal/feature/submission_assignment"
	submission_status_history "github.com/justblue/samsa/internal/feature/submission_status_history"
	"github.com/justblue/samsa/internal/feature/tag"
	"github.com/justblue/samsa/internal/feature/user"
)

type SubmissionType string

const (
	AuthorRequest   SubmissionType = "author_request"
	StoryApproval   SubmissionType = "story_approval"
	ChapterApproval SubmissionType = "chapter_approval"
	Other           SubmissionType = "other"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	Create(ctx context.Context, req *CreateSubmissionRequest) (*sqlc.Submission, error)
	GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	GetByExposeID(ctx context.Context, exposeID string) (*sqlc.Submission, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateSubmissionRequest) (*sqlc.Submission, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	Archive(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	List(ctx context.Context, f *SubmissionFilter) ([]*sqlc.Submission, int64, error)
	GetMySubmissions(ctx context.Context, requesterID uuid.UUID, f *SubmissionFilter) ([]*sqlc.Submission, int64, error)
	Approve(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	Reject(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error)
	UpdateContext(ctx context.Context, id uuid.UUID, ctxData SubmissionContext) (*sqlc.Submission, error)
	GetContext(ctx context.Context, id uuid.UUID) (*SubmissionContext, error)
	GetTags(ctx context.Context, entityID uuid.UUID) ([]*sqlc.Tag, error)
	BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status sqlc.SubmissionStatus) error
	Claim(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*sqlc.Submission, error)
	Assign(ctx context.Context, id uuid.UUID, assignedBy, assignedTo uuid.UUID) (*sqlc.Submission, error)
	ApproveWithReason(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) (*sqlc.Submission, error)
	RejectWithReason(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) (*sqlc.Submission, error)
	GetAssignment(ctx context.Context, submissionID uuid.UUID) (*sqlc.SubmissionAssignment, error)
	ListStatusHistory(ctx context.Context, submissionID uuid.UUID) ([]sqlc.SubmissionStatusHistory, error)
	GetAvailable(ctx context.Context, limit, offset int) ([]*sqlc.Submission, int64, error)
	// SLA tracking
	GetSubmissionsExceedingSLA(ctx context.Context, slaHours int) ([]*sqlc.Submission, error)
	CountSubmissionsExceedingSLA(ctx context.Context, slaHours int) (int64, error)
	GetSLAComplianceStats(ctx context.Context, slaHours int) (*SLAComplianceStats, error)
	GetAverageProcessingTime(ctx context.Context, days int) (float64, error)
	GetSubmissionsBySLAStatus(ctx context.Context, includeCompliant bool, slaHours int, limit, offset int32) ([]*sqlc.Submission, error)
	GetPendingDuration(ctx context.Context, id uuid.UUID) (int, error)
	BulkUpdateSLABreach(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error)
}

type usecase struct {
	pool           *pgxpool.Pool
	q              *sqlc.Queries
	cfg            *config.Config
	submissionRepo Repository
	tagRepo        tag.Repository
	authorRepo     author.Repository
	userRepo       user.Repository
	notifRepo      notification.Repository
	assignRepo     submission_assignment.Repository
	historyRepo    submission_status_history.Repository
	notifier       Notifier
}

func NewUseCase(
	pool *pgxpool.Pool,
	q *sqlc.Queries,
	cfg *config.Config,
	submissionRepo Repository,
	tagRepo tag.Repository,
	authorRepo author.Repository,
	userRepo user.Repository,
	notifRepo notification.Repository,
	assignRepo submission_assignment.Repository,
	historyRepo submission_status_history.Repository,
	notifier Notifier,
) UseCase {
	return &usecase{
		pool:           pool,
		q:              q,
		cfg:            cfg,
		submissionRepo: submissionRepo,
		tagRepo:        tagRepo,
		authorRepo:     authorRepo,
		userRepo:       userRepo,
		notifRepo:      notifRepo,
		assignRepo:     assignRepo,
		historyRepo:    historyRepo,
		notifier:       notifier,
	}
}

// recordHistory creates a status history entry; errors are non-fatal (best-effort).
func (u *usecase) recordHistory(ctx context.Context, submissionID uuid.UUID, changedBy *uuid.UUID, from, to sqlc.SubmissionStatus, reason *string) {
	_, _ = u.historyRepo.Create(ctx, &sqlc.SubmissionStatusHistory{
		ID:           uuid.New(),
		SubmissionID: submissionID,
		ChangedBy:    changedBy,
		OldStatus:    from,
		NewStatus:    to,
		Reason:       reason,
		CreatedAt:    time.Now(),
	})
}

// getOrNotFound wraps GetByID with ErrNotFound mapping.
func (u *usecase) getOrNotFound(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	s, err := u.submissionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

// Create implements [UseCase].
func (u *usecase) Create(ctx context.Context, req *CreateSubmissionRequest) (*sqlc.Submission, error) {
	now := time.Now()

	// Validate requester exists
	if _, err := u.userRepo.GetByID(ctx, req.RequesterID, false); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("requester not found")
		}
		return nil, err
	}

	// Validate approver if provided
	if req.ApproverID != uuid.Nil {
		if _, err := u.userRepo.GetByID(ctx, req.ApproverID, false); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("approver not found")
			}
			return nil, err
		}
	}

	contextBytes, err := json.Marshal(req.Context)
	if err != nil {
		return nil, ErrInvalidContext
	}

	var approverID *uuid.UUID
	if req.ApproverID != uuid.Nil {
		approverID = &req.ApproverID
	}

	result, err := u.submissionRepo.Create(ctx, &sqlc.Submission{
		ID:          uuid.New(),
		RequesterID: req.RequesterID,
		ApproverID:  approverID,
		Message:     &req.Message,
		Title:       req.Title,
		Type:        req.Type,
		Status:      sqlc.SubmissionStatusPending,
		Context:     contextBytes,
		IsDeleted:   false,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		return nil, err
	}

	// Record initial status history
	u.recordHistory(ctx, result.ID, &req.RequesterID, "", sqlc.SubmissionStatusPending, nil)

	// Create tags
	for _, tagName := range req.Tags {
		color, _ := common.GenerateRandomColor()
		_, _ = u.tagRepo.UpsertTag(ctx, &sqlc.Tag{
			ID:            uuid.New(),
			OwnerID:       req.RequesterID,
			Name:          tagName,
			Color:         color,
			EntityType:    sqlc.EntityTypeSubmission,
			EntityID:      result.ID,
			IsHidden:      common.Ptr(false),
			IsSystem:      common.Ptr(false),
			IsRecommended: common.Ptr(false),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		})
	}

	// Create assignments and notify
	for _, assigneeID := range req.AssigneeIDs {
		assigneeID := assigneeID
		_, _ = u.assignRepo.Create(ctx, &sqlc.SubmissionAssignment{
			ID:           uuid.New(),
			SubmissionID: result.ID,
			AssignedBy:   &req.RequesterID,
			AssignedTo:   &assigneeID,
			AssignedAt:   now,
			CreatedAt:    now,
		})
		_ = u.notifier.NotifyAssignment(ctx, result, assigneeID, req.RequesterID)
	}

	return result, nil
}

// GetByID implements [UseCase].
func (u *usecase) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	return u.getOrNotFound(ctx, id)
}

// GetByExposeID implements [UseCase].
func (u *usecase) GetByExposeID(ctx context.Context, exposeID string) (*sqlc.Submission, error) {
	s, err := u.submissionRepo.GetByExposeID(ctx, exposeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

// Update implements [UseCase].
func (u *usecase) Update(ctx context.Context, id uuid.UUID, req *UpdateSubmissionRequest) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		submission.Title = req.Title
	}
	if req.Type != "" {
		submission.Type = req.Type
	}
	if req.Message != "" {
		submission.Message = &req.Message
	}
	if req.ApproverID != nil {
		submission.ApproverID = req.ApproverID
	}

	if req.Context.RequestType != "" || req.Context.Justification != "" || len(req.Context.Documents) > 0 {
		contextBytes, err := json.Marshal(req.Context)
		if err != nil {
			return nil, ErrInvalidContext
		}
		submission.Context = contextBytes
	}

	return u.submissionRepo.Update(ctx, submission)
}

// Delete implements [UseCase].
func (u *usecase) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := u.getOrNotFound(ctx, id); err != nil {
		return err
	}
	return u.submissionRepo.Delete(ctx, id)
}

// List implements [UseCase].
func (u *usecase) List(ctx context.Context, f *SubmissionFilter) ([]*sqlc.Submission, int64, error) {
	return u.submissionRepo.List(ctx, f)
}

// GetMySubmissions implements [UseCase].
func (u *usecase) GetMySubmissions(ctx context.Context, requesterID uuid.UUID, f *SubmissionFilter) ([]*sqlc.Submission, int64, error) {
	f.RequesterIDs = []uuid.UUID{requesterID}
	return u.submissionRepo.List(ctx, f)
}

// Approve implements [UseCase].
func (u *usecase) Approve(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	return u.ApproveWithReason(ctx, id, uuid.Nil, "")
}

// Reject implements [UseCase].
func (u *usecase) Reject(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	return u.RejectWithReason(ctx, id, uuid.Nil, "")
}

// ApproveWithReason implements [UseCase].
func (u *usecase) ApproveWithReason(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}
	if submission.Status != sqlc.SubmissionStatusPending &&
		submission.Status != sqlc.SubmissionStatusClaimed &&
		submission.Status != sqlc.SubmissionStatusAssigned {
		return nil, ErrNotPending
	}
	if submission.Status == sqlc.SubmissionStatusApproved {
		return nil, ErrAlreadyApproved
	}

	now := time.Now()
	result, err := u.q.UpdateSubmissionApproved(ctx, sqlc.UpdateSubmissionApprovedParams{
		ID:         id,
		ApprovedAt: &now,
		ApproverID: &approverID,
	})
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, id, &approverID, submission.Status, sqlc.SubmissionStatusApproved, &reason)
	_ = u.notifier.NotifyApproval(ctx, &result, approverID, reason)
	return &result, nil
}

// RejectWithReason implements [UseCase].
func (u *usecase) RejectWithReason(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}
	if submission.Status == sqlc.SubmissionStatusRejected {
		return nil, ErrAlreadyRejected
	}
	if !IsValidTransition(submission.Status, sqlc.SubmissionStatusRejected) {
		return nil, ErrInvalidTransition
	}

	result, err := u.submissionRepo.UpdateStatus(ctx, id, sqlc.SubmissionStatusRejected)
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, id, &approverID, submission.Status, sqlc.SubmissionStatusRejected, &reason)
	_ = u.notifier.NotifyRejection(ctx, result, approverID, reason)
	return result, nil
}

// UpdateContext implements [UseCase].
func (u *usecase) UpdateContext(ctx context.Context, id uuid.UUID, ctxData SubmissionContext) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}

	contextBytes, err := json.Marshal(ctxData)
	if err != nil {
		return nil, ErrInvalidContext
	}
	submission.Context = contextBytes

	return u.submissionRepo.Update(ctx, submission)
}

// GetContext implements [UseCase].
func (u *usecase) GetContext(ctx context.Context, id uuid.UUID) (*SubmissionContext, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}

	var submissionCtx SubmissionContext
	if err := json.Unmarshal(submission.Context, &submissionCtx); err != nil {
		return nil, ErrInvalidContext
	}
	return &submissionCtx, nil
}

// GetTags implements [UseCase].
func (u *usecase) GetTags(ctx context.Context, entityID uuid.UUID) ([]*sqlc.Tag, error) {
	return u.tagRepo.GetTagsByEntityID(ctx, entityID, sqlc.EntityTypeSubmission, nil, nil, nil)
}

// BulkUpdateStatus implements [UseCase].
func (u *usecase) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status sqlc.SubmissionStatus) error {
	// Validate all transitions first
	for _, id := range ids {
		submission, err := u.getOrNotFound(ctx, id)
		if err != nil {
			return err
		}
		if !IsValidTransition(submission.Status, status) {
			return fmt.Errorf("%w: %s → %s for submission %s", ErrInvalidTransition, submission.Status, status, id)
		}
	}

	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, id := range ids {
		switch status {
		case sqlc.SubmissionStatusApproved:
			err = u.approveInTx(ctx, tx, id, uuid.Nil, "")
		case sqlc.SubmissionStatusRejected:
			err = u.rejectInTx(ctx, tx, id, uuid.Nil, "")
		default:
			err = errors.New("unsupported status for bulk update")
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (u *usecase) approveInTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, approverID uuid.UUID, reason string) error {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return err
	}
	if submission.Status == sqlc.SubmissionStatusApproved {
		return ErrAlreadyApproved
	}
	if !IsValidTransition(submission.Status, sqlc.SubmissionStatusApproved) {
		return ErrInvalidTransition
	}

	now := time.Now()
	result, err := u.q.UpdateSubmissionApproved(ctx, sqlc.UpdateSubmissionApprovedParams{
		ID:         id,
		ApprovedAt: &now,
		ApproverID: &approverID,
	})
	if err != nil {
		return err
	}

	u.recordHistory(ctx, id, &approverID, submission.Status, sqlc.SubmissionStatusApproved, &reason)
	_ = u.notifier.NotifyApproval(ctx, &result, approverID, reason)
	return nil
}

func (u *usecase) rejectInTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, approverID uuid.UUID, reason string) error {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return err
	}
	if !IsValidTransition(submission.Status, sqlc.SubmissionStatusRejected) {
		return ErrInvalidTransition
	}

	result, err := u.submissionRepo.UpdateStatus(ctx, id, sqlc.SubmissionStatusRejected)
	if err != nil {
		return err
	}

	u.recordHistory(ctx, id, &approverID, submission.Status, sqlc.SubmissionStatusRejected, &reason)
	_ = u.notifier.NotifyRejection(ctx, result, approverID, reason)
	return nil
}

// Claim implements [UseCase].
func (u *usecase) Claim(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}
	if submission.Status != sqlc.SubmissionStatusPending {
		return nil, ErrNotPending
	}

	assignment, _ := u.assignRepo.GetBySubmissionID(ctx, id)
	if assignment != nil && assignment.AssignedTo != nil {
		return nil, ErrAlreadyClaimed
	}

	now := time.Now()
	_, err = u.assignRepo.Create(ctx, &sqlc.SubmissionAssignment{
		ID:           uuid.New(),
		SubmissionID: id,
		AssignedBy:   &userID,
		AssignedTo:   &userID,
		AssignedAt:   now,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, err
	}

	// Update approver + status
	submission.ApproverID = &userID
	submission.Status = sqlc.SubmissionStatusClaimed
	result, err := u.submissionRepo.Update(ctx, submission)
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, id, &userID, sqlc.SubmissionStatusPending, sqlc.SubmissionStatusClaimed, nil)
	_ = u.notifier.NotifyClaim(ctx, result, userID)
	return result, nil
}

// Assign implements [UseCase].
func (u *usecase) Assign(ctx context.Context, id uuid.UUID, assignedBy, assignedTo uuid.UUID) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = u.assignRepo.Create(ctx, &sqlc.SubmissionAssignment{
		ID:           uuid.New(),
		SubmissionID: id,
		AssignedBy:   &assignedBy,
		AssignedTo:   &assignedTo,
		AssignedAt:   now,
		CreatedAt:    now,
	})
	if err != nil {
		return nil, err
	}

	oldStatus := submission.Status
	submission.ApproverID = &assignedTo
	submission.Status = sqlc.SubmissionStatusAssigned
	result, err := u.submissionRepo.Update(ctx, submission)
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, id, &assignedBy, oldStatus, sqlc.SubmissionStatusAssigned, nil)
	_ = u.notifier.NotifyAssignment(ctx, result, assignedTo, assignedBy)
	return result, nil
}

// GetAssignment implements [UseCase].
func (u *usecase) GetAssignment(ctx context.Context, submissionID uuid.UUID) (*sqlc.SubmissionAssignment, error) {
	return u.assignRepo.GetBySubmissionID(ctx, submissionID)
}

// ListStatusHistory implements [UseCase].
func (u *usecase) ListStatusHistory(ctx context.Context, submissionID uuid.UUID) ([]sqlc.SubmissionStatusHistory, error) {
	return u.historyRepo.GetBySubmissionID(ctx, submissionID)
}

// GetAvailable implements [UseCase].
func (u *usecase) GetAvailable(ctx context.Context, limit, offset int) ([]*sqlc.Submission, int64, error) {
	return u.submissionRepo.GetAvailable(ctx, int32(limit), int32(offset))
}

// SoftDelete implements [UseCase].
func (u *usecase) SoftDelete(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := submission.Status
	result, err := u.submissionRepo.SoftDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, result.ID, nil, oldStatus, sqlc.SubmissionStatusArchived, common.Ptr("Submission soft deleted"))
	return result, nil
}

// Archive implements [UseCase].
func (u *usecase) Archive(ctx context.Context, id uuid.UUID) (*sqlc.Submission, error) {
	submission, err := u.getOrNotFound(ctx, id)
	if err != nil {
		return nil, err
	}
	if !IsValidTransition(submission.Status, sqlc.SubmissionStatusArchived) {
		return nil, ErrInvalidTransition
	}

	oldStatus := submission.Status
	result, err := u.submissionRepo.UpdateStatus(ctx, id, sqlc.SubmissionStatusArchived)
	if err != nil {
		return nil, err
	}

	u.recordHistory(ctx, id, nil, oldStatus, sqlc.SubmissionStatusArchived, common.Ptr("Submission archived"))
	return result, nil
}

// SLAComplianceStats represents SLA compliance statistics.
type SLAComplianceStats struct {
	CompliantCount    int32   `json:"compliant_count"`
	NonCompliantCount int32   `json:"non_compliant_count"`
	TotalPending      int32   `json:"total_pending"`
	ComplianceRate    float64 `json:"compliance_rate"`
}

// GetSubmissionsExceedingSLA implements [UseCase].
func (u *usecase) GetSubmissionsExceedingSLA(ctx context.Context, slaHours int) ([]*sqlc.Submission, error) {
	return u.submissionRepo.GetSubmissionsExceedingSLA(ctx, slaHours)
}

// CountSubmissionsExceedingSLA implements [UseCase].
func (u *usecase) CountSubmissionsExceedingSLA(ctx context.Context, slaHours int) (int64, error) {
	return u.submissionRepo.CountSubmissionsExceedingSLA(ctx, slaHours)
}

// GetSLAComplianceStats implements [UseCase].
func (u *usecase) GetSLAComplianceStats(ctx context.Context, slaHours int) (*SLAComplianceStats, error) {
	stats, err := u.submissionRepo.GetSLAComplianceStats(ctx, slaHours)
	if err != nil {
		return nil, err
	}

	complianceRate := float64(0)
	total := stats.CompliantCount + stats.NonCompliantCount
	if total > 0 {
		complianceRate = float64(stats.CompliantCount) / float64(total) * 100
	}

	return &SLAComplianceStats{
		CompliantCount:    stats.CompliantCount,
		NonCompliantCount: stats.NonCompliantCount,
		TotalPending:      stats.TotalPending,
		ComplianceRate:    complianceRate,
	}, nil
}

// GetAverageProcessingTime implements [UseCase].
func (u *usecase) GetAverageProcessingTime(ctx context.Context, days int) (float64, error) {
	return u.submissionRepo.GetAverageProcessingTime(ctx, days)
}

// GetSubmissionsBySLAStatus implements [UseCase].
func (u *usecase) GetSubmissionsBySLAStatus(ctx context.Context, includeCompliant bool, slaHours int, limit, offset int32) ([]*sqlc.Submission, error) {
	return u.submissionRepo.GetSubmissionsBySLAStatus(ctx, includeCompliant, slaHours, limit, offset)
}

// GetPendingDuration implements [UseCase].
func (u *usecase) GetPendingDuration(ctx context.Context, id uuid.UUID) (int, error) {
	return u.submissionRepo.GetPendingDuration(ctx, id)
}

// BulkUpdateSLABreach implements [UseCase].
func (u *usecase) BulkUpdateSLABreach(ctx context.Context, ids []uuid.UUID) ([]*sqlc.Submission, error) {
	return u.submissionRepo.BulkUpdateSLABreach(ctx, ids)
}

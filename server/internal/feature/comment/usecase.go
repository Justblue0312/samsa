package comment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
	"github.com/justblue/samsa/pkg/queryparam"
)

var (
	ErrNotFound       = errors.New("comment not found")
	ErrParentNotFound = errors.New("parent comment not found")
	ErrEntityMismatch = errors.New("parent comment belongs to different entity")
	ErrDepthExceeded  = errors.New("nesting depth exceeded (max 3)")
	ErrNotOwner       = errors.New("not the owner of this comment")
	ErrAlreadyDeleted = errors.New("comment already deleted")
	ErrNotModerator   = errors.New("only moderators can perform this action")
)

type ModerationAction string

const (
	ModerationActionReport  ModerationAction = "report"
	ModerationActionRestore ModerationAction = "restore"
	ModerationActionPin     ModerationAction = "pin"
	ModerationActionUnpin   ModerationAction = "unpin"
	ModerationActionResolve ModerationAction = "resolve"
	ModerationActionArchive ModerationAction = "archive"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

type UseCase interface {
	Create(ctx context.Context, userID uuid.UUID, req *CreateCommentRequest) (*CommentResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, entityType string, includeDeleted bool) (*CommentResponse, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateCommentRequest) (*CommentResponse, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool) error
	List(ctx context.Context, filter *CommentFilter) (*CommentListResponse, error)
	Moderate(ctx context.Context, id uuid.UUID, action ModerationAction, userID uuid.UUID, isModerator bool) (*CommentResponse, error)
	// Bulk moderation
	BulkDelete(ctx context.Context, ids []uuid.UUID, entityType string, userID uuid.UUID, isModerator bool) ([]CommentResponse, error)
	BulkArchive(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error)
	BulkResolve(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error)
	BulkPin(ctx context.Context, ids []uuid.UUID, entityType string, userID uuid.UUID, isModerator bool) ([]CommentResponse, error)
	BulkUnpin(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error)
	// Search and filtering
	Search(ctx context.Context, entityType string, entityID uuid.UUID, search string, limit, offset int32) (*CommentListResponse, error)
	ListWithFilters(ctx context.Context, entityType string, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported, isPinned *bool, parentID *uuid.UUID, limit, offset int32) (*CommentListResponse, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]CommentResponse, error)
}

type usecase struct {
	r            Repository
	voteRepo     VoteRepository
	reactionRepo ReactionRepository
}

type VoteRepository interface {
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.VoteType]int32, error)
	CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

type ReactionRepository interface {
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.ReactionType]int32, error)
	CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

func NewUseCase(r Repository, voteRepo VoteRepository, reactionRepo ReactionRepository) UseCase {
	return &usecase{
		r:            r,
		voteRepo:     voteRepo,
		reactionRepo: reactionRepo,
	}
}

func (u *usecase) Create(ctx context.Context, userID uuid.UUID, req *CreateCommentRequest) (*CommentResponse, error) {
	entityType := sqlc.EntityType(req.EntityType)

	if req.ParentID != nil {
		includedDeleted := false
		parent, err := u.r.GetByID(ctx, *req.ParentID, entityType, &includedDeleted)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrParentNotFound
			}
			return nil, err
		}

		if parent.EntityID != req.EntityID {
			return nil, ErrEntityMismatch
		}

		depth, err := u.r.GetNestingDepth(ctx, *req.ParentID, entityType, &includedDeleted)
		if err != nil {
			return nil, err
		}

		if depth >= 3 {
			return nil, ErrDepthExceeded
		}
	}

	now := time.Now()
	depth := int32(0)
	score := float32(0)
	replyCount := int32(0)
	reactionCount := int32(0)

	if req.ParentID != nil {
		includedDeleted := false
		parent, err := u.r.GetByID(ctx, *req.ParentID, entityType, &includedDeleted)
		if err == nil && parent.Depth != nil {
			depth = *parent.Depth + 1
		}
	}

	comment := &sqlc.Comment{
		UserID:        userID,
		ParentID:      req.ParentID,
		Content:       []byte(req.Content),
		Depth:         &depth,
		Score:         &score,
		IsDeleted:     boolPtr(false),
		IsResolved:    boolPtr(false),
		IsArchived:    boolPtr(false),
		IsReported:    boolPtr(false),
		IsPinned:      boolPtr(false),
		EntityType:    entityType,
		EntityID:      req.EntityID,
		Source:        strPtr(req.Source),
		ReplyCount:    &replyCount,
		ReactionCount: &reactionCount,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	result, err := u.r.Create(ctx, comment)
	if err != nil {
		return nil, err
	}

	return u.buildResponse(ctx, result)
}

func (u *usecase) GetByID(ctx context.Context, id uuid.UUID, entityType string, includeDeleted bool) (*CommentResponse, error) {
	comment, err := u.r.GetByID(ctx, id, sqlc.EntityType(entityType), &includeDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if comment.IsDeleted != nil && *comment.IsDeleted && !includeDeleted {
		return nil, ErrNotFound
	}

	return u.buildResponse(ctx, comment)
}

func (u *usecase) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateCommentRequest) (*CommentResponse, error) {
	includedDeleted := false
	comment, err := u.r.GetByID(ctx, id, "", &includedDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if comment.UserID != userID {
		return nil, ErrNotOwner
	}

	comment.Content = []byte(req.Content)
	if req.Source != "" {
		comment.Source = &req.Source
	}
	now := time.Now()
	comment.UpdatedAt = &now

	result, err := u.r.Update(ctx, comment)
	if err != nil {
		return nil, err
	}

	return u.buildResponse(ctx, result)
}

func (u *usecase) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, isModerator bool) error {
	includedDeleted := false
	comment, err := u.r.GetByID(ctx, id, "", &includedDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if comment.UserID != userID && !isModerator {
		return ErrNotOwner
	}

	if comment.IsDeleted != nil && *comment.IsDeleted {
		return ErrAlreadyDeleted
	}

	_, err = u.r.SoftDelete(ctx, id, comment.EntityType)
	return err
}

func (u *usecase) List(ctx context.Context, filter *CommentFilter) (*CommentListResponse, error) {
	comments, err := u.r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	entityType := sqlc.EntityType(filter.EntityType)

	var totalCount int64
	if len(comments) > 0 {
		comment := comments[0]
		totalCount, err = u.reactionRepo.CountTotal(ctx, comment.ID, entityType)
		if err != nil {
			return nil, err
		}
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}

	return &CommentListResponse{
		Comments: responses,
		Meta:     queryparam.NewPaginationMeta(filter.Page, filter.Limit, totalCount),
	}, nil
}

func (u *usecase) Moderate(ctx context.Context, id uuid.UUID, action ModerationAction, userID uuid.UUID, isModerator bool) (*CommentResponse, error) {
	if !isModerator {
		return nil, ErrNotModerator
	}

	includedDeleted := true
	comment, err := u.r.GetByID(ctx, id, "", &includedDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	now := time.Now()

	switch action {
	case ModerationActionReport:
		comment.IsReported = boolPtr(true)
		comment.ReportedAt = &now
		comment.ReportedBy = &userID
	case ModerationActionRestore:
		comment.IsReported = boolPtr(false)
		comment.ReportedAt = nil
		comment.ReportedBy = nil
	case ModerationActionPin:
		comment.IsPinned = boolPtr(true)
		comment.PinnedAt = &now
		comment.PinnedBy = &userID
	case ModerationActionUnpin:
		comment.IsPinned = boolPtr(false)
		comment.PinnedAt = nil
		comment.PinnedBy = nil
	case ModerationActionResolve:
		comment.IsResolved = boolPtr(true)
	case ModerationActionArchive:
		comment.IsArchived = boolPtr(true)
	}

	comment.UpdatedAt = &now

	result, err := u.r.Update(ctx, comment)
	if err != nil {
		return nil, err
	}

	return u.buildResponse(ctx, result)
}

func (u *usecase) buildResponse(ctx context.Context, comment *sqlc.Comment) (*CommentResponse, error) {
	voteCounts, err := u.voteRepo.CountByCommentID(ctx, comment.ID, comment.EntityType)
	if err != nil {
		return nil, err
	}

	totalVotes, err := u.voteRepo.CountTotal(ctx, comment.ID, comment.EntityType)
	if err != nil {
		return nil, err
	}

	reactionCounts, err := u.reactionRepo.CountByCommentID(ctx, comment.ID, comment.EntityType)
	if err != nil {
		return nil, err
	}

	totalReactions, err := u.reactionRepo.CountTotal(ctx, comment.ID, comment.EntityType)
	if err != nil {
		return nil, err
	}

	voteDetails := make(map[sqlc.VoteType]int64, len(voteCounts))
	for k, v := range voteCounts {
		voteDetails[k] = int64(v)
	}

	reactionDetails := make(map[sqlc.ReactionType]int64, len(reactionCounts))
	for k, v := range reactionCounts {
		reactionDetails[k] = int64(v)
	}

	return &CommentResponse{
		Comment:         *comment,
		ReactionCount:   int32(totalReactions),
		ReactionDetails: reactionDetails,
		VoteCount:       int32(totalVotes),
		VoteDetails:     voteDetails,
	}, nil
}

// BulkDelete implements [UseCase].
func (u *usecase) BulkDelete(ctx context.Context, ids []uuid.UUID, entityType string, userID uuid.UUID, isModerator bool) ([]CommentResponse, error) {
	comments, err := u.r.BulkDelete(ctx, ids, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// BulkArchive implements [UseCase].
func (u *usecase) BulkArchive(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error) {
	if !isModerator {
		return nil, ErrNotModerator
	}

	comments, err := u.r.BulkArchive(ctx, ids)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// BulkResolve implements [UseCase].
func (u *usecase) BulkResolve(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error) {
	if !isModerator {
		return nil, ErrNotModerator
	}

	comments, err := u.r.BulkResolve(ctx, ids)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// BulkPin implements [UseCase].
func (u *usecase) BulkPin(ctx context.Context, ids []uuid.UUID, entityType string, userID uuid.UUID, isModerator bool) ([]CommentResponse, error) {
	if !isModerator {
		return nil, ErrNotModerator
	}

	comments, err := u.r.BulkPin(ctx, ids, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// BulkUnpin implements [UseCase].
func (u *usecase) BulkUnpin(ctx context.Context, ids []uuid.UUID, entityType string, isModerator bool) ([]CommentResponse, error) {
	if !isModerator {
		return nil, ErrNotModerator
	}

	comments, err := u.r.BulkUnpin(ctx, ids)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

// Search implements [UseCase].
func (u *usecase) Search(ctx context.Context, entityType string, entityID uuid.UUID, search string, limit, offset int32) (*CommentListResponse, error) {
	comments, err := u.r.Search(ctx, sqlc.EntityType(entityType), entityID, search, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}

	return &CommentListResponse{
		Comments: responses,
		Meta:     queryparam.NewPaginationMeta(1, limit, int64(len(comments))),
	}, nil
}

// ListWithFilters implements [UseCase].
func (u *usecase) ListWithFilters(ctx context.Context, entityType string, entityID uuid.UUID, isDeleted, isResolved, isArchived, isReported, isPinned *bool, parentID *uuid.UUID, limit, offset int32) (*CommentListResponse, error) {
	comments, err := u.r.ListWithFilters(ctx, sqlc.EntityType(entityType), entityID, isDeleted, isResolved, isArchived, isReported, isPinned, parentID, limit, offset)
	if err != nil {
		return nil, err
	}

	totalCount, err := u.r.CountWithFilters(ctx, sqlc.EntityType(entityType), entityID, isDeleted, isResolved, isArchived, isReported)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}

	return &CommentListResponse{
		Comments: responses,
		Meta:     queryparam.NewPaginationMeta(1, limit, totalCount),
	}, nil
}

// GetByIDs implements [UseCase].
func (u *usecase) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]CommentResponse, error) {
	comments, err := u.r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	responses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp, err := u.buildResponse(ctx, &c)
		if err != nil {
			return nil, err
		}
		responses = append(responses, *resp)
	}
	return responses, nil
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

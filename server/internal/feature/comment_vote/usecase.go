package commentvote

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

type VoteInput struct {
	CommentID  uuid.UUID
	EntityType sqlc.EntityType
	UserID     uuid.UUID
	VoteType   sqlc.VoteType
}

type VoteResult struct {
	Vote    *sqlc.CommentVote
	Removed bool
	Message string
}

type UseCase interface {
	Get(ctx context.Context, voteID uuid.UUID) (*sqlc.CommentVote, error)
	GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) ([]sqlc.CommentVote, int64, error)
	Vote(ctx context.Context, input *VoteInput) (*VoteResult, error)
	Unvote(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, userID uuid.UUID) error
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.VoteType]int32, error)
	CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

type usecase struct {
	r Repository
}

func NewUseCase(r Repository) UseCase {
	return &usecase{r: r}
}

func (u *usecase) Get(ctx context.Context, voteID uuid.UUID) (*sqlc.CommentVote, error) {
	vote, err := u.r.Get(ctx, voteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return vote, nil
}

func (u *usecase) GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) ([]sqlc.CommentVote, int64, error) {
	votes, err := u.r.GetByCommentID(ctx, commentID, entityType, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var voteList []sqlc.CommentVote
	if votes != nil {
		voteList = *votes
	} else {
		voteList = []sqlc.CommentVote{}
	}

	total, err := u.r.CountTotal(ctx, commentID, entityType)
	if err != nil {
		return nil, 0, err
	}

	return voteList, total, nil
}

func (u *usecase) Vote(ctx context.Context, input *VoteInput) (*VoteResult, error) {
	existingVotes, err := u.r.GetByCommentID(ctx, input.CommentID, input.EntityType, 100, 0)
	if err != nil {
		return nil, err
	}

	if existingVotes != nil {
		for _, v := range *existingVotes {
			if v.UserID == input.UserID {
				if v.VoteType == input.VoteType {
					err = u.r.Delete(ctx, &v)
					if err != nil {
						return nil, err
					}
					return &VoteResult{Removed: true, Message: "vote removed"}, nil
				}
				now := time.Now()
				updatedVote := &sqlc.CommentVote{
					ID:         v.ID,
					CommentID:  input.CommentID,
					EntityType: input.EntityType,
					UserID:     input.UserID,
					VoteType:   input.VoteType,
					CreatedAt:  v.CreatedAt,
					UpdatedAt:  &now,
				}
				result, err := u.r.Upsert(ctx, updatedVote)
				if err != nil {
					return nil, err
				}
				return &VoteResult{Vote: result}, nil
			}
		}
	}

	now := time.Now()
	vote := &sqlc.CommentVote{
		CommentID:  input.CommentID,
		EntityType: input.EntityType,
		UserID:     input.UserID,
		VoteType:   input.VoteType,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}

	result, err := u.r.Upsert(ctx, vote)
	if err != nil {
		return nil, err
	}

	return &VoteResult{Vote: result}, nil
}

func (u *usecase) Unvote(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, userID uuid.UUID) error {
	votes, err := u.r.GetByCommentID(ctx, commentID, entityType, 100, 0)
	if err != nil {
		return err
	}

	if votes != nil {
		for _, v := range *votes {
			if v.UserID == userID {
				return u.r.Delete(ctx, &v)
			}
		}
	}

	return nil
}

func (u *usecase) CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.VoteType]int32, error) {
	return u.r.CountByCommentID(ctx, commentID, entityType)
}

func (u *usecase) CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	return u.r.CountTotal(ctx, commentID, entityType)
}

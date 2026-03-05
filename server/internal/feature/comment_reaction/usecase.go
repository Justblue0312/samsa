package commentreaction

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

var (
	ErrNotFound         = errors.New("reaction not found")
	ErrReactionNotFound = errors.New("reaction not found for user")
)

type UseCase interface {
	Get(ctx context.Context, id uuid.UUID) (*sqlc.CommentReaction, error)
	GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) (*[]sqlc.CommentReaction, int64, error)
	React(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, entityType sqlc.EntityType, reactionType sqlc.ReactionType) (*sqlc.CommentReaction, error)
	Unreact(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, entityType sqlc.EntityType) error
	CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.ReactionType]int32, error)
	CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error)
}

type usecase struct {
	r Repository
}

func NewUseCase(r Repository) UseCase {
	return &usecase{r: r}
}

func (u *usecase) Get(ctx context.Context, id uuid.UUID) (*sqlc.CommentReaction, error) {
	reaction, err := u.r.Get(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return reaction, nil
}

func (u *usecase) GetByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType, limit, offset int32) (*[]sqlc.CommentReaction, int64, error) {
	reactions, err := u.r.GetByCommentID(ctx, commentID, entityType, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var reactionList []sqlc.CommentReaction
	if reactions != nil {
		reactionList = *reactions
	} else {
		reactionList = []sqlc.CommentReaction{}
	}

	total, err := u.r.CountTotal(ctx, commentID, entityType)
	if err != nil {
		return nil, 0, err
	}

	return &reactionList, total, nil
}

func (u *usecase) React(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, entityType sqlc.EntityType, reactionType sqlc.ReactionType) (*sqlc.CommentReaction, error) {
	existingReactions, err := u.r.GetByCommentID(ctx, commentID, entityType, 100, 0)
	if err != nil {
		return nil, err
	}

	if existingReactions != nil {
		for _, r := range *existingReactions {
			if r.UserID == userID {
				if r.ReactionType == reactionType {
					err = u.r.Delete(ctx, &r)
					if err != nil {
						return nil, err
					}
					return nil, nil
				}
				break
			}
		}
	}

	now := time.Now()
	reaction := &sqlc.CommentReaction{
		CommentID:    commentID,
		EntityType:   entityType,
		UserID:       userID,
		ReactionType: reactionType,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}

	result, err := u.r.Upsert(ctx, reaction)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (u *usecase) Unreact(ctx context.Context, userID uuid.UUID, commentID uuid.UUID, entityType sqlc.EntityType) error {
	reactions, err := u.r.GetByCommentID(ctx, commentID, entityType, 100, 0)
	if err != nil {
		return err
	}

	if reactions != nil {
		for _, r := range *reactions {
			if r.UserID == userID {
				err = u.r.Delete(ctx, &r)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	return ErrReactionNotFound
}

func (u *usecase) CountByCommentID(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (map[sqlc.ReactionType]int32, error) {
	return u.r.CountByCommentID(ctx, commentID, entityType)
}

func (u *usecase) CountTotal(ctx context.Context, commentID uuid.UUID, entityType sqlc.EntityType) (int64, error) {
	return u.r.CountTotal(ctx, commentID, entityType)
}

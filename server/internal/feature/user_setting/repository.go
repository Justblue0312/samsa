package user_setting

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/justblue/samsa/gen/sqlc"
)

//go:generate mockgen -destination=mocks/mock_repository.go -source=repository.go -package=mocks

type Repository interface {
	GetByUserId(ctx context.Context, userId uuid.UUID) (*[]sqlc.UserSetting, error)
	GetByKey(ctx context.Context, userId uuid.UUID, key string) (*sqlc.UserSetting, error)
	Update(ctx context.Context, userId uuid.UUID, userSetting *sqlc.UserSetting) (*sqlc.UserSetting, error)
	Delete(ctx context.Context, userId uuid.UUID, key string) error
}

type repository struct {
	q  *sqlc.Queries
	db sqlc.DBTX
}

func NewRepository(db sqlc.DBTX) Repository {
	return &repository{
		q:  sqlc.New(db),
		db: db,
	}
}

func (r *repository) Delete(ctx context.Context, userId uuid.UUID, key string) error {
	return r.q.DeleteUserSetting(ctx, sqlc.DeleteUserSettingParams{
		UserID: userId,
		Key:    key,
	})
}

func (r *repository) GetByKey(ctx context.Context, userId uuid.UUID, key string) (*sqlc.UserSetting, error) {
	us, err := r.q.GetUserSettingByKey(ctx, sqlc.GetUserSettingByKeyParams{
		UserID: userId,
		Key:    key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserSettingNotFound
		}
		return nil, err
	}
	return &us, nil
}

func (r *repository) GetByUserId(ctx context.Context, userId uuid.UUID) (*[]sqlc.UserSetting, error) {
	uss, err := r.q.GetUserSettings(ctx, userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserSettingNotFound
		}
		return nil, err
	}
	return &uss, nil
}

func (r *repository) Update(ctx context.Context, userId uuid.UUID, userSetting *sqlc.UserSetting) (*sqlc.UserSetting, error) {
	us, err := r.q.UpdateUserSetting(ctx, sqlc.UpdateUserSettingParams{
		UserID: userId,
		Key:    userSetting.Key,
		Value:  userSetting.Value,
	})
	if err != nil {
		return nil, err
	}
	return &us, nil
}

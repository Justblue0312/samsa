package user_setting

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/justblue/samsa/gen/sqlc"
	"go.uber.org/multierr"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -source=usecase.go -package=mocks

var ErrUserSettingNotFound = errors.New("user setting not found or user is not author")

type UseCase interface {
	Get(ctx context.Context, user *sqlc.User) (*[]sqlc.UserSetting, error)
	Update(ctx context.Context, user *sqlc.User, userSettings UserSettingUpdate) (*[]sqlc.UserSetting, error)
	Reset(ctx context.Context, user *sqlc.User) error
}

type usecase struct {
	userSettingrepo Repository
}

func NewUseCase(repo Repository) UseCase {
	return &usecase{
		userSettingrepo: repo,
	}
}
func (u *usecase) Get(ctx context.Context, user *sqlc.User) (*[]sqlc.UserSetting, error) {
	us, err := u.userSettingrepo.GetByUserId(ctx, user.ID)
	if err != nil {
		if errors.Is(err, ErrUserSettingNotFound) {
			return nil, ErrUserSettingNotFound
		}
		return nil, err
	}
	if len(*us) == 0 && !user.IsAuthor {
		return nil, ErrUserSettingNotFound
	}

	return us, nil
}

func (u *usecase) Reset(ctx context.Context, user *sqlc.User) error {
	var result error

	settings := []struct {
		key   string
		value any
	}{
		{UserPreferenceSettingKey, DefaultUserPreferenceSettingSchema},
		{UserNotificationSettingKey, DefaultUserNotificationSettingSchema},
		{UserEditorSettingKey, DefaultUserEditorSettingSchema},
	}

	for _, s := range settings {
		b, err := json.Marshal(s.value)
		if err != nil {
			return err
		}

		_, err = u.userSettingrepo.Update(ctx, user.ID, &sqlc.UserSetting{
			UserID: user.ID,
			Key:    s.key,
			Value:  b,
		})

		result = multierr.Append(result, err)
	}

	return result
}

func (u *usecase) Update(ctx context.Context, user *sqlc.User, userSettings UserSettingUpdate) (*[]sqlc.UserSetting, error) {
	var updatedSettings []sqlc.UserSetting
	var result error

	settings := []struct {
		key   string
		value any
	}{
		{UserPreferenceSettingKey, userSettings.Preference},
		{UserNotificationSettingKey, userSettings.Notification},
		{UserEditorSettingKey, userSettings.Editor},
	}

	for _, s := range settings {
		b, err := json.Marshal(s.value)
		if err != nil {
			return nil, err
		}

		_, err = u.userSettingrepo.Update(ctx, user.ID, &sqlc.UserSetting{
			UserID: user.ID,
			Key:    s.key,
			Value:  b,
		})

		result = multierr.Append(result, err)
	}

	return &updatedSettings, result
}

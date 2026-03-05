package user_setting

import (
	"encoding/json"

	"github.com/justblue/samsa/gen/sqlc"
	"go.uber.org/multierr"
)

type UserSettingUpdate struct {
	Preference   UserPreferenceSetting   `json:"preference"`
	Editor       UserEditorSetting       `json:"editor"`
	Notification UserNotificationSetting `json:"notification"`
}

type UserSettingResponse struct {
	UserSetting struct {
		Preference   UserPreferenceSetting   `json:"preference"`
		Editor       UserEditorSetting       `json:"editor"`
		Notification UserNotificationSetting `json:"notification"`
	} `json:"user_setting"`
}

func ConvertToUserSettingResponse(settings *[]sqlc.UserSetting) (UserSettingResponse, error) {
	var (
		response UserSettingResponse
		result   error
	)

	unmarshal := func(data []byte, target any) {
		if err := json.Unmarshal(data, target); err != nil {
			result = multierr.Append(result, err)
		}
	}

	for _, setting := range *settings {
		switch setting.Key {
		case UserPreferenceSettingKey:
			unmarshal(setting.Value, &response.UserSetting.Preference)
		case UserEditorSettingKey:
			unmarshal(setting.Value, &response.UserSetting.Editor)
		case UserNotificationSettingKey:
			unmarshal(setting.Value, &response.UserSetting.Notification)
		}
	}

	return response, result
}

type UserPreferenceSetting struct {
	Language       string `json:"language" validate:"required,oneof=en es fr de zh ja vi"`
	Timezone       string `json:"timezone" validate:"required,timezone"`
	Theme          string `json:"theme" validate:"required,oneof=light dark system"`
	MarketingOptIn bool   `json:"marketing_opt_in"`
}

type UserNotificationSetting struct {
	EmailEnabled    bool   `json:"email_enabled"`
	PushEnabled     bool   `json:"push_enabled"`
	DigestFrequency string `json:"digest_frequency" validate:"required,oneof=instant daily weekly"`
}

type UserEditorSetting struct {
	FontSize        int    `json:"font_size" validate:"required,min=10,max=32"`
	FontFamily      string `json:"font_family" validate:"required"`
	TabSize         int    `json:"tab_size" validate:"required,min=2,max=8"`
	AutoSave        bool   `json:"auto_save"`
	AutoSaveDelay   int    `json:"auto_save_delay" validate:"required_if=AutoSave true,min=5,max=300"`
	LineWrapping    bool   `json:"line_wrapping"`
	BackgroundColor string `json:"background_color" validate:"required,hexcolor"`
	TextColor       string `json:"text_color" validate:"required,hexcolor"`
}

var (
	UserPreferenceSettingKey   = "preference"
	UserEditorSettingKey       = "editor"
	UserNotificationSettingKey = "notification"

	DefaultUserPreferenceSettingSchema = &UserPreferenceSetting{
		Language:       "en",
		Timezone:       "UTC",
		Theme:          "light",
		MarketingOptIn: false,
	}
	DefaultUserNotificationSettingSchema = &UserNotificationSetting{
		EmailEnabled:    false,
		PushEnabled:     false,
		DigestFrequency: "daily",
	}
	DefaultUserEditorSettingSchema = &UserEditorSetting{
		FontSize:        14,
		FontFamily:      "monospace",
		TabSize:         4,
		AutoSave:        false,
		AutoSaveDelay:   10,
		LineWrapping:    true,
		BackgroundColor: "#ffffff",
		TextColor:       "#000000",
	}
)

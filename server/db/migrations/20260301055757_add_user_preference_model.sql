-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_notification_preference (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    notification_type TEXT NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_notification_preference_user_type_key UNIQUE (user_id, notification_type)
);

CREATE INDEX idx_user_notification_preference_user_id ON user_notification_preference(user_id);

CREATE TRIGGER update_user_notification_preference_updated_at
    BEFORE UPDATE ON user_notification_preference
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_user_notification_preference_updated_at ON user_notification_preference;

DROP TABLE user_notification_preference;
-- +goose StatementEnd

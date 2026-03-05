-- +goose Up
-- +goose StatementBegin

CREATE TYPE submission_type AS ENUM (
    'author_request',
    'story_approval',
    'chapter_approval',
    'other'
);

ALTER TABLE submission
    ALTER COLUMN type TYPE submission_type USING type::submission_type;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE submission
    ALTER COLUMN type TYPE TEXT USING type::text;

DROP TYPE submission_type;

-- +goose StatementEnd

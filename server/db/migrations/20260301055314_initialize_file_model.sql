-- +goose Up
-- +goose StatementBegin
CREATE TYPE file_upload_source AS ENUM ('url', 'file', 'presigned', 'base64');

CREATE TABLE file (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- File Information
    name CHAR(255) NOT NULL,
    path TEXT NOT NULL,
    mime_type CHAR(100),
    size BIGINT NOT NULL DEFAULT 0,
    reference TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '{}',

    -- Storage
    service CHAR(50),

    -- Source
    source file_upload_source NOT NULL DEFAULT 'file',

    -- Status
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_file_owner_id ON file (owner_id);
CREATE INDEX idx_file_path ON file (path);
CREATE INDEX idx_file_service ON file (service);
CREATE INDEX idx_file_source ON file (source);

COMMENT ON TABLE file IS 'File metadata and storage tracking';

CREATE TRIGGER update_file_updated_at
BEFORE UPDATE ON file
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_file_updated_at ON file;

DROP TABLE file;

DROP TYPE file_upload_source;
-- +goose StatementEnd

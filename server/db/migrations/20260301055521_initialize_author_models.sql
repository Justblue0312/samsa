-- +goose Up
-- +goose StatementBegin
CREATE TABLE submission (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    requester_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    approver_id UUID REFERENCES "user"(id) ON DELETE CASCADE,

    -- Submission Information
    approved_at TIMESTAMP WITH TIME ZONE,
    message TEXT,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,

    context JSONB DEFAULT '{}'::JSONB,

    -- Flags
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pending BOOLEAN NOT NULL DEFAULT TRUE,
    is_approved BOOLEAN NOT NULL DEFAULT FALSE,
    is_timeouted BOOLEAN DEFAULT FALSE,
    timeouted_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_submissions_approved_at ON submission(approved_at) WHERE deleted_at IS NULL;

CREATE TABLE submission_assignment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submission(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES "user"(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submission_assignment_submission_id ON submission_assignment(submission_id);
CREATE INDEX idx_submission_assignment_assigned_to ON submission_assignment(assigned_to);

CREATE TABLE submission_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submission(id) ON DELETE CASCADE,
    changed_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    old_status VARCHAR(50) NOT NULL,
    new_status VARCHAR(50) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submission_status_history_submission_id ON submission_status_history(submission_id);

CREATE TABLE author (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    media_id UUID REFERENCES file(id) ON DELETE SET NULL,

    -- Author Information
    stage_name VARCHAR(255) NOT NULL UNIQUE,
    gender VARCHAR(50) NOT NULL DEFAULT 'other',
    slug VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    dob DATE,
    phone VARCHAR(15),
    bio VARCHAR(188),
    description TEXT,
    accepted_terms_of_service BOOLEAN DEFAULT FALSE,
    email_newsletters_and_changelogs BOOLEAN DEFAULT FALSE,
    email_promotions_and_events BOOLEAN DEFAULT FALSE,

    -- Author Flags
    is_recommended BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,

    -- Author Metadata
    stats JSONB DEFAULT '{}'::JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_authors_stage_name ON author(stage_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_authors_slug ON author(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_authors_user_id ON author(user_id) WHERE deleted_at IS NULL;

-- Triggers
CREATE TRIGGER update_submissions_updated_at
BEFORE UPDATE ON submission
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_authors_updated_at
BEFORE UPDATE ON author
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_submissions_updated_at ON submission;
DROP TRIGGER update_authors_updated_at ON author;

DROP TABLE IF EXISTS submission_status_history;
DROP TABLE IF EXISTS submission_assignment;
DROP TABLE IF EXISTS author;
DROP TABLE IF EXISTS submission;
-- +goose StatementEnd

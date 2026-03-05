-- +goose Up
-- +goose StatementBegin

-- Document status enum for approval workflow
CREATE TYPE document_status AS ENUM (
    'draft',
    'pending_review',
    'is_reviewed',
    'is_approved',
    'rejected',
    'archived',
    'deleted'
);

-- Add approval workflow columns to existing document table
ALTER TABLE document ADD COLUMN IF NOT EXISTS status document_status NOT NULL DEFAULT 'draft';
ALTER TABLE document ADD COLUMN IF NOT EXISTS title CHAR(500);
ALTER TABLE document ADD COLUMN IF NOT EXISTS slug CHAR(255);
ALTER TABLE document ADD COLUMN IF NOT EXISTS summary TEXT;
ALTER TABLE document ADD COLUMN IF NOT EXISTS document_type CHAR(100) DEFAULT 'general';
ALTER TABLE document ADD COLUMN IF NOT EXISTS is_locked BOOLEAN DEFAULT FALSE;
ALTER TABLE document ADD COLUMN IF NOT EXISTS is_template BOOLEAN DEFAULT FALSE;
ALTER TABLE document ADD COLUMN IF NOT EXISTS previous_version_id UUID REFERENCES document(id) ON DELETE SET NULL;
ALTER TABLE document ADD COLUMN IF NOT EXISTS total_words INTEGER DEFAULT 0;
ALTER TABLE document ADD COLUMN IF NOT EXISTS total_views INTEGER DEFAULT 0;
ALTER TABLE document ADD COLUMN IF NOT EXISTS total_downloads INTEGER DEFAULT 0;
ALTER TABLE document ADD COLUMN IF NOT EXISTS total_shares INTEGER DEFAULT 0;

-- Add indexes for new columns
CREATE INDEX IF NOT EXISTS idx_documents_status ON document(status);
CREATE INDEX IF NOT EXISTS idx_documents_slug ON document(slug) WHERE slug IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_title ON document(title) WHERE title IS NOT NULL;

-- Document status history for audit trail
CREATE TABLE IF NOT EXISTS document_status_history (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    set_status_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Status Information
    content CHAR(255) NOT NULL,
    status document_status NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for document_status_history
CREATE INDEX IF NOT EXISTS idx_document_status_history_document_id ON document_status_history(document_id);
CREATE INDEX IF NOT EXISTS idx_document_status_history_set_status_by ON document_status_history(set_status_by);
CREATE INDEX IF NOT EXISTS idx_document_status_history_status ON document_status_history(status);

-- Document chapter table (links chapters to documents)
-- This allows chapters (from story module) to be associated with approval documents
CREATE TABLE IF NOT EXISTS document_chapter (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    chapter_id UUID NOT NULL REFERENCES chapter(id) ON DELETE CASCADE,

    -- Ordering within document
    sort_order INTEGER DEFAULT 0,

    -- Metadata
    added_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    notes TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT document_chapters_document_id_chapter_id_key UNIQUE (document_id, chapter_id),
    CONSTRAINT document_chapters_document_id_order_key UNIQUE (document_id, sort_order)
);

-- Indexes for document_chapter
CREATE INDEX IF NOT EXISTS idx_document_chapters_document_id ON document_chapter(document_id);
CREATE INDEX IF NOT EXISTS idx_document_chapters_chapter_id ON document_chapter(chapter_id);

-- Document view tracking
CREATE TABLE IF NOT EXISTS document_view (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- View Information
    viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    view_duration INTEGER, -- in seconds
    completion_percentage INTEGER DEFAULT 0 CHECK (completion_percentage >= 0 AND completion_percentage <= 100),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT document_views_document_id_user_id_key UNIQUE (document_id, user_id)
);

-- Indexes for document_view
CREATE INDEX IF NOT EXISTS idx_document_views_document_id ON document_view(document_id);
CREATE INDEX IF NOT EXISTS idx_document_views_user_id ON document_view(user_id);

-- Document share tracking
CREATE TABLE IF NOT EXISTS document_share (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    shared_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Share Information
    shared_with_email TEXT,
    shared_with_user_id UUID REFERENCES "user"(id) ON DELETE CASCADE,
    share_token TEXT UNIQUE,
    share_message TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Permissions
    can_view BOOLEAN DEFAULT TRUE,
    can_download BOOLEAN DEFAULT FALSE,
    can_comment BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT document_shares_document_id_shared_by_key UNIQUE (document_id, shared_by, shared_with_email)
);

-- Indexes for document_share
CREATE INDEX IF NOT EXISTS idx_document_shares_document_id ON document_share(document_id);
CREATE INDEX IF NOT EXISTS idx_document_shares_shared_by ON document_share(shared_by);
CREATE INDEX IF NOT EXISTS idx_document_shares_share_token ON document_share(share_token) WHERE share_token IS NOT NULL;

-- Document comments for review process
CREATE TABLE IF NOT EXISTS document_comment (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES document_comment(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Comment Information
    content TEXT NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Position (for inline comments)
    line_number INTEGER,
    character_offset INTEGER,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for document_comment
CREATE INDEX IF NOT EXISTS idx_document_comments_document_id ON document_comment(document_id);
CREATE INDEX IF NOT EXISTS idx_document_comments_author_id ON document_comment(author_id);
CREATE INDEX IF NOT EXISTS idx_document_comments_parent_comment_id ON document_comment(parent_comment_id) WHERE parent_comment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_comments_resolved ON document_comment(document_id, resolved) WHERE deleted_at IS NULL;

-- Triggers
CREATE TRIGGER update_document_status_history_updated_at
    BEFORE UPDATE ON document_status_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_document_chapters_updated_at
    BEFORE UPDATE ON document_chapter
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_document_comments_updated_at
    BEFORE UPDATE ON document_comment
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS update_document_comments_updated_at ON document_comment;
DROP TRIGGER IF EXISTS update_document_chapters_updated_at ON document_chapter;
DROP TRIGGER IF EXISTS update_document_status_history_updated_at ON document_status_history;

DROP TABLE IF EXISTS document_comment;
DROP TABLE IF EXISTS document_share;
DROP TABLE IF EXISTS document_view;
DROP TABLE IF EXISTS document_chapter;
DROP TABLE IF EXISTS document_status_history;

-- Remove added columns from document table
ALTER TABLE document DROP COLUMN IF EXISTS status;
ALTER TABLE document DROP COLUMN IF EXISTS title;
ALTER TABLE document DROP COLUMN IF EXISTS slug;
ALTER TABLE document DROP COLUMN IF EXISTS summary;
ALTER TABLE document DROP COLUMN IF EXISTS document_type;
ALTER TABLE document DROP COLUMN IF EXISTS is_locked;
ALTER TABLE document DROP COLUMN IF EXISTS is_template;
ALTER TABLE document DROP COLUMN IF EXISTS previous_version_id;
ALTER TABLE document DROP COLUMN IF EXISTS total_words;
ALTER TABLE document DROP COLUMN IF EXISTS total_views;
ALTER TABLE document DROP COLUMN IF EXISTS total_downloads;
ALTER TABLE document DROP COLUMN IF EXISTS total_shares;

DROP TYPE IF EXISTS document_status;

-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
CREATE TYPE entity_type AS ENUM ('story', 'chapter' ,'comment', 'submission');

CREATE TABLE tag (
    -- Primary Key
    id UUID,

    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Tag Information
    name TEXT NOT NULL,
    description TEXT,
    color CHAR(7) NOT NULL,

    entity_type entity_type NOT NULL,
    entity_id UUID NOT NULL,

    is_hidden BOOLEAN DEFAULT FALSE,
    is_system BOOLEAN DEFAULT FALSE,
    is_recommended BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT tags_name_unique UNIQUE (entity_type, name, color),
    PRIMARY KEY (entity_type, id)

) PARTITION BY LIST (entity_type);

-- Indexes
CREATE INDEX tags_name_trgm_idx ON tag USING gin(name gin_trgm_ops);

-- Create Partitions with Local Indexes
CREATE TABLE tag_story
PARTITION OF tag
FOR VALUES IN ('story');

CREATE TABLE tag_chapter
PARTITION OF tag
FOR VALUES IN ('chapter');

CREATE TABLE tag_submission
PARTITION OF tag
FOR VALUES IN ('submission');

-- Add local indexes and constraints on partitions
-- tag_story partition
ALTER TABLE tag_story ADD CONSTRAINT tag_story_id_unique
    UNIQUE (id);
CREATE INDEX ix_tag_story_entity_id
ON tag_story(entity_id, created_at);

-- tag_chapter partition
ALTER TABLE tag_chapter ADD CONSTRAINT tag_chapter_id_unique
    UNIQUE (id);
CREATE INDEX ix_tag_chapter_entity_id
ON tag_chapter(entity_id, created_at);

-- tag_submission partition
ALTER TABLE tag_submission ADD CONSTRAINT tag_submission_id_unique
    UNIQUE (id);
CREATE INDEX ix_tag_submission_entity_id
ON tag_submission(entity_id, created_at);


CREATE TABLE document_folder (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES document_folder(id) ON DELETE SET NULL,
    depth INT NOT NULL DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT folder_name_unique_per_parent UNIQUE (story_id, parent_id, name),
    CONSTRAINT folder_depth_check CHECK (depth <= 2)
);

CREATE INDEX document_folders_story_id_idx ON document_folder (story_id) WHERE is_deleted = FALSE;
CREATE INDEX document_folders_parent_id_idx ON document_folder (parent_id) WHERE is_deleted = FALSE;


CREATE TABLE document (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    folder_id UUID REFERENCES document_folder(id) ON DELETE SET NULL,
    parent_document_id UUID REFERENCES document(id) ON DELETE SET NULL,

    -- Document Information
    language CHAR(3) NOT NULL,
    branch_name VARCHAR(100) NOT NULL DEFAULT 'main',
    version_number INT NOT NULL DEFAULT 1,
    content JSONB NOT NULL,

    -- Flags
    is_deleted BOOLEAN DEFAULT FALSE,

    stats JSONB DEFAULT '{}'::JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
ALTER TABLE document ADD CONSTRAINT doc_story_creator_branch_unique UNIQUE (story_id, created_by, branch_name, is_deleted);

CREATE INDEX documents_id_idx ON document (id) WHERE deleted_at IS NULL;
CREATE INDEX documents_story_id_idx ON document (story_id) WHERE deleted_at IS NULL;
CREATE INDEX documents_created_by_idx ON document (created_by) WHERE deleted_at IS NULL;



CREATE TABLE spinnet (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    owner_id UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Template Information
    name CHAR(255) NOT NULL,
    content JSONB,
    category CHAR(100),
    smart_syntax CHAR(100),

    -- Flags
    is_deleted BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);


-- Triggers
CREATE TRIGGER update_tags_updated_at
BEFORE UPDATE ON tag
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_documents_updated_at
BEFORE UPDATE ON document
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_document_folders_updated_at
BEFORE UPDATE ON document_folder
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_spinnets_updated_at
BEFORE UPDATE ON spinnet
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_tags_updated_at ON tag;
DROP TRIGGER update_documents_updated_at ON document;
DROP TRIGGER update_document_folders_updated_at ON document_folder;
DROP TRIGGER update_spinnets_updated_at ON spinnet;

DROP TABLE spinnet;
DROP TABLE document_folder;
DROP TABLE document;
DROP TABLE tag_story;
DROP TABLE tag_chapter;
DROP TABLE tag_submission;
DROP TABLE tag;

DROP TYPE entity_type;
-- +goose StatementEnd

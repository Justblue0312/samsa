-- +goose Up
-- +goose StatementBegin
CREATE TYPE notification_level AS ENUM ('low', 'medium', 'high', 'default');
CREATE TYPE reaction_type AS ENUM ('like', 'love', 'haha', 'wow', 'sad', 'angry', 'support');
CREATE TYPE vote_type AS ENUM ('up', 'down');

-- Create Partitioned Comment Table
CREATE TABLE comment (
    -- Primary Key
    id UUID DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    parent_id UUID, -- No FK constraint - app-level validation

    -- Comment Information
    content JSONB NOT NULL,
    depth INTEGER DEFAULT 0,
    score REAL DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_resolved BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,

    is_reported BOOLEAN DEFAULT FALSE,
    reported_at TIMESTAMP WITH TIME ZONE,
    reported_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    is_pinned BOOLEAN DEFAULT FALSE,
    pinned_at TIMESTAMP WITH TIME ZONE,
    pinned_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    entity_type entity_type NOT NULL,
    entity_id UUID NOT NULL,

    -- Metadata
    source TEXT,
    reply_count INTEGER DEFAULT 0,
    reaction_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Primary key must include partition key
    PRIMARY KEY (entity_type, id)
) PARTITION BY LIST (entity_type);

-- Create Partitions with Local Indexes
CREATE TABLE comment_story
PARTITION OF comment
FOR VALUES IN ('story');

CREATE TABLE comment_chapter
PARTITION OF comment
FOR VALUES IN ('chapter');

CREATE TABLE comment_submission
PARTITION OF comment
FOR VALUES IN ('submission');

-- Add local indexes and constraints on partitions
-- comment_story partition
ALTER TABLE comment_story ADD CONSTRAINT comment_story_id_unique
    UNIQUE (id);
CREATE INDEX ix_comment_story_entity_id
    ON comment_story(entity_id, created_at)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_story_user_id
    ON comment_story(user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_story_parent_id
    ON comment_story(parent_id)
    WHERE deleted_at IS NULL;

-- comment_chapter partition
ALTER TABLE comment_chapter ADD CONSTRAINT comment_chapter_id_unique
    UNIQUE (id);
CREATE INDEX ix_comment_chapter_entity_id
    ON comment_chapter(entity_id, created_at)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_chapter_user_id
    ON comment_chapter(user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_chapter_parent_id
    ON comment_chapter(parent_id)
    WHERE deleted_at IS NULL;

-- comment_submission partition
ALTER TABLE comment_submission ADD CONSTRAINT comment_submission_id_unique
    UNIQUE (id);
CREATE INDEX ix_comment_submission_entity_id
    ON comment_submission(entity_id, created_at)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_submission_user_id
    ON comment_submission(user_id)
    WHERE deleted_at IS NULL;
CREATE INDEX ix_comment_submission_parent_id
    ON comment_submission(parent_id)
    WHERE deleted_at IS NULL;

-- Add check constraint to parent table
ALTER TABLE comment ADD CONSTRAINT chk_parent_not_self
    CHECK (parent_id IS NULL OR parent_id <> id);

-- Create other tables (unchanged)
CREATE TABLE notification (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Notification Information
    title CHAR(255),
    icon CHAR(255),
    action_url CHAR(255),
    level notification_level NOT NULL DEFAULT 'default',
    is_read BOOLEAN DEFAULT FALSE,
    type CHAR(50) NOT NULL,
    body JSONB NOT NULL,

    -- Flags
    is_deleted BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX ix_notifications_user_read
    ON notification(user_id, created_at, is_read)
    WHERE deleted_at IS NULL;

CREATE TABLE notification_recipient (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notification(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(notification_id, user_id)
);

CREATE INDEX ix_notification_recipient_user_id ON notification_recipient(user_id, created_at, is_read);


CREATE TABLE user_follow (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    follower_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    followed_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    followed_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT user_follows_unique UNIQUE(follower_id, followed_id)
);

CREATE INDEX ix_user_follows_follower_id
    ON user_follow(follower_id);
CREATE INDEX ix_user_follows_followed_id
    ON user_follow(followed_id);

-- Comment Reaction Table with entity_type for proper constraints
CREATE TABLE comment_reaction (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Add entity_type for proper unique constraint
    entity_type entity_type NOT NULL,
    comment_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,

    reaction_type reaction_type NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint includes partition key
    UNIQUE (entity_type, comment_id, user_id, reaction_type)
);

CREATE MATERIALIZED VIEW comment_reaction_count_mv AS (
    SELECT
        entity_type,
        comment_id,
        reaction_type,
        COUNT(*) AS total
    FROM comment_reaction
    GROUP BY entity_type, comment_id, reaction_type
);

-- Indexes for comment_reaction
CREATE INDEX ix_comment_reactions_entity_comment
    ON comment_reaction(entity_type, comment_id);
CREATE INDEX ix_comment_reactions_user
    ON comment_reaction(user_id, reaction_type);

CREATE TABLE comment_vote (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Add entity_type for proper unique constraint
    entity_type entity_type NOT NULL,
    comment_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,

    vote_type vote_type NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint includes partition key
    UNIQUE (entity_type, comment_id, user_id, vote_type)
);

-- Indexes for comment_vote
CREATE INDEX ix_comment_votes_entity_comment
    ON comment_vote(entity_type, comment_id);
CREATE INDEX ix_comment_votes_user
    ON comment_vote(user_id, vote_type);

CREATE MATERIALIZED VIEW comment_vote_count_mv AS (
    SELECT
        entity_type,
        comment_id,
        vote_type,
        COUNT(*) AS total
    FROM comment_vote
    GROUP BY entity_type, comment_id, vote_type
);


-- Triggers
CREATE TRIGGER update_comments_updated_at
BEFORE UPDATE ON comment
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notifications_updated_at
BEFORE UPDATE ON notification
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_follows_updated_at
BEFORE UPDATE ON user_follow
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comment_reactions_updated_at
BEFORE UPDATE ON comment_reaction
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comment_votes_updated_at
BEFORE UPDATE ON comment_vote
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop triggers first
DROP MATERIALIZED VIEW comment_reaction_count_mv;
DROP MATERIALIZED VIEW comment_vote_count_mv;

DROP TRIGGER update_comments_updated_at ON comment;
DROP TRIGGER update_notifications_updated_at ON notification;
DROP TRIGGER update_user_follows_updated_at ON user_follow;
DROP TRIGGER update_comment_reactions_updated_at ON comment_reaction;
DROP TRIGGER update_comment_votes_updated_at ON comment_vote;

-- Drop tables
DROP TABLE comment_vote;
DROP TABLE comment_reaction;
DROP TABLE user_follow;
DROP TABLE notification_recipient;
DROP TABLE notification;
DROP TABLE comment_submission;
DROP TABLE comment_chapter;
DROP TABLE comment_story;
DROP TABLE comment;

-- Drop types
DROP TYPE notification_level;
DROP TYPE vote_type;
DROP TYPE reaction_type;
-- +goose StatementEnd

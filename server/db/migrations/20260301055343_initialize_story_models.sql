-- +goose Up
-- +goose StatementBegin
CREATE TYPE story_status AS ENUM ('draft', 'published', 'archived', 'deleted', 'is_reviewed', 'is_approved');
CREATE TYPE flag_types AS ENUM ('spam', 'inappropriate', 'copyright', 'plagiarism', 'harassment', 'hate_speech', 'self_harm', 'explicit', 'privacy', 'misinformation', 'other');
CREATE TYPE flag_rate AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE report_status AS ENUM ('pending', 'resolved', 'rejected', 'archived');

CREATE TABLE genre (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Genre Information
    name TEXT NOT NULL,
    description TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE story (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES file(id) ON DELETE CASCADE,

    -- Story Information
    name CHAR(255) NOT NULL,
    slug CHAR(255) UNIQUE NOT NULL,
    synopsis TEXT,

    -- Flags
    is_deleted BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    is_recommended BOOLEAN DEFAULT FALSE,

    status story_status NOT NULL DEFAULT 'draft',

    first_published_at TIMESTAMP WITH TIME ZONE,
    last_published_at TIMESTAMP WITH TIME ZONE,

    settings JSONB DEFAULT '{}'::JSONB,

    search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(slug, ''))
    ) STORED,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_stories_owner_active
    ON story(owner_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_stories_status_active
    ON story(status)
    WHERE deleted_at IS NULL;

CREATE TABLE story_status_history (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    set_status_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Status Information
    content CHAR(255) NOT NULL,
    status story_status NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_story_status_history_story_id ON story_status_history(story_id);
CREATE INDEX idx_story_status_history_set_status_by ON story_status_history(set_status_by);
CREATE INDEX idx_story_status_history_status ON story_status_history(status);

CREATE TABLE chapter (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,

    -- Chapter Information
    title CHAR(500) NOT NULL,
    number INTEGER,
    sort_order INTEGER DEFAULT 0,
    summary TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    is_published BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMP WITH TIME ZONE,

    total_words INTEGER DEFAULT 0,
    total_views INTEGER DEFAULT 0,
    total_votes INTEGER DEFAULT 0,
    total_favorites INTEGER DEFAULT 0,
    total_bookmarks INTEGER DEFAULT 0,
    total_flags INTEGER DEFAULT 0,
    total_reports INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chapters_story_id_order_key UNIQUE (story_id, sort_order),
    CONSTRAINT chapters_story_id_number_key UNIQUE (story_id, number)
);

-- Indexes
CREATE INDEX idx_chapters_story_active
    ON chapter(story_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_chapters_number_active
    ON chapter(number)
    WHERE deleted_at IS NULL;

CREATE MATERIALIZED VIEW story_stats_mv AS
SELECT
    story_id,
    COUNT(*) AS total_chapters,
    SUM(CASE WHEN is_published THEN 1 ELSE 0 END) AS published_chapters,
    SUM(CASE WHEN NOT is_published THEN 1 ELSE 0 END) AS draft_chapters,
    MAX(published_at) AS last_published_at,
    COALESCE(SUM(total_words), 0) AS total_words,
    COALESCE(SUM(total_views), 0) AS total_views,
    COALESCE(SUM(total_votes), 0) AS total_votes,
    COALESCE(SUM(total_favorites), 0) AS total_favorites,
    COALESCE(SUM(total_bookmarks), 0) AS total_bookmarks,
    COALESCE(SUM(total_flags), 0) AS total_flags,
    COALESCE(SUM(total_reports), 0) AS total_reports
FROM chapter
GROUP BY story_id;

CREATE UNIQUE INDEX story_stats_mv_story_id_idx
    ON story_stats_mv (story_id);

-- REFRESH MATERIALIZED VIEW CONCURRENTLY story_stats_mv;

CREATE TABLE story_vote (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Rating Information
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    CONSTRAINT story_votes_story_id_user_id_key UNIQUE (story_id, user_id)
);

-- Indexes
CREATE INDEX idx_story_votes_story_id ON story_vote(story_id);
CREATE INDEX idx_story_votes_user_id ON story_vote(user_id);

CREATE TABLE user_bookmark (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    CONSTRAINT user_bookmarks_story_id_user_id_key UNIQUE (story_id, user_id)
);

CREATE TABLE user_favorite (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    CONSTRAINT user_favorites_story_id_user_id_key UNIQUE (story_id, user_id)
);

CREATE TABLE flag (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    inspector_id UUID REFERENCES "user"(id) ON DELETE CASCADE,

    -- Flag Information
    title CHAR(255) NOT NULL,
    description TEXT,
    flag_type flag_types NOT NULL,
    flag_rate flag_rate NOT NULL,
    flag_score FLOAT NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX flags_story_id_idx ON flag(story_id);
CREATE INDEX flags_inspector_id_idx ON flag(inspector_id) WHERE inspector_id IS NOT NULL;
CREATE INDEX flags_flag_type_idx ON flag(flag_type);
CREATE INDEX flags_flag_rate_idx ON flag(flag_rate);


CREATE TABLE story_report (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Report Information
    title CHAR(255) NOT NULL,
    description TEXT,

    -- Status
    status report_status DEFAULT 'pending',

    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID REFERENCES "user"(id) ON DELETE CASCADE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX story_reports_story_id_idx ON story_report(story_id);
CREATE INDEX story_reports_reporter_id_idx ON story_report(reporter_id);
CREATE INDEX story_reports_chapter_id_idx ON story_report(chapter_id) WHERE chapter_id IS NOT NULL;
CREATE INDEX story_reports_status_idx ON story_report(status);

-- Triggers
CREATE TRIGGER update_genres_updated_at
BEFORE UPDATE ON genre
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_stories_updated_at
BEFORE UPDATE ON story
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_story_status_history_updated_at
BEFORE UPDATE ON story_status_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chapters_updated_at
BEFORE UPDATE ON chapter
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_story_votes_updated_at
BEFORE UPDATE ON story_vote
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_bookmarks_updated_at
BEFORE UPDATE ON user_bookmark
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_favorites_updated_at
BEFORE UPDATE ON user_favorite
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_flags_updated_at
BEFORE UPDATE ON flag
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_story_reports_updated_at
BEFORE UPDATE ON story_report
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW story_stats_mv;

DROP TRIGGER update_story_reports_updated_at ON story_report;
DROP TRIGGER update_genres_updated_at ON genre;
DROP TRIGGER update_stories_updated_at ON story;
DROP TRIGGER update_story_status_history_updated_at ON story_status_history;
DROP TRIGGER update_chapters_updated_at ON chapter;
DROP TRIGGER update_story_votes_updated_at ON story_vote;
DROP TRIGGER update_user_bookmarks_updated_at ON user_bookmark;
DROP TRIGGER update_user_favorites_updated_at ON user_favorite;
DROP TRIGGER update_flags_updated_at ON flag;

DROP TABLE flag;
DROP TABLE user_favorite;
DROP TABLE user_bookmark;
DROP TABLE story_vote;
DROP TABLE story_status_history;
DROP TABLE story_report;
DROP TABLE chapter;
DROP TABLE story;
DROP TABLE genre;

DROP TYPE report_status;
DROP TYPE flag_rate;
DROP TYPE flag_types;
DROP TYPE story_status;
-- +goose StatementEnd

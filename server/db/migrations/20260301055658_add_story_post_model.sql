-- +goose Up
-- +goose StatementBegin
CREATE TABLE story_post (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    media_ids UUID[] DEFAULT '{}',
    story_id UUID REFERENCES story(id) ON DELETE SET NULL,
    chapter_id UUID REFERENCES chapter(id) ON DELETE SET NULL,
    is_notify_followers BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_story_post_author_id ON story_post(author_id);
CREATE INDEX idx_story_post_story_id ON story_post(story_id);
CREATE INDEX idx_story_post_chapter_id ON story_post(chapter_id);

CREATE TRIGGER update_story_post_updated_at
    BEFORE UPDATE ON story_post
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_story_post_updated_at ON story_post;

DROP TABLE story_post;
-- +goose StatementEnd

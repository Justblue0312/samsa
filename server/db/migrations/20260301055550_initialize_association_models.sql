-- +goose Up
-- +goose StatementBegin
CREATE TABLE story_genre (
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    genre_id UUID NOT NULL REFERENCES genre(id) ON DELETE CASCADE,
    PRIMARY KEY (story_id, genre_id),
    UNIQUE(story_id, genre_id)
);

CREATE TABLE chapter_document (
    chapter_id UUID NOT NULL REFERENCES chapter(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,
    PRIMARY KEY (chapter_id, document_id),
    UNIQUE(chapter_id, document_id)
);

CREATE TABLE story_flag (
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    flag_id UUID NOT NULL REFERENCES flag(id) ON DELETE CASCADE,
    PRIMARY KEY (story_id, flag_id),
    UNIQUE(story_id, flag_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE story_flag;
DROP TABLE chapter_document;
DROP TABLE story_genre;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin

-- 1. Create submission_status enum
CREATE TYPE submission_status AS ENUM (
    'pending',
    'claimed',
    'assigned',
    'approved',
    'rejected',
    'timeouted',
    'archived'
);

-- 2. Auto-increment sequence for human-readable expose_id
CREATE SEQUENCE submission_expose_id_seq START 1;

-- 3. Add new columns to submission
ALTER TABLE submission
    ADD COLUMN status       submission_status NOT NULL DEFAULT 'pending',
    ADD COLUMN expose_id    TEXT              NOT NULL DEFAULT 'SUB-' || LPAD(nextval('submission_expose_id_seq')::text, 4, '0');

-- 4. Add FTS search_vector (message is nullable, coalesce to empty)
ALTER TABLE submission
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        to_tsvector('english', title) ||
        COALESCE(to_tsvector('english', message), ''::tsvector)
    ) STORED;

-- 5. Backfill status from existing boolean flag columns
UPDATE submission
SET status = CASE
    WHEN is_timeouted = TRUE                          THEN 'timeouted'::submission_status
    WHEN is_approved  = TRUE                          THEN 'approved'::submission_status
    WHEN is_pending   = FALSE AND is_approved = FALSE THEN 'rejected'::submission_status
    ELSE                                                   'pending'::submission_status
END;

-- 6. Drop obsolete boolean flag columns
ALTER TABLE submission
    DROP COLUMN is_pending,
    DROP COLUMN is_approved,
    DROP COLUMN is_timeouted,
    DROP COLUMN timeouted_at;

-- 7. Update submission_status_history to use the enum
ALTER TABLE submission_status_history
    ALTER COLUMN old_status TYPE submission_status USING old_status::submission_status,
    ALTER COLUMN new_status TYPE submission_status USING new_status::submission_status;

-- 8. Indexes
CREATE UNIQUE INDEX idx_submission_expose_id    ON submission(expose_id);
CREATE INDEX        idx_submission_status       ON submission(status) WHERE deleted_at IS NULL;
CREATE INDEX        idx_submission_search       ON submission USING GIN(search_vector);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore boolean flag columns
ALTER TABLE submission
    ADD COLUMN is_pending    BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN is_approved   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_timeouted  BOOLEAN DEFAULT FALSE,
    ADD COLUMN timeouted_at  TIMESTAMP WITH TIME ZONE;

-- Backfill boolean flags from status
UPDATE submission
SET
    is_pending   = (status = 'pending' OR status = 'claimed' OR status = 'assigned'),
    is_approved  = (status = 'approved'),
    is_timeouted = (status = 'timeouted');

-- Revert submission_status_history columns to varchar
ALTER TABLE submission_status_history
    ALTER COLUMN old_status TYPE VARCHAR(50) USING old_status::text,
    ALTER COLUMN new_status TYPE VARCHAR(50) USING new_status::text;

-- Drop new submission columns
DROP INDEX IF EXISTS idx_submission_expose_id;
DROP INDEX IF EXISTS idx_submission_status;
DROP INDEX IF EXISTS idx_submission_search;

ALTER TABLE submission
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS expose_id,
    DROP COLUMN IF EXISTS search_vector;

-- Drop sequence and enum
DROP SEQUENCE IF EXISTS submission_expose_id_seq;
DROP TYPE IF EXISTS submission_status;

-- +goose StatementEnd

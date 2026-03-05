-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gist";
CREATE EXTENSION IF NOT EXISTS "unaccent";

CREATE TYPE rate_limit_group AS ENUM ('default', 'web', 'elevated', 'restricted');
CREATE TYPE oauth_provider AS ENUM ('google', 'github');

CREATE TABLE "user" (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Profile Information
    email CHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash TEXT NOT NULL,

    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_staff BOOLEAN NOT NULL DEFAULT FALSE,
    is_author BOOLEAN NOT NULL DEFAULT FALSE,
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    banned_at TIMESTAMP WITH TIME ZONE,
    ban_reason TEXT,

    -- Activity Tracking
    last_login_at TIMESTAMP WITH TIME ZONE,

    -- Rate Limiting
    rate_limit_group rate_limit_group NOT NULL DEFAULT 'default',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT users_email_format CHECK (trim(email) ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_ban_logic CHECK (
        (is_banned = FALSE AND banned_at IS NULL AND ban_reason IS NULL) OR
        (is_banned = TRUE AND banned_at IS NOT NULL)
    )
);
CREATE INDEX idx_user_created_at ON "user" (created_at DESC);
CREATE INDEX idx_user_last_login_at ON "user" (last_login_at DESC) WHERE last_login_at IS NOT NULL;

CREATE TABLE user_setting (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value JSONB NOT NULL DEFAULT '{}'::JSONB,
    UNIQUE(user_id, key)
);

CREATE TABLE session (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    token TEXT NOT NULL UNIQUE,

    ip_address TEXT,
    user_agent TEXT,
    device_info TEXT,
    scopes TEXT[] DEFAULT ARRAY[]::text[],
    metadata JSONB DEFAULT '{}'::jsonb,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_user_token UNIQUE (user_id, token)
);
CREATE INDEX idx_session_user_id ON session(user_id);
CREATE INDEX idx_session_token ON session(token);
CREATE INDEX idx_session_expires_at ON session(expires_at);


CREATE TABLE oauth_account (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- User Reference
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- OAuth Provider
    provider oauth_provider NOT NULL,

    -- Token Management
    access_token TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    refresh_token TEXT,
    refresh_token_expires_at TIMESTAMP WITH TIME ZONE,

    -- Provider Account Information
    account_id CHAR(255) NOT NULL, -- Provider's user ID
    account_email CHAR(255),
    account_username CHAR(255),
    account_avatar_url CHAR(255),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT oauth_accounts_user_provider_unique UNIQUE (user_id, provider)
);
CREATE INDEX idx_oauth_account_user_id_idx ON oauth_account (user_id);
CREATE INDEX idx_oauth_account_provider_idx ON oauth_account (provider);
CREATE INDEX idx_oauth_account_account_id_idx ON oauth_account (provider, account_id);
CREATE INDEX idx_oauth_account_expires_at_idx ON oauth_account (expires_at) WHERE expires_at IS NOT NULL;


CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_updated_at
    BEFORE UPDATE ON "user"
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at
BEFORE UPDATE ON session
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_oauth_accounts_updated_at
    BEFORE UPDATE ON oauth_account
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_user_updated_at ON "user";
DROP TRIGGER update_sessions_updated_at ON session;
DROP TRIGGER update_oauth_accounts_updated_at ON oauth_account;

DROP TABLE session;
DROP TABLE oauth_account;
DROP TABLE user_setting;
DROP TABLE "user";

DROP TYPE rate_limit_group;
DROP TYPE oauth_provider;
-- +goose StatementEnd

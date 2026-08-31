-- App users have no login email or password.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'human';

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_kind_check;
ALTER TABLE users
    ADD CONSTRAINT users_kind_check CHECK (kind IN ('human', 'app'));

ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

ALTER TABLE workspace_members
    DROP CONSTRAINT IF EXISTS workspace_members_kind_check;
ALTER TABLE workspace_members
    ADD CONSTRAINT workspace_members_kind_check CHECK (kind IN ('member', 'external', 'app'));

CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL CHECK (origin IN ('first_party', 'custom')),
    owner_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspace_app_installs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    app_id UUID NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    installed_by UUID REFERENCES users (id) ON DELETE SET NULL,
    bot_user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, app_id)
);

CREATE INDEX workspace_app_installs_workspace_idx
    ON workspace_app_installs (workspace_id);

CREATE TABLE channel_app_installs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    workspace_app_install_id UUID NOT NULL REFERENCES workspace_app_installs (id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_app_install_id, channel_id)
);

CREATE INDEX channel_app_installs_channel_idx
    ON channel_app_installs (channel_id);

CREATE TABLE incoming_webhooks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    channel_app_install_id UUID NOT NULL UNIQUE REFERENCES channel_app_installs (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS payload JSONB;

INSERT INTO apps (slug, name, description, origin, capabilities)
VALUES (
    'incoming-webhooks',
    'Incoming Webhooks',
    'Post rich messages into a channel from any service that can make an HTTP request.',
    'first_party',
    '["incoming_webhook"]'::jsonb
);

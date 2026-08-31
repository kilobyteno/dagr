DELETE FROM incoming_webhooks;
DELETE FROM channel_app_installs;
DELETE FROM workspace_app_installs;
DELETE FROM apps WHERE slug = 'incoming-webhooks';

ALTER TABLE messages DROP COLUMN IF EXISTS payload;

DROP TABLE IF EXISTS incoming_webhooks;
DROP TABLE IF EXISTS channel_app_installs;
DROP TABLE IF EXISTS workspace_app_installs;
DROP TABLE IF EXISTS apps;

ALTER TABLE workspace_members
    DROP CONSTRAINT IF EXISTS workspace_members_kind_check;
ALTER TABLE workspace_members
    ADD CONSTRAINT workspace_members_kind_check CHECK (kind IN ('member', 'external'));

ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_kind_check;
ALTER TABLE users DROP COLUMN IF EXISTS kind;

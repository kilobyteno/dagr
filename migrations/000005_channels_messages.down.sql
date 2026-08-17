DROP TABLE IF EXISTS scheduled_messages;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS workspace_invites;
DROP TABLE IF EXISTS channel_members;

ALTER TABLE channels
    DROP COLUMN IF EXISTS updated_by,
    DROP COLUMN IF EXISTS topic;

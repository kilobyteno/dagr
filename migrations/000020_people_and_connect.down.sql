DROP INDEX IF EXISTS federated_message_refs_local_uidx;
DROP TABLE IF EXISTS federated_message_refs;
DROP INDEX IF EXISTS conversation_peers_peer_idx;
DROP TABLE IF EXISTS conversation_peers;
DROP TABLE IF EXISTS federated_peers;

ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_sharing_mode_check;
ALTER TABLE channels DROP COLUMN IF EXISTS sharing_mode;

DROP INDEX IF EXISTS users_remote_identity_uidx;
ALTER TABLE users DROP COLUMN IF EXISTS remote_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS remote_server_id;
ALTER TABLE users DROP COLUMN IF EXISTS is_remote;

ALTER TABLE workspace_members DROP COLUMN IF EXISTS home_workspace_icon_url;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS home_workspace_name;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS home_workspace_remote_id;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS home_server_id;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS home_workspace_id;
ALTER TABLE workspace_members DROP CONSTRAINT IF EXISTS workspace_members_kind_check;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS kind;

-- Membership origin (same-server and federated external people)
ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'member';

ALTER TABLE workspace_members
	DROP CONSTRAINT IF EXISTS workspace_members_kind_check;
ALTER TABLE workspace_members
	ADD CONSTRAINT workspace_members_kind_check CHECK (kind IN ('member', 'external'));

ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS home_workspace_id UUID REFERENCES workspaces (id) ON DELETE SET NULL;
ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS home_server_id TEXT;
ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS home_workspace_remote_id TEXT;
ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS home_workspace_name TEXT;
ALTER TABLE workspace_members
	ADD COLUMN IF NOT EXISTS home_workspace_icon_url TEXT;

-- Shadow users for federated remote people
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS is_remote BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS remote_server_id TEXT;
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS remote_user_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS users_remote_identity_uidx
	ON users (remote_server_id, remote_user_id)
	WHERE is_remote = true;

-- Shared conversations
ALTER TABLE channels
	ADD COLUMN IF NOT EXISTS sharing_mode TEXT NOT NULL DEFAULT 'local';

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_sharing_mode_check;
ALTER TABLE channels
	ADD CONSTRAINT channels_sharing_mode_check CHECK (sharing_mode IN ('local', 'shared'));

CREATE TABLE IF NOT EXISTS federated_peers (
	server_id TEXT PRIMARY KEY,
	public_url TEXT NOT NULL,
	signing_public_key TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'trusted' CHECK (status IN ('trusted', 'revoked')),
	trusted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS conversation_peers (
	channel_id UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
	peer_server_id TEXT NOT NULL REFERENCES federated_peers (server_id) ON DELETE CASCADE,
	peer_channel_id TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (channel_id, peer_server_id)
);

CREATE INDEX IF NOT EXISTS conversation_peers_peer_idx
	ON conversation_peers (peer_server_id, peer_channel_id);

CREATE TABLE IF NOT EXISTS federated_message_refs (
	channel_id UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
	origin_server_id TEXT NOT NULL,
	origin_message_id TEXT NOT NULL,
	local_message_id UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (channel_id, origin_server_id, origin_message_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS federated_message_refs_local_uidx
	ON federated_message_refs (local_message_id);

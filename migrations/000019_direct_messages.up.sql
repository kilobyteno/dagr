ALTER TABLE channels
	ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'channel';

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_kind_check;
ALTER TABLE channels
	ADD CONSTRAINT channels_kind_check CHECK (kind IN ('channel', 'dm'));

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_dm_private_check;
ALTER TABLE channels
	ADD CONSTRAINT channels_dm_private_check CHECK (kind <> 'dm' OR is_private = true);

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_workspace_id_name_key;

DROP INDEX IF EXISTS channels_workspace_name_channel_uidx;
CREATE UNIQUE INDEX channels_workspace_name_channel_uidx
	ON channels (workspace_id, name)
	WHERE kind = 'channel';

CREATE TABLE IF NOT EXISTS dm_pairs (
	channel_id UUID PRIMARY KEY REFERENCES channels (id) ON DELETE CASCADE,
	workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	user_a UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	user_b UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	CHECK (user_a < user_b),
	UNIQUE (workspace_id, user_a, user_b)
);

CREATE INDEX IF NOT EXISTS dm_pairs_workspace_id_idx ON dm_pairs (workspace_id);

DROP TABLE IF EXISTS dm_pairs;

DROP INDEX IF EXISTS channels_workspace_name_channel_uidx;

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_dm_private_check;
ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_kind_check;

DELETE FROM channels WHERE kind = 'dm';

ALTER TABLE channels
	DROP COLUMN IF EXISTS kind;

ALTER TABLE channels
	DROP CONSTRAINT IF EXISTS channels_workspace_id_name_key;
ALTER TABLE channels
	ADD CONSTRAINT channels_workspace_id_name_key UNIQUE (workspace_id, name);

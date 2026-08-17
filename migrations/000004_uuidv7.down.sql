ALTER TABLE workspace_domains ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE channels ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE workspaces ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE sessions ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();

DROP FUNCTION IF EXISTS uuidv7();

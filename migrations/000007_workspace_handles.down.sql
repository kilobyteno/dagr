DROP INDEX IF EXISTS workspace_members_workspace_id_handle_uidx;
ALTER TABLE workspace_members DROP COLUMN IF EXISTS handle;

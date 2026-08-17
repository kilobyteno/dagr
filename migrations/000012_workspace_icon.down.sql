ALTER TABLE workspaces
    DROP COLUMN IF EXISTS icon_updated_at,
    DROP COLUMN IF EXISTS icon_content_type,
    DROP COLUMN IF EXISTS icon_bytes;

ALTER TABLE workspaces
    ADD COLUMN icon_bytes BYTEA,
    ADD COLUMN icon_content_type TEXT,
    ADD COLUMN icon_updated_at TIMESTAMPTZ;

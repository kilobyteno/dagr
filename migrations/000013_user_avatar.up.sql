ALTER TABLE users
    ADD COLUMN avatar_bytes BYTEA,
    ADD COLUMN avatar_content_type TEXT,
    ADD COLUMN avatar_updated_at TIMESTAMPTZ;

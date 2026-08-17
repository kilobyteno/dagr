ALTER TABLE users
    ADD COLUMN notification_level TEXT NOT NULL DEFAULT 'mentions'
        CHECK (notification_level IN ('all', 'mentions', 'nothing'));

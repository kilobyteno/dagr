ALTER TABLE users
    ADD COLUMN locale TEXT NOT NULL DEFAULT 'en-GB'
        CHECK (locale IN ('en-GB', 'nb'));

CREATE TABLE message_link_previews (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    normalized_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'ready', 'failed', 'skipped')),
    title TEXT,
    description TEXT,
    site_name TEXT,
    image_url TEXT,
    error TEXT,
    fetched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (message_id, normalized_url)
);

CREATE INDEX message_link_previews_message_id_idx
    ON message_link_previews (message_id);

CREATE INDEX message_link_previews_pending_idx
    ON message_link_previews (created_at)
    WHERE status = 'pending';

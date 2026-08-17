CREATE TABLE channel_notification_settings (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    level TEXT NOT NULL CHECK (level IN ('all', 'mentions', 'nothing')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX channel_notification_settings_channel_idx
    ON channel_notification_settings (channel_id);

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications
    ADD CONSTRAINT notifications_kind_check
    CHECK (kind IN (
        'mention',
        'message',
        'channel_invite',
        'workspace_invite',
        'workspace_join'
    ));

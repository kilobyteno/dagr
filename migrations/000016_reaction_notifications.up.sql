ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_kind_check;
ALTER TABLE notifications
    ADD CONSTRAINT notifications_kind_check
    CHECK (kind IN (
        'mention',
        'message',
        'reaction',
        'channel_invite',
        'workspace_invite',
        'workspace_join'
    ));

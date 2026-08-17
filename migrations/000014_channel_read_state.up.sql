CREATE TABLE channel_read_state (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    last_read_message_id UUID REFERENCES messages (id) ON DELETE SET NULL,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX channel_read_state_channel_id_idx ON channel_read_state (channel_id);

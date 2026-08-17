CREATE TABLE workspace_member_handle_aliases (
    workspace_id UUID NOT NULL,
    user_id UUID NOT NULL,
    handle CITEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, handle),
    FOREIGN KEY (workspace_id, user_id)
        REFERENCES workspace_members (workspace_id, user_id)
        ON DELETE CASCADE
);

CREATE INDEX workspace_member_handle_aliases_member_idx
    ON workspace_member_handle_aliases (workspace_id, user_id);

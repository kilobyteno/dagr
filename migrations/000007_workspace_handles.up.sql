ALTER TABLE workspace_members
    ADD COLUMN handle CITEXT;

WITH base AS (
    SELECT
        wm.workspace_id,
        wm.user_id,
        COALESCE(
            NULLIF(
                TRIM(BOTH '_' FROM regexp_replace(lower(u.display_name), '[^a-z0-9]+', '_', 'g')),
                ''
            ),
            'member'
        ) AS base_handle
    FROM workspace_members wm
    INNER JOIN users u ON u.id = wm.user_id
),
numbered AS (
    SELECT
        workspace_id,
        user_id,
        base_handle,
        ROW_NUMBER() OVER (
            PARTITION BY workspace_id, base_handle
            ORDER BY user_id
        ) AS rn
    FROM base
)
UPDATE workspace_members wm
SET handle = CASE
    WHEN n.rn = 1 THEN LEFT(n.base_handle, 32)
    ELSE LEFT(n.base_handle, 28) || '_' || n.rn::text
END
FROM numbered n
WHERE wm.workspace_id = n.workspace_id
  AND wm.user_id = n.user_id;

UPDATE workspace_members
SET handle = 'member_' || substr(replace(user_id::text, '-', ''), 1, 8)
WHERE handle IS NULL OR handle = '';

ALTER TABLE workspace_members
    ALTER COLUMN handle SET NOT NULL;

CREATE UNIQUE INDEX workspace_members_workspace_id_handle_uidx
    ON workspace_members (workspace_id, handle);

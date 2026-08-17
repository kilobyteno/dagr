CREATE TABLE workspace_domains (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    domain CITEXT NOT NULL,
    verification_token TEXT NOT NULL,
    verified_at TIMESTAMPTZ,
    auto_join BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, domain)
);

CREATE UNIQUE INDEX workspace_domains_verified_domain_uidx
    ON workspace_domains (domain)
    WHERE verified_at IS NOT NULL;

CREATE INDEX workspace_domains_workspace_id_idx ON workspace_domains (workspace_id);

CREATE INDEX workspace_domains_auto_join_idx
    ON workspace_domains (domain)
    WHERE verified_at IS NOT NULL AND auto_join = true;

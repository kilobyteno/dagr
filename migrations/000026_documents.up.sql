CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    parent_id UUID REFERENCES documents (id) ON DELETE RESTRICT,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    updated_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, slug)
);

CREATE INDEX documents_workspace_parent_idx
    ON documents (workspace_id, parent_id);

CREATE TABLE document_import_sources (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, kind, name)
);

CREATE INDEX document_import_sources_workspace_idx
    ON document_import_sources (workspace_id);

CREATE TABLE document_import_refs (
    document_id UUID PRIMARY KEY REFERENCES documents (id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES document_import_sources (id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    external_path TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

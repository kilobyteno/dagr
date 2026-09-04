ALTER TABLE documents
    ADD COLUMN icon TEXT NOT NULL DEFAULT '';

CREATE TABLE document_revisions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    document_id UUID NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    parent_id UUID,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_id, version)
);

CREATE INDEX document_revisions_document_created_idx
    ON document_revisions (document_id, created_at DESC);

INSERT INTO document_revisions (
    document_id, version, parent_id, slug, title, body, icon, created_by, created_at
)
SELECT id, 1, parent_id, slug, title, body, icon, updated_by, updated_at
FROM documents;

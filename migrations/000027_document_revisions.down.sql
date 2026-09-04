DROP TABLE IF EXISTS document_revisions;

ALTER TABLE documents
    DROP COLUMN IF EXISTS icon;

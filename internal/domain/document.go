package domain

import (
	"context"
	"time"
)

// Document is a markdown page in a workspace wiki tree.
type Document struct {
	ID          string
	WorkspaceID string
	ParentID    string
	Slug        string
	Title       string
	Body        string
	Icon        string
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DocumentRevision is a full snapshot of a page at one save.
type DocumentRevision struct {
	ID            string
	DocumentID    string
	Version       int
	ParentID      string
	Slug          string
	Title         string
	Body          string
	Icon          string
	CreatedBy     string
	CreatedByName string
	CreatedAt     time.Time
}

// ImportSourceKind identifies a future document importer.
const (
	ImportSourceDocmost    = "docmost"
	ImportSourceConfluence = "confluence"
)

// ImportSourceConfig is connection settings for a document importer.
type ImportSourceConfig struct {
	Kind     string
	BaseURL  string
	Token    string
	Username string
	Extra    map[string]string
}

// ImportDraft is one page produced by a document importer.
type ImportDraft struct {
	ExternalID       string
	Title            string
	BodyMarkdown     string
	ParentExternalID string
}

// DocumentImporter copies pages from an external wiki into Dagr documents.
// DocMost and Confluence implementations are not shipped yet.
type DocumentImporter interface {
	Kind() string
	Preview(ctx context.Context, cfg ImportSourceConfig) ([]ImportDraft, error)
	Import(ctx context.Context, cfg ImportSourceConfig) ([]ImportDraft, error)
}

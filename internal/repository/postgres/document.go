package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

var (
	ErrDocumentSlugConflict = errors.New("document slug already exists")
	ErrDocumentHasChildren  = errors.New("document has children")
)

const documentSelectColumns = `
	id, workspace_id, parent_id, slug, title, body, icon,
	created_by, updated_by, created_at, updated_at
`

const documentListColumns = `
	id, workspace_id, parent_id, slug, title, '', icon,
	created_by, updated_by, created_at, updated_at
`

type DocumentRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ParentID    *uuid.UUID
	Slug        string
	Title       string
	Body        string
	Icon        string
	CreatedBy   uuid.UUID
	UpdatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r DocumentRow) ToDomain() domain.Document {
	out := domain.Document{
		ID:          r.ID.String(),
		WorkspaceID: r.WorkspaceID.String(),
		Slug:        r.Slug,
		Title:       r.Title,
		Body:        r.Body,
		Icon:        r.Icon,
		CreatedBy:   r.CreatedBy.String(),
		UpdatedBy:   r.UpdatedBy.String(),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.ParentID != nil {
		out.ParentID = r.ParentID.String()
	}
	return out
}

func scanDocumentFields(row *DocumentRow) []any {
	return []any{
		&row.ID, &row.WorkspaceID, &row.ParentID, &row.Slug, &row.Title, &row.Body, &row.Icon,
		&row.CreatedBy, &row.UpdatedBy, &row.CreatedAt, &row.UpdatedAt,
	}
}

type DocumentRevisionRow struct {
	ID         uuid.UUID
	DocumentID uuid.UUID
	Version    int
	ParentID   *uuid.UUID
	Slug       string
	Title      string
	Body       string
	Icon       string
	CreatedBy  uuid.UUID
	CreatedAt  time.Time
}

func (r DocumentRevisionRow) ToDomain() domain.DocumentRevision {
	out := domain.DocumentRevision{
		ID:         r.ID.String(),
		DocumentID: r.DocumentID.String(),
		Version:    r.Version,
		Slug:       r.Slug,
		Title:      r.Title,
		Body:       r.Body,
		Icon:       r.Icon,
		CreatedBy:  r.CreatedBy.String(),
		CreatedAt:  r.CreatedAt,
	}
	if r.ParentID != nil {
		out.ParentID = r.ParentID.String()
	}
	return out
}

func scanDocumentRevisionFields(row *DocumentRevisionRow) []any {
	return []any{
		&row.ID, &row.DocumentID, &row.Version, &row.ParentID,
		&row.Slug, &row.Title, &row.Body, &row.Icon,
		&row.CreatedBy, &row.CreatedAt,
	}
}

const documentRevisionSelectColumns = `
	id, document_id, version, parent_id, slug, title, body, icon, created_by, created_at
`

const documentRevisionListColumns = `
	id, document_id, version, parent_id, slug, title, '', icon, created_by, created_at
`

func (s *Store) InsertDocument(
	ctx context.Context,
	workspaceID uuid.UUID,
	parentID *uuid.UUID,
	slug, title, body, icon string,
	authorID uuid.UUID,
) (DocumentRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DocumentRow{}, fmt.Errorf("begin document insert: %w", err)
	}
	defer tx.Rollback(ctx)

	var row DocumentRow
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (
			workspace_id, parent_id, slug, title, body, icon, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING `+documentSelectColumns+`
	`, workspaceID, parentID, slug, title, body, icon, authorID).Scan(scanDocumentFields(&row)...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return DocumentRow{}, ErrDocumentSlugConflict
		}
		return DocumentRow{}, fmt.Errorf("insert document: %w", err)
	}
	if err := insertDocumentRevisionTx(ctx, tx, row); err != nil {
		return DocumentRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentRow{}, fmt.Errorf("commit document insert: %w", err)
	}
	return row, nil
}

func (s *Store) GetDocument(ctx context.Context, documentID uuid.UUID) (DocumentRow, error) {
	var row DocumentRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+documentSelectColumns+`
		FROM documents WHERE id = $1
	`, documentID).Scan(scanDocumentFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentRow{}, ErrNotFound
	}
	if err != nil {
		return DocumentRow{}, fmt.Errorf("get document: %w", err)
	}
	return row, nil
}

func (s *Store) ListDocuments(ctx context.Context, workspaceID uuid.UUID) ([]DocumentRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+documentListColumns+`
		FROM documents
		WHERE workspace_id = $1
		ORDER BY title ASC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []DocumentRow
	for rows.Next() {
		var row DocumentRow
		if err := rows.Scan(scanDocumentFields(&row)...); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	if out == nil {
		out = []DocumentRow{}
	}
	return out, nil
}

func (s *Store) SearchDocuments(ctx context.Context, workspaceID uuid.UUID, query string, limit int) ([]DocumentRow, error) {
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.pool.Query(ctx, `
		SELECT `+documentListColumns+`
		FROM documents
		WHERE workspace_id = $1
		  AND (title ILIKE $2 OR slug ILIKE $2)
		ORDER BY
			CASE
				WHEN slug ILIKE $3 THEN 0
				WHEN title ILIKE $3 THEN 1
				ELSE 2
			END,
			title ASC, id ASC
		LIMIT $4
	`, workspaceID, pattern, query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()

	var out []DocumentRow
	for rows.Next() {
		var row DocumentRow
		if err := rows.Scan(scanDocumentFields(&row)...); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	if out == nil {
		out = []DocumentRow{}
	}
	return out, nil
}

func (s *Store) DocumentSlugExists(ctx context.Context, workspaceID uuid.UUID, slug string, exceptID *uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM documents
			WHERE workspace_id = $1 AND slug = $2
			  AND ($3::uuid IS NULL OR id <> $3)
		)
	`, workspaceID, slug, exceptID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("document slug exists: %w", err)
	}
	return exists, nil
}

func (s *Store) CountDocumentChildren(ctx context.Context, documentID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM documents WHERE parent_id = $1
	`, documentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count document children: %w", err)
	}
	return count, nil
}

func (s *Store) UpdateDocument(
	ctx context.Context,
	documentID uuid.UUID,
	parentID *uuid.UUID,
	slug, title, body, icon string,
	updatedBy uuid.UUID,
) (DocumentRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DocumentRow{}, fmt.Errorf("begin document update: %w", err)
	}
	defer tx.Rollback(ctx)

	var row DocumentRow
	err = tx.QueryRow(ctx, `
		UPDATE documents
		SET parent_id = $2,
		    slug = $3,
		    title = $4,
		    body = $5,
		    icon = $6,
		    updated_by = $7,
		    updated_at = now()
		WHERE id = $1
		RETURNING `+documentSelectColumns+`
	`, documentID, parentID, slug, title, body, icon, updatedBy).Scan(scanDocumentFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentRow{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return DocumentRow{}, ErrDocumentSlugConflict
		}
		return DocumentRow{}, fmt.Errorf("update document: %w", err)
	}
	if err := insertDocumentRevisionTx(ctx, tx, row); err != nil {
		return DocumentRow{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentRow{}, fmt.Errorf("commit document update: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteDocument(ctx context.Context, documentID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, documentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrDocumentHasChildren
		}
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListDocumentRevisions(ctx context.Context, documentID uuid.UUID) ([]DocumentRevisionRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+documentRevisionListColumns+`
		FROM document_revisions
		WHERE document_id = $1
		ORDER BY version DESC
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("list document revisions: %w", err)
	}
	defer rows.Close()

	var out []DocumentRevisionRow
	for rows.Next() {
		var row DocumentRevisionRow
		if err := rows.Scan(scanDocumentRevisionFields(&row)...); err != nil {
			return nil, fmt.Errorf("scan document revision: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document revisions: %w", err)
	}
	if out == nil {
		out = []DocumentRevisionRow{}
	}
	return out, nil
}

func (s *Store) GetDocumentRevision(ctx context.Context, revisionID uuid.UUID) (DocumentRevisionRow, error) {
	var row DocumentRevisionRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+documentRevisionSelectColumns+`
		FROM document_revisions
		WHERE id = $1
	`, revisionID).Scan(scanDocumentRevisionFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentRevisionRow{}, ErrNotFound
	}
	if err != nil {
		return DocumentRevisionRow{}, fmt.Errorf("get document revision: %w", err)
	}
	return row, nil
}

func insertDocumentRevisionTx(ctx context.Context, tx pgx.Tx, doc DocumentRow) error {
	var version int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM document_revisions
		WHERE document_id = $1
	`, doc.ID).Scan(&version); err != nil {
		return fmt.Errorf("next document revision: %w", err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO document_revisions (
			document_id, version, parent_id, slug, title, body, icon, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, doc.ID, version, doc.ParentID, doc.Slug, doc.Title, doc.Body, doc.Icon, doc.UpdatedBy)
	if err != nil {
		return fmt.Errorf("insert document revision: %w", err)
	}
	return nil
}

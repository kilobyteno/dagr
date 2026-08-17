package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

type LinkPreviewRow struct {
	ID            uuid.UUID
	MessageID     uuid.UUID
	URL           string
	NormalizedURL string
	Status        string
	Title         *string
	Description   *string
	SiteName      *string
	ImageURL      *string
	Error         *string
	FetchedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r LinkPreviewRow) ToDomain() domain.LinkPreview {
	out := domain.LinkPreview{
		ID:            r.ID.String(),
		MessageID:     r.MessageID.String(),
		URL:           r.URL,
		NormalizedURL: r.NormalizedURL,
		Status:        domain.LinkPreviewStatus(r.Status),
		FetchedAt:     r.FetchedAt,
		CreatedAt:     r.CreatedAt,
	}
	if r.Title != nil {
		out.Title = *r.Title
	}
	if r.Description != nil {
		out.Description = *r.Description
	}
	if r.SiteName != nil {
		out.SiteName = *r.SiteName
	}
	if r.ImageURL != nil {
		out.ImageURL = *r.ImageURL
	}
	return out
}

type InsertLinkPreviewInput struct {
	MessageID     uuid.UUID
	URL           string
	NormalizedURL string
}

func (s *Store) InsertLinkPreview(ctx context.Context, in InsertLinkPreviewInput) (LinkPreviewRow, error) {
	var row LinkPreviewRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO message_link_previews (message_id, url, normalized_url, status)
		VALUES ($1, $2, $3, 'pending')
		ON CONFLICT (message_id, normalized_url) DO UPDATE
		SET updated_at = message_link_previews.updated_at
		RETURNING id, message_id, url, normalized_url, status, title, description,
			site_name, image_url, error, fetched_at, created_at, updated_at
	`, in.MessageID, in.URL, in.NormalizedURL).Scan(
		&row.ID, &row.MessageID, &row.URL, &row.NormalizedURL, &row.Status,
		&row.Title, &row.Description, &row.SiteName, &row.ImageURL, &row.Error,
		&row.FetchedAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return LinkPreviewRow{}, fmt.Errorf("insert link preview: %w", err)
	}
	return row, nil
}

func (s *Store) GetLinkPreview(ctx context.Context, id uuid.UUID) (LinkPreviewRow, error) {
	var row LinkPreviewRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, message_id, url, normalized_url, status, title, description,
			site_name, image_url, error, fetched_at, created_at, updated_at
		FROM message_link_previews
		WHERE id = $1
	`, id).Scan(
		&row.ID, &row.MessageID, &row.URL, &row.NormalizedURL, &row.Status,
		&row.Title, &row.Description, &row.SiteName, &row.ImageURL, &row.Error,
		&row.FetchedAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LinkPreviewRow{}, ErrNotFound
	}
	if err != nil {
		return LinkPreviewRow{}, fmt.Errorf("get link preview: %w", err)
	}
	return row, nil
}

func (s *Store) ListLinkPreviewsForMessages(
	ctx context.Context, messageIDs []uuid.UUID,
) ([]LinkPreviewRow, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, message_id, url, normalized_url, status, title, description,
			site_name, image_url, error, fetched_at, created_at, updated_at
		FROM message_link_previews
		WHERE message_id = ANY($1)
		ORDER BY created_at ASC, id ASC
	`, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("list link previews: %w", err)
	}
	defer rows.Close()

	var out []LinkPreviewRow
	for rows.Next() {
		var row LinkPreviewRow
		if err := rows.Scan(
			&row.ID, &row.MessageID, &row.URL, &row.NormalizedURL, &row.Status,
			&row.Title, &row.Description, &row.SiteName, &row.ImageURL, &row.Error,
			&row.FetchedAt, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan link preview: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListPendingLinkPreviews returns stale pending unfurl rows for reclaim.
func (s *Store) ListPendingLinkPreviews(
	ctx context.Context,
	olderThan time.Duration,
	limit int,
) ([]LinkPreviewRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if olderThan <= 0 {
		olderThan = 5 * time.Second
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	rows, err := s.pool.Query(ctx, `
		SELECT id, message_id, url, normalized_url, status, title, description,
			site_name, image_url, error, fetched_at, created_at, updated_at
		FROM message_link_previews
		WHERE status = 'pending'
		  AND created_at <= $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2
	`, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending link previews: %w", err)
	}
	defer rows.Close()

	var out []LinkPreviewRow
	for rows.Next() {
		var row LinkPreviewRow
		if err := rows.Scan(
			&row.ID, &row.MessageID, &row.URL, &row.NormalizedURL, &row.Status,
			&row.Title, &row.Description, &row.SiteName, &row.ImageURL, &row.Error,
			&row.FetchedAt, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending link preview: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type UpdateLinkPreviewInput struct {
	ID          uuid.UUID
	Status      domain.LinkPreviewStatus
	Title       string
	Description string
	SiteName    string
	ImageURL    string
	Error       string
	URL         string
}

func (s *Store) UpdateLinkPreview(ctx context.Context, in UpdateLinkPreviewInput) error {
	var title, description, siteName, imageURL, errText *string
	if in.Title != "" {
		title = &in.Title
	}
	if in.Description != "" {
		description = &in.Description
	}
	if in.SiteName != "" {
		siteName = &in.SiteName
	}
	if in.ImageURL != "" {
		imageURL = &in.ImageURL
	}
	if in.Error != "" {
		errText = &in.Error
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE message_link_previews
		SET status = $2,
			title = $3,
			description = $4,
			site_name = $5,
			image_url = $6,
			error = $7,
			url = COALESCE(NULLIF($8, ''), url),
			fetched_at = now(),
			updated_at = now()
		WHERE id = $1
	`, in.ID, string(in.Status), title, description, siteName, imageURL, errText, in.URL)
	if err != nil {
		return fmt.Errorf("update link preview: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

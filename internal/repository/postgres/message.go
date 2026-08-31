package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

const messageSelectSQL = `
		m.id, m.channel_id, m.author_id, u.display_name, COALESCE(wm.handle, ''),
			(u.avatar_bytes IS NOT NULL), u.avatar_updated_at, COALESCE(u.kind, 'human'),
			m.body, m.content_type, m.payload, m.created_at, m.updated_at
`

func scanMessageFields(row *MessageRow) []any {
	return []any{
		&row.ID, &row.ChannelID, &row.AuthorID, &row.AuthorName, &row.AuthorHandle,
		&row.AuthorHasAvatar, &row.AuthorAvatarUpdated, &row.AuthorKind,
		&row.Body, &row.ContentType, &row.Payload, &row.CreatedAt, &row.UpdatedAt,
	}
}

type MessageRow struct {
	ID                  uuid.UUID
	ChannelID           uuid.UUID
	AuthorID            uuid.UUID
	AuthorName          string
	AuthorHandle        string
	AuthorHasAvatar     bool
	AuthorAvatarUpdated *time.Time
	AuthorKind          string
	Body                string
	ContentType         string
	Payload             []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (r MessageRow) ToDomain() domain.Message {
	out := domain.Message{
		ID:                  r.ID.String(),
		ChannelID:           r.ChannelID.String(),
		AuthorID:            r.AuthorID.String(),
		AuthorName:          r.AuthorName,
		AuthorHandle:        r.AuthorHandle,
		AuthorHasAvatar:     r.AuthorHasAvatar,
		AuthorAvatarUpdated: r.AuthorAvatarUpdated,
		AuthorKind:          r.AuthorKind,
		Body:                r.Body,
		ContentType:         r.ContentType,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
	if out.AuthorKind == "" {
		out.AuthorKind = domain.UserKindHuman
	}
	if len(r.Payload) > 0 {
		var payload domain.RichPayload
		if err := json.Unmarshal(r.Payload, &payload); err == nil {
			out.Payload = &payload
			if payload.Username != "" {
				out.AuthorName = payload.Username
			}
			out.AuthorIconURL = payload.IconURL
		}
	}
	return out
}

type ScheduledMessageRow struct {
	ID            uuid.UUID
	ChannelID     uuid.UUID
	AuthorID      uuid.UUID
	Body          string
	ContentType   string
	SendAt        time.Time
	Status        string
	SentMessageID *uuid.UUID
	Error         *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r ScheduledMessageRow) ToDomain() domain.ScheduledMessage {
	out := domain.ScheduledMessage{
		ID:          r.ID.String(),
		ChannelID:   r.ChannelID.String(),
		AuthorID:    r.AuthorID.String(),
		Body:        r.Body,
		ContentType: r.ContentType,
		SendAt:      r.SendAt,
		Status:      domain.ScheduledMessageStatus(r.Status),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.SentMessageID != nil {
		s := r.SentMessageID.String()
		out.SentMessageID = &s
	}
	if r.Error != nil {
		out.Error = *r.Error
	}
	return out
}

func (s *Store) InsertMessage(
	ctx context.Context,
	channelID, authorID uuid.UUID,
	body, contentType string,
) (MessageRow, error) {
	var row MessageRow
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO messages (channel_id, author_id, body, content_type)
			VALUES ($1, $2, $3, $4)
			RETURNING id, channel_id, author_id, body, content_type, payload, created_at, updated_at
		)
		SELECT i.id, i.channel_id, i.author_id, u.display_name, COALESCE(wm.handle, ''),
			(u.avatar_bytes IS NOT NULL), u.avatar_updated_at, COALESCE(u.kind, 'human'),
			i.body, i.content_type, i.payload, i.created_at, i.updated_at
		FROM inserted i
		INNER JOIN users u ON u.id = i.author_id
		INNER JOIN channels c ON c.id = i.channel_id
		LEFT JOIN workspace_members wm
			ON wm.workspace_id = c.workspace_id AND wm.user_id = i.author_id
	`, channelID, authorID, body, contentType).Scan(scanMessageFields(&row)...)
	if err != nil {
		return MessageRow{}, fmt.Errorf("insert message: %w", err)
	}
	return row, nil
}

func (s *Store) InsertAppMessage(
	ctx context.Context,
	channelID, authorID uuid.UUID,
	body, contentType string,
	payload []byte,
) (MessageRow, error) {
	var row MessageRow
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO messages (channel_id, author_id, body, content_type, payload)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, channel_id, author_id, body, content_type, payload, created_at, updated_at
		)
		SELECT i.id, i.channel_id, i.author_id, u.display_name, COALESCE(wm.handle, ''),
			(u.avatar_bytes IS NOT NULL), u.avatar_updated_at, COALESCE(u.kind, 'human'),
			i.body, i.content_type, i.payload, i.created_at, i.updated_at
		FROM inserted i
		INNER JOIN users u ON u.id = i.author_id
		INNER JOIN channels c ON c.id = i.channel_id
		LEFT JOIN workspace_members wm
			ON wm.workspace_id = c.workspace_id AND wm.user_id = i.author_id
	`, channelID, authorID, body, contentType, payload).Scan(scanMessageFields(&row)...)
	if err != nil {
		return MessageRow{}, fmt.Errorf("insert app message: %w", err)
	}
	return row, nil
}

func (s *Store) ListMessages(
	ctx context.Context,
	channelID uuid.UUID,
	before *time.Time,
	beforeID *uuid.UUID,
	after *time.Time,
	limit int,
) ([]MessageRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT`+messageSelectSQL+`
		FROM messages m
		INNER JOIN users u ON u.id = m.author_id
		INNER JOIN channels c ON c.id = m.channel_id
		LEFT JOIN workspace_members wm
			ON wm.workspace_id = c.workspace_id AND wm.user_id = m.author_id
		WHERE m.channel_id = $1
		  AND ($2::timestamptz IS NULL OR m.created_at >= $2)
		  AND (
			$3::timestamptz IS NULL OR $4::uuid IS NULL
			OR (m.created_at, m.id) < ($3, $4)
		  )
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $5
	`, channelID, after, before, beforeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []MessageRow
	for rows.Next() {
		var row MessageRow
		if err := rows.Scan(scanMessageFields(&row)...); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetMessage(ctx context.Context, messageID uuid.UUID) (MessageRow, error) {
	var row MessageRow
	err := s.pool.QueryRow(ctx, `
		SELECT`+messageSelectSQL+`
		FROM messages m
		INNER JOIN users u ON u.id = m.author_id
		INNER JOIN channels c ON c.id = m.channel_id
		LEFT JOIN workspace_members wm
			ON wm.workspace_id = c.workspace_id AND wm.user_id = m.author_id
		WHERE m.id = $1
	`, messageID).Scan(scanMessageFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageRow{}, ErrNotFound
	}
	if err != nil {
		return MessageRow{}, fmt.Errorf("get message: %w", err)
	}
	return row, nil
}

func (s *Store) UpdateMessageBody(
	ctx context.Context,
	messageID uuid.UUID,
	body string,
) (MessageRow, error) {
	var row MessageRow
	err := s.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE messages
			SET body = $2,
				updated_at = now()
			WHERE id = $1
			RETURNING id, channel_id, author_id, body, content_type, payload, created_at, updated_at
		)
		SELECT u.id, u.channel_id, u.author_id, usr.display_name, COALESCE(wm.handle, ''),
			(usr.avatar_bytes IS NOT NULL), usr.avatar_updated_at, COALESCE(usr.kind, 'human'),
			u.body, u.content_type, u.payload, u.created_at, u.updated_at
		FROM updated u
		INNER JOIN users usr ON usr.id = u.author_id
		INNER JOIN channels c ON c.id = u.channel_id
		LEFT JOIN workspace_members wm
			ON wm.workspace_id = c.workspace_id AND wm.user_id = u.author_id
	`, messageID, body).Scan(scanMessageFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageRow{}, ErrNotFound
	}
	if err != nil {
		return MessageRow{}, fmt.Errorf("update message: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM messages WHERE id = $1`, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertScheduledMessage(
	ctx context.Context,
	channelID, authorID uuid.UUID,
	body, contentType string,
	sendAt time.Time,
) (ScheduledMessageRow, error) {
	var row ScheduledMessageRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scheduled_messages (channel_id, author_id, body, content_type, send_at, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, channel_id, author_id, body, content_type, send_at, status, sent_message_id, error, created_at, updated_at
	`, channelID, authorID, body, contentType, sendAt).Scan(
		&row.ID, &row.ChannelID, &row.AuthorID, &row.Body, &row.ContentType, &row.SendAt,
		&row.Status, &row.SentMessageID, &row.Error, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return ScheduledMessageRow{}, fmt.Errorf("insert scheduled message: %w", err)
	}
	return row, nil
}

func (s *Store) ListScheduledMessages(ctx context.Context, channelID uuid.UUID) ([]ScheduledMessageRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, channel_id, author_id, body, content_type, send_at, status, sent_message_id, error, created_at, updated_at
		FROM scheduled_messages
		WHERE channel_id = $1 AND status = 'pending'
		ORDER BY send_at ASC
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list scheduled: %w", err)
	}
	defer rows.Close()

	var out []ScheduledMessageRow
	for rows.Next() {
		var row ScheduledMessageRow
		if err := rows.Scan(
			&row.ID, &row.ChannelID, &row.AuthorID, &row.Body, &row.ContentType, &row.SendAt,
			&row.Status, &row.SentMessageID, &row.Error, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scheduled: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetScheduledMessage(ctx context.Context, id uuid.UUID) (ScheduledMessageRow, error) {
	var row ScheduledMessageRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, channel_id, author_id, body, content_type, send_at, status, sent_message_id, error, created_at, updated_at
		FROM scheduled_messages WHERE id = $1
	`, id).Scan(
		&row.ID, &row.ChannelID, &row.AuthorID, &row.Body, &row.ContentType, &row.SendAt,
		&row.Status, &row.SentMessageID, &row.Error, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduledMessageRow{}, ErrNotFound
	}
	if err != nil {
		return ScheduledMessageRow{}, fmt.Errorf("get scheduled: %w", err)
	}
	return row, nil
}

func (s *Store) CancelScheduledMessage(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE scheduled_messages
		SET status = 'cancelled', updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, id)
	if err != nil {
		return fmt.Errorf("cancel scheduled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PublishedScheduledMessage is a message created from a due scheduled row.
type PublishedScheduledMessage struct {
	ID        uuid.UUID
	ChannelID uuid.UUID
	AuthorID  uuid.UUID
	Body      string
}

// ClaimAndPublishDueScheduledMessages publishes pending rows due at or before now.
func (s *Store) ClaimAndPublishDueScheduledMessages(
	ctx context.Context, now time.Time, limit int,
) ([]PublishedScheduledMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, channel_id, author_id, body, content_type
		FROM scheduled_messages
		WHERE status = 'pending' AND send_at <= $1
		ORDER BY send_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("claim scheduled: %w", err)
	}

	type due struct {
		id, channelID, authorID uuid.UUID
		body, contentType       string
	}
	var items []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.channelID, &d.authorID, &d.body, &d.contentType); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan due: %w", err)
		}
		items = append(items, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var published []PublishedScheduledMessage
	for _, d := range items {
		var messageID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO messages (channel_id, author_id, body, content_type)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, d.channelID, d.authorID, d.body, d.contentType).Scan(&messageID)
		if err != nil {
			_, _ = tx.Exec(ctx, `
				UPDATE scheduled_messages
				SET status = 'failed', error = $2, updated_at = now()
				WHERE id = $1
			`, d.id, err.Error())
			continue
		}
		_, err = tx.Exec(ctx, `
			UPDATE scheduled_messages
			SET status = 'sent', sent_message_id = $2, updated_at = now()
			WHERE id = $1
		`, d.id, messageID)
		if err != nil {
			return published, fmt.Errorf("mark sent: %w", err)
		}
		published = append(published, PublishedScheduledMessage{
			ID: messageID, ChannelID: d.channelID, AuthorID: d.authorID, Body: d.body,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return published, nil
}

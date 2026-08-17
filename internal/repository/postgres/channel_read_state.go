package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ChannelReadState struct {
	UserID             uuid.UUID
	ChannelID          uuid.UUID
	LastReadMessageID  *uuid.UUID
	HasRow             bool
	UnreadCount        int
	FirstUnreadMessage *uuid.UUID
}

func (s *Store) UpsertChannelReadState(
	ctx context.Context,
	userID, channelID uuid.UUID,
	lastReadMessageID *uuid.UUID,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channel_read_state (user_id, channel_id, last_read_message_id, last_read_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, channel_id) DO UPDATE
		SET last_read_message_id = EXCLUDED.last_read_message_id,
			last_read_at = now()
	`, userID, channelID, lastReadMessageID)
	if err != nil {
		return fmt.Errorf("upsert channel read state: %w", err)
	}
	return nil
}

func (s *Store) GetChannelReadState(
	ctx context.Context,
	userID, channelID uuid.UUID,
) (ChannelReadState, error) {
	var state ChannelReadState
	state.UserID = userID
	state.ChannelID = channelID
	err := s.pool.QueryRow(ctx, `
		SELECT last_read_message_id
		FROM channel_read_state
		WHERE user_id = $1 AND channel_id = $2
	`, userID, channelID).Scan(&state.LastReadMessageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return ChannelReadState{}, fmt.Errorf("get channel read state: %w", err)
	}
	state.HasRow = true
	return state, nil
}

func (s *Store) GetPreviousMessageID(
	ctx context.Context,
	channelID, messageID uuid.UUID,
) (*uuid.UUID, error) {
	var (
		createdAt time.Time
		id        uuid.UUID
	)
	err := s.pool.QueryRow(ctx, `
		SELECT created_at, id
		FROM messages
		WHERE id = $1 AND channel_id = $2
	`, messageID, channelID).Scan(&createdAt, &id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get message for previous: %w", err)
	}

	var prev uuid.UUID
	err = s.pool.QueryRow(ctx, `
		SELECT id
		FROM messages
		WHERE channel_id = $1
		  AND (created_at, id) < ($2, $3)
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, channelID, createdAt, id).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get previous message: %w", err)
	}
	return &prev, nil
}

func (s *Store) GetLatestMessageID(ctx context.Context, channelID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id
		FROM messages
		WHERE channel_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, channelID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest message: %w", err)
	}
	return &id, nil
}

// EnsureChannelReadBaselines creates caught-up read cursors for accessible
// channels that have none yet, so later messages can surface as unread.
func (s *Store) EnsureChannelReadBaselines(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channel_read_state (user_id, channel_id, last_read_message_id, last_read_at)
		SELECT $2, c.id, (
			SELECT m.id
			FROM messages m
			WHERE m.channel_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		), now()
		FROM channels c
		WHERE c.workspace_id = $1
		  AND (
			c.is_private = false
			OR EXISTS (
				SELECT 1 FROM channel_members cm
				WHERE cm.channel_id = c.id AND cm.user_id = $2
			)
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM channel_read_state rs
			WHERE rs.channel_id = c.id AND rs.user_id = $2
		  )
	`, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("ensure channel read baselines: %w", err)
	}
	return nil
}

func (s *Store) GetChannelUnreadSummary(
	ctx context.Context,
	userID, channelID uuid.UUID,
) (unreadCount int, firstUnread *uuid.UUID, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT
			CASE
				WHEN rs.user_id IS NULL THEN 0
				ELSE (
					SELECT COUNT(*)::int
					FROM messages m
					WHERE m.channel_id = $2
					  AND (
						rs.last_read_message_id IS NULL
						OR (m.created_at, m.id) > (
							SELECT lr.created_at, lr.id
							FROM messages lr
							WHERE lr.id = rs.last_read_message_id
						)
					  )
				)
			END AS unread_count,
			CASE
				WHEN rs.user_id IS NULL THEN NULL
				ELSE (
					SELECT m.id
					FROM messages m
					WHERE m.channel_id = $2
					  AND (
						rs.last_read_message_id IS NULL
						OR (m.created_at, m.id) > (
							SELECT lr.created_at, lr.id
							FROM messages lr
							WHERE lr.id = rs.last_read_message_id
						)
					  )
					ORDER BY m.created_at ASC, m.id ASC
					LIMIT 1
				)
			END AS first_unread_message_id
		FROM (SELECT $1::uuid AS user_id, $2::uuid AS channel_id) keys
		LEFT JOIN channel_read_state rs
			ON rs.user_id = keys.user_id AND rs.channel_id = keys.channel_id
	`, userID, channelID).Scan(&unreadCount, &firstUnread)
	if err != nil {
		return 0, nil, fmt.Errorf("get channel unread summary: %w", err)
	}
	return unreadCount, firstUnread, nil
}

// MessageIsAtOrBefore reports whether candidate is at or before reference in channel order.
func MessageIsAtOrBefore(candidate, reference MessageRow) bool {
	if candidate.CreatedAt.Before(reference.CreatedAt) {
		return true
	}
	if candidate.CreatedAt.After(reference.CreatedAt) {
		return false
	}
	return candidate.ID.String() <= reference.ID.String()
}

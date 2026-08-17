package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MessageReactionRow struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
	CreatedAt time.Time
}

func (s *Store) AddMessageReaction(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("add message reaction: %w", err)
	}
	return nil
}

func (s *Store) RemoveMessageReaction(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM message_reactions
		WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`, messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("remove message reaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListReactionsForMessages(
	ctx context.Context,
	messageIDs []uuid.UUID,
) ([]MessageReactionRow, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT message_id, user_id, emoji, created_at
		FROM message_reactions
		WHERE message_id = ANY($1)
		ORDER BY created_at ASC, emoji ASC, user_id ASC
	`, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("list message reactions: %w", err)
	}
	defer rows.Close()

	var out []MessageReactionRow
	for rows.Next() {
		var row MessageReactionRow
		if err := rows.Scan(&row.MessageID, &row.UserID, &row.Emoji, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message reaction: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) HasMessageReaction(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM message_reactions
			WHERE message_id = $1 AND user_id = $2 AND emoji = $3
		)
	`, messageID, userID, emoji).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has message reaction: %w", err)
	}
	return exists, nil
}

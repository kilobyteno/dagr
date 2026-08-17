package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

var ErrChannelNameConflict = errors.New("channel name already exists")

const channelSelectColumns = `
	id, workspace_id, name, COALESCE(topic, ''), is_private,
	COALESCE(kind, 'channel'), COALESCE(sharing_mode, 'local'),
	created_by, created_at, updated_at
`

func scanChannelFields(row *ChannelRow) []any {
	return []any{
		&row.ID, &row.WorkspaceID, &row.Name, &row.Topic, &row.IsPrivate,
		&row.Kind, &row.SharingMode, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	}
}

func (s *Store) CreateChannel(
	ctx context.Context,
	workspaceID, createdBy uuid.UUID,
	name, topic string,
	isPrivate bool,
) (ChannelRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ChannelRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var row ChannelRow
	err = tx.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, name, topic, is_private, kind, created_by, updated_by)
		VALUES ($1, $2, NULLIF($3, ''), $4, 'channel', $5, $5)
		RETURNING `+channelSelectColumns+`
	`, workspaceID, name, topic, isPrivate, createdBy).Scan(scanChannelFields(&row)...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ChannelRow{}, ErrChannelNameConflict
		}
		return ChannelRow{}, fmt.Errorf("insert channel: %w", err)
	}

	if isPrivate {
		_, err = tx.Exec(ctx, `
			INSERT INTO channel_members (channel_id, user_id, role)
			VALUES ($1, $2, $3)
		`, row.ID, createdBy, domain.ChannelMemberRoleAdmin)
		if err != nil {
			return ChannelRow{}, fmt.Errorf("insert channel member: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ChannelRow{}, fmt.Errorf("commit: %w", err)
	}
	return row, nil
}

func (s *Store) GetChannel(ctx context.Context, channelID uuid.UUID) (ChannelRow, error) {
	var row ChannelRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+channelSelectColumns+`
		FROM channels WHERE id = $1
	`, channelID).Scan(scanChannelFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelRow{}, ErrNotFound
	}
	if err != nil {
		return ChannelRow{}, fmt.Errorf("get channel: %w", err)
	}
	return row, nil
}

func (s *Store) GetDMChannelForUser(
	ctx context.Context, channelID, viewerID uuid.UUID,
) (ChannelRow, error) {
	var row ChannelRow
	err := s.pool.QueryRow(ctx, `
		SELECT c.id, c.workspace_id, c.name, COALESCE(c.topic, ''), c.is_private,
			COALESCE(c.kind, 'channel'), COALESCE(c.sharing_mode, 'local'),
			c.created_by, c.created_at, c.updated_at,
			peer.id, COALESCE(peer.display_name, ''), COALESCE(wm_peer.handle, ''),
			(peer.avatar_bytes IS NOT NULL), peer.avatar_updated_at
		FROM channels c
		INNER JOIN dm_pairs dp ON dp.channel_id = c.id
		INNER JOIN users peer ON peer.id = CASE
			WHEN dp.user_a = $2 THEN dp.user_b
			ELSE dp.user_a
		END
		LEFT JOIN workspace_members wm_peer
			ON wm_peer.workspace_id = c.workspace_id AND wm_peer.user_id = peer.id
		WHERE c.id = $1 AND c.kind = 'dm'
		  AND (dp.user_a = $2 OR dp.user_b = $2)
	`, channelID, viewerID).Scan(
		&row.ID, &row.WorkspaceID, &row.Name, &row.Topic, &row.IsPrivate,
		&row.Kind, &row.SharingMode, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
		&row.PeerUserID, &row.PeerDisplayName, &row.PeerHandle,
		&row.PeerHasAvatar, &row.PeerAvatarUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelRow{}, ErrNotFound
	}
	if err != nil {
		return ChannelRow{}, fmt.Errorf("get dm channel: %w", err)
	}
	return row, nil
}

func orderedDMPair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if strings.Compare(a.String(), b.String()) < 0 {
		return a, b
	}
	return b, a
}

func (s *Store) FindDMChannel(
	ctx context.Context, workspaceID, userA, userB uuid.UUID,
) (ChannelRow, error) {
	left, right := orderedDMPair(userA, userB)
	var channelID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT channel_id FROM dm_pairs
		WHERE workspace_id = $1 AND user_a = $2 AND user_b = $3
	`, workspaceID, left, right).Scan(&channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelRow{}, ErrNotFound
	}
	if err != nil {
		return ChannelRow{}, fmt.Errorf("find dm pair: %w", err)
	}
	return s.GetDMChannelForUser(ctx, channelID, userA)
}

func (s *Store) CreateDMChannel(
	ctx context.Context, workspaceID, createdBy, peerID uuid.UUID,
) (ChannelRow, error) {
	left, right := orderedDMPair(createdBy, peerID)
	name := fmt.Sprintf("dm_%s_%s", left.String(), right.String())

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ChannelRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var row ChannelRow
	err = tx.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, name, topic, is_private, kind, created_by, updated_by)
		VALUES ($1, $2, '', true, 'dm', $3, $3)
		RETURNING `+channelSelectColumns+`
	`, workspaceID, name, createdBy).Scan(scanChannelFields(&row)...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ChannelRow{}, ErrChannelNameConflict
		}
		return ChannelRow{}, fmt.Errorf("insert dm channel: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO channel_members (channel_id, user_id, role)
		VALUES ($1, $2, $3), ($1, $4, $3)
	`, row.ID, createdBy, domain.ChannelMemberRoleMember, peerID); err != nil {
		return ChannelRow{}, fmt.Errorf("insert dm members: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO dm_pairs (channel_id, workspace_id, user_a, user_b)
		VALUES ($1, $2, $3, $4)
	`, row.ID, workspaceID, left, right); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ChannelRow{}, ErrChannelNameConflict
		}
		return ChannelRow{}, fmt.Errorf("insert dm pair: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ChannelRow{}, fmt.Errorf("commit: %w", err)
	}
	return s.GetDMChannelForUser(ctx, row.ID, createdBy)
}

func (s *Store) UpdateChannel(
	ctx context.Context,
	channelID, updatedBy uuid.UUID,
	name, topic string,
	isPrivate bool,
) (ChannelRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ChannelRow{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var row ChannelRow
	err = tx.QueryRow(ctx, `
		UPDATE channels
		SET name = $2, topic = NULLIF($3, ''), is_private = $4, updated_by = $5, updated_at = now()
		WHERE id = $1 AND kind = 'channel'
		RETURNING `+channelSelectColumns+`
	`, channelID, name, topic, isPrivate, updatedBy).Scan(scanChannelFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelRow{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ChannelRow{}, ErrChannelNameConflict
		}
		return ChannelRow{}, fmt.Errorf("update channel: %w", err)
	}

	if isPrivate {
		_, err = tx.Exec(ctx, `
			INSERT INTO channel_members (channel_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (channel_id, user_id) DO NOTHING
		`, row.ID, updatedBy, domain.ChannelMemberRoleAdmin)
		if err != nil {
			return ChannelRow{}, fmt.Errorf("ensure updater membership: %w", err)
		}
		if row.CreatedBy != updatedBy {
			_, err = tx.Exec(ctx, `
				INSERT INTO channel_members (channel_id, user_id, role)
				VALUES ($1, $2, $3)
				ON CONFLICT (channel_id, user_id) DO NOTHING
			`, row.ID, row.CreatedBy, domain.ChannelMemberRoleAdmin)
			if err != nil {
				return ChannelRow{}, fmt.Errorf("ensure creator membership: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ChannelRow{}, fmt.Errorf("commit: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteChannel(ctx context.Context, channelID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM channels WHERE id = $1 AND kind = 'channel'
	`, channelID)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM channel_members WHERE channel_id = $1 AND user_id = $2
		)
	`, channelID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is channel member: %w", err)
	}
	return exists, nil
}

func (s *Store) ListChannelMemberIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id FROM channel_members WHERE channel_id = $1
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list channel member ids: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan channel member id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) AddChannelMember(
	ctx context.Context,
	channelID, userID uuid.UUID,
	role domain.ChannelMemberRole,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channel_members (channel_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel_id, user_id) DO NOTHING
	`, channelID, userID, role)
	if err != nil {
		return fmt.Errorf("add channel member: %w", err)
	}
	return nil
}

func (s *Store) RemoveChannelMember(ctx context.Context, channelID, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM channel_members WHERE channel_id = $1 AND user_id = $2
	`, channelID, userID)
	if err != nil {
		return fmt.Errorf("remove channel member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) IsWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) (bool, string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("is workspace member: %w", err)
	}
	return true, role, nil
}

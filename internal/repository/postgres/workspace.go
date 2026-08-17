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

var ErrSlugConflict = errors.New("workspace slug already exists")

type WorkspaceRow struct {
	ID              uuid.UUID
	Name            string
	Slug            string
	CreatedBy       uuid.UUID
	HasIcon         bool
	IconContentType string
	IconUpdatedAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Role            string
}

func (w WorkspaceRow) ToDomain() domain.Workspace {
	return domain.Workspace{
		ID:            w.ID.String(),
		Name:          w.Name,
		Slug:          w.Slug,
		Role:          domain.WorkspaceRole(w.Role),
		HasIcon:       w.HasIcon,
		IconUpdatedAt: w.IconUpdatedAt,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

type ChannelRow struct {
	ID                   uuid.UUID
	WorkspaceID          uuid.UUID
	Name                 string
	Topic                string
	IsPrivate            bool
	Kind                 string
	SharingMode          string
	CreatedBy            uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
	UnreadCount          int
	FirstUnreadMessageID *uuid.UUID
	PeerUserID           *uuid.UUID
	PeerDisplayName      string
	PeerHandle           string
	PeerHasAvatar        bool
	PeerAvatarUpdatedAt  *time.Time
}

func (c ChannelRow) ToDomain() domain.Channel {
	kind := c.Kind
	if kind == "" {
		kind = "channel"
	}
	sharing := c.SharingMode
	if sharing == "" {
		sharing = "local"
	}
	out := domain.Channel{
		ID:          c.ID.String(),
		WorkspaceID: c.WorkspaceID.String(),
		Name:        c.Name,
		Topic:       c.Topic,
		IsPrivate:   c.IsPrivate,
		IsDM:        kind == "dm",
		IsShared:    sharing == "shared",
		CreatedBy:   c.CreatedBy.String(),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		UnreadCount: c.UnreadCount,
	}
	if c.FirstUnreadMessageID != nil {
		out.FirstUnreadMessageID = c.FirstUnreadMessageID.String()
	}
	if c.PeerUserID != nil {
		out.PeerUserID = c.PeerUserID.String()
		out.PeerDisplayName = c.PeerDisplayName
		out.PeerHandle = c.PeerHandle
		out.PeerHasAvatar = c.PeerHasAvatar
		out.PeerAvatarUpdatedAt = c.PeerAvatarUpdatedAt
	}
	return out
}

type CreateWorkspaceResult struct {
	Workspace WorkspaceRow
	Channels  []ChannelRow
}

func (s *Store) CreateWorkspace(
	ctx context.Context,
	name, slug string,
	createdBy uuid.UUID,
) (CreateWorkspaceResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var workspace WorkspaceRow
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, created_by, created_at, updated_at,
			(icon_bytes IS NOT NULL), COALESCE(icon_content_type, ''), icon_updated_at
	`, name, slug, createdBy).Scan(
		&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.CreatedBy,
		&workspace.CreatedAt, &workspace.UpdatedAt,
		&workspace.HasIcon, &workspace.IconContentType, &workspace.IconUpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return CreateWorkspaceResult{}, ErrSlugConflict
		}
		return CreateWorkspaceResult{}, fmt.Errorf("insert workspace: %w", err)
	}
	workspace.Role = string(domain.WorkspaceRoleOwner)

	var ownerDisplayName string
	err = tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, createdBy).Scan(&ownerDisplayName)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("load owner display name: %w", err)
	}
	ownerHandle, err := allocateUniqueHandle(ctx, tx, workspace.ID, ownerDisplayName)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("allocate owner handle: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, handle)
		VALUES ($1, $2, $3, $4)
	`, workspace.ID, createdBy, domain.WorkspaceRoleOwner, ownerHandle)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("insert member: %w", err)
	}

	var channel ChannelRow
	err = tx.QueryRow(ctx, `
		INSERT INTO channels (workspace_id, name, is_private, kind, created_by)
		VALUES ($1, 'general', false, 'channel', $2)
		RETURNING id, workspace_id, name, COALESCE(topic, ''), is_private,
			COALESCE(kind, 'channel'), created_by, created_at, updated_at
	`, workspace.ID, createdBy).Scan(
		&channel.ID, &channel.WorkspaceID, &channel.Name, &channel.Topic, &channel.IsPrivate,
		&channel.Kind, &channel.CreatedBy, &channel.CreatedAt, &channel.UpdatedAt,
	)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("insert general channel: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workspace_billing (workspace_id, plan, status, billable_seats)
		VALUES ($1, 'free', 'active', 1)
	`, workspace.ID)
	if err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("insert workspace billing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return CreateWorkspaceResult{}, fmt.Errorf("commit: %w", err)
	}

	return CreateWorkspaceResult{
		Workspace: workspace,
		Channels:  []ChannelRow{channel},
	}, nil
}

func (s *Store) ListWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.created_by, w.created_at, w.updated_at, m.role,
			(w.icon_bytes IS NOT NULL), COALESCE(w.icon_content_type, ''), w.icon_updated_at
		FROM workspaces w
		INNER JOIN workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1
		ORDER BY w.name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var out []WorkspaceRow
	for rows.Next() {
		var row WorkspaceRow
		if err := rows.Scan(
			&row.ID, &row.Name, &row.Slug, &row.CreatedBy,
			&row.CreatedAt, &row.UpdatedAt, &row.Role,
			&row.HasIcon, &row.IconContentType, &row.IconUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspaceForUser(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (WorkspaceRow, error) {
	var row WorkspaceRow
	err := s.pool.QueryRow(ctx, `
		SELECT w.id, w.name, w.slug, w.created_by, w.created_at, w.updated_at, m.role,
			(w.icon_bytes IS NOT NULL), COALESCE(w.icon_content_type, ''), w.icon_updated_at
		FROM workspaces w
		INNER JOIN workspace_members m ON m.workspace_id = w.id
		WHERE w.id = $1 AND m.user_id = $2
	`, workspaceID, userID).Scan(
		&row.ID, &row.Name, &row.Slug, &row.CreatedBy,
		&row.CreatedAt, &row.UpdatedAt, &row.Role,
		&row.HasIcon, &row.IconContentType, &row.IconUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceRow{}, fmt.Errorf("get workspace: %w", err)
	}
	return row, nil
}

func (s *Store) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = $1)
	`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug exists: %w", err)
	}
	return exists, nil
}

func (s *Store) SlugExistsExcluding(ctx context.Context, slug string, excludeID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = $1 AND id <> $2)
	`, slug, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("slug exists excluding: %w", err)
	}
	return exists, nil
}

func (s *Store) UpdateWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	name, slug string,
) (WorkspaceRow, error) {
	var row WorkspaceRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspaces
		SET name = $2, slug = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, created_by, created_at, updated_at,
			(icon_bytes IS NOT NULL), COALESCE(icon_content_type, ''), icon_updated_at
	`, workspaceID, name, slug).Scan(
		&row.ID, &row.Name, &row.Slug, &row.CreatedBy,
		&row.CreatedAt, &row.UpdatedAt,
		&row.HasIcon, &row.IconContentType, &row.IconUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceRow{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceRow{}, ErrSlugConflict
		}
		return WorkspaceRow{}, fmt.Errorf("update workspace: %w", err)
	}
	return row, nil
}

type WorkspaceIcon struct {
	ContentType string
	Bytes       []byte
	UpdatedAt   time.Time
}

func (s *Store) SetWorkspaceIcon(
	ctx context.Context,
	workspaceID uuid.UUID,
	contentType string,
	data []byte,
) (WorkspaceRow, error) {
	var row WorkspaceRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspaces
		SET icon_bytes = $2,
			icon_content_type = $3,
			icon_updated_at = now(),
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, created_by, created_at, updated_at,
			(icon_bytes IS NOT NULL), COALESCE(icon_content_type, ''), icon_updated_at
	`, workspaceID, data, contentType).Scan(
		&row.ID, &row.Name, &row.Slug, &row.CreatedBy,
		&row.CreatedAt, &row.UpdatedAt,
		&row.HasIcon, &row.IconContentType, &row.IconUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceRow{}, fmt.Errorf("set workspace icon: %w", err)
	}
	return row, nil
}

func (s *Store) ClearWorkspaceIcon(
	ctx context.Context,
	workspaceID uuid.UUID,
) (WorkspaceRow, error) {
	var row WorkspaceRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspaces
		SET icon_bytes = NULL,
			icon_content_type = NULL,
			icon_updated_at = NULL,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, created_by, created_at, updated_at,
			(icon_bytes IS NOT NULL), COALESCE(icon_content_type, ''), icon_updated_at
	`, workspaceID).Scan(
		&row.ID, &row.Name, &row.Slug, &row.CreatedBy,
		&row.CreatedAt, &row.UpdatedAt,
		&row.HasIcon, &row.IconContentType, &row.IconUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceRow{}, fmt.Errorf("clear workspace icon: %w", err)
	}
	return row, nil
}

func (s *Store) GetWorkspaceIcon(
	ctx context.Context,
	workspaceID uuid.UUID,
) (WorkspaceIcon, error) {
	var icon WorkspaceIcon
	var updatedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT icon_bytes, COALESCE(icon_content_type, ''), icon_updated_at
		FROM workspaces
		WHERE id = $1 AND icon_bytes IS NOT NULL
	`, workspaceID).Scan(&icon.Bytes, &icon.ContentType, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceIcon{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceIcon{}, fmt.Errorf("get workspace icon: %w", err)
	}
	if updatedAt != nil {
		icon.UpdatedAt = *updatedAt
	}
	return icon, nil
}

func (s *Store) DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1`, workspaceID)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListChannelsForWorkspace(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) ([]ChannelRow, error) {
	var memberCount int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(&memberCount)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if memberCount == 0 {
		return nil, ErrNotFound
	}

	if err := s.EnsureChannelReadBaselines(ctx, workspaceID, userID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.workspace_id, c.name, COALESCE(c.topic, ''), c.is_private,
			COALESCE(c.kind, 'channel'), COALESCE(c.sharing_mode, 'local'),
			c.created_by, c.created_at, c.updated_at,
			CASE
				WHEN rs.user_id IS NULL THEN 0
				ELSE (
					SELECT COUNT(*)::int
					FROM messages m
					WHERE m.channel_id = c.id
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
					WHERE m.channel_id = c.id
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
			END AS first_unread_message_id,
			peer.id, COALESCE(peer.display_name, ''), COALESCE(wm_peer.handle, ''),
			(peer.avatar_bytes IS NOT NULL), peer.avatar_updated_at
		FROM channels c
		LEFT JOIN channel_read_state rs
			ON rs.channel_id = c.id AND rs.user_id = $2
		LEFT JOIN dm_pairs dp
			ON dp.channel_id = c.id AND c.kind = 'dm'
		LEFT JOIN users peer
			ON peer.id = CASE
				WHEN dp.user_a = $2 THEN dp.user_b
				WHEN dp.user_b = $2 THEN dp.user_a
				ELSE NULL
			END
		LEFT JOIN workspace_members wm_peer
			ON wm_peer.workspace_id = c.workspace_id AND wm_peer.user_id = peer.id
		WHERE c.workspace_id = $1
		  AND (
			c.is_private = false
			OR EXISTS (
				SELECT 1 FROM channel_members cm
				WHERE cm.channel_id = c.id AND cm.user_id = $2
			)
		  )
		ORDER BY
			CASE WHEN c.kind = 'dm' THEN 1 ELSE 0 END,
			lower(COALESCE(peer.display_name, c.name)) ASC
	`, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	var out []ChannelRow
	for rows.Next() {
		var row ChannelRow
		if err := rows.Scan(
			&row.ID, &row.WorkspaceID, &row.Name, &row.Topic, &row.IsPrivate,
			&row.Kind, &row.SharingMode, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
			&row.UnreadCount, &row.FirstUnreadMessageID,
			&row.PeerUserID, &row.PeerDisplayName, &row.PeerHandle,
			&row.PeerHasAvatar, &row.PeerAvatarUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

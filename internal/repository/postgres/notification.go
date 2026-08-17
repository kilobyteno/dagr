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

type NotificationRow struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ActorID     *uuid.UUID
	ActorName   string
	Kind        string
	WorkspaceID *uuid.UUID
	ChannelID   *uuid.UUID
	ChannelName string
	IsDM        bool
	MessageID   *uuid.UUID
	Body        string
	ReadAt      *time.Time
	CreatedAt   time.Time
}

const notificationChannelLabelSQL = `
	CASE
		WHEN COALESCE(c.kind, 'channel') = 'dm' THEN
			COALESCE(NULLIF(peer.display_name, ''), 'Direct message')
		ELSE COALESCE(c.name, '')
	END
`

const notificationChannelJoinsSQL = `
	LEFT JOIN channels c ON c.id = n.channel_id
	LEFT JOIN dm_pairs dp ON dp.channel_id = c.id AND COALESCE(c.kind, 'channel') = 'dm'
	LEFT JOIN users peer ON peer.id = CASE
		WHEN dp.user_a = n.user_id THEN dp.user_b
		WHEN dp.user_b = n.user_id THEN dp.user_a
		ELSE NULL
	END
`

func (r NotificationRow) ToDomain() domain.Notification {
	out := domain.Notification{
		ID:          r.ID.String(),
		UserID:      r.UserID.String(),
		ActorName:   r.ActorName,
		Kind:        domain.NotificationKind(r.Kind),
		Body:        r.Body,
		ReadAt:      r.ReadAt,
		CreatedAt:   r.CreatedAt,
		ChannelName: r.ChannelName,
		IsDM:        r.IsDM,
	}
	if r.ActorID != nil {
		out.ActorID = r.ActorID.String()
	}
	if r.WorkspaceID != nil {
		out.WorkspaceID = r.WorkspaceID.String()
	}
	if r.ChannelID != nil {
		out.ChannelID = r.ChannelID.String()
	}
	if r.MessageID != nil {
		out.MessageID = r.MessageID.String()
	}
	return out
}

type CreateNotificationInput struct {
	UserID      uuid.UUID
	ActorID     *uuid.UUID
	Kind        domain.NotificationKind
	WorkspaceID *uuid.UUID
	ChannelID   *uuid.UUID
	MessageID   *uuid.UUID
	Body        string
}

type WorkspaceMemberInfo struct {
	UserID                uuid.UUID
	DisplayName           string
	Handle                string
	FormerHandles         []string
	Role                  string
	Kind                  string
	HomeWorkspaceID       *uuid.UUID
	HomeWorkspaceName     string
	HomeServerID          string
	HomeWorkspaceRemoteID string
	HomeWorkspaceIconURL  string
	HomeServerURL         string
	StatusEmoji           string
	StatusText            string
	StatusExpiresAt       *time.Time
	HasAvatar             bool
	AvatarUpdatedAt       *time.Time
}

const workspaceMemberSelectSQL = `
	wm.user_id, u.display_name, wm.handle, wm.role,
	COALESCE(wm.kind, 'member'), wm.home_workspace_id,
	COALESCE(wm.home_workspace_name, COALESCE(hw.name, '')),
	COALESCE(wm.home_server_id, ''), COALESCE(wm.home_workspace_remote_id, ''),
	COALESCE(wm.home_workspace_icon_url, ''), COALESCE(fp.public_url, ''),
	COALESCE(u.status_emoji, ''), COALESCE(u.status_text, ''), u.status_expires_at,
	(u.avatar_bytes IS NOT NULL), u.avatar_updated_at
`

func scanWorkspaceMember(m *WorkspaceMemberInfo) []any {
	return []any{
		&m.UserID, &m.DisplayName, &m.Handle, &m.Role,
		&m.Kind, &m.HomeWorkspaceID, &m.HomeWorkspaceName,
		&m.HomeServerID, &m.HomeWorkspaceRemoteID, &m.HomeWorkspaceIconURL, &m.HomeServerURL,
		&m.StatusEmoji, &m.StatusText, &m.StatusExpiresAt,
		&m.HasAvatar, &m.AvatarUpdatedAt,
	}
}

var ErrHandleConflict = errors.New("workspace handle already taken")

func (s *Store) CreateNotification(ctx context.Context, in CreateNotificationInput) (NotificationRow, error) {
	var row NotificationRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, actor_id, kind, workspace_id, channel_id, message_id, body)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, actor_id, kind, workspace_id, channel_id, message_id, body, read_at, created_at
	`, in.UserID, in.ActorID, string(in.Kind), in.WorkspaceID, in.ChannelID, in.MessageID, in.Body).Scan(
		&row.ID, &row.UserID, &row.ActorID, &row.Kind, &row.WorkspaceID, &row.ChannelID,
		&row.MessageID, &row.Body, &row.ReadAt, &row.CreatedAt,
	)
	if err != nil {
		return NotificationRow{}, fmt.Errorf("insert notification: %w", err)
	}
	return row, nil
}

func (s *Store) ListNotifications(
	ctx context.Context,
	userID uuid.UUID,
	filter string,
	limit int,
) ([]NotificationRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
		SELECT n.id, n.user_id, n.actor_id, COALESCE(u.display_name, ''), n.kind,
			n.workspace_id, n.channel_id, ` + notificationChannelLabelSQL + `,
			COALESCE(c.kind, 'channel') = 'dm', n.message_id,
			n.body, n.read_at, n.created_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		` + notificationChannelJoinsSQL + `
		WHERE n.user_id = $1
	`
	args := []any{userID}
	switch filter {
	case "unread":
		query += ` AND n.read_at IS NULL`
	case "mentions":
		query += ` AND n.kind = 'mention'`
	}
	query += ` ORDER BY n.created_at DESC LIMIT $2`
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var out []NotificationRow
	for rows.Next() {
		var row NotificationRow
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.ActorID, &row.ActorName, &row.Kind,
			&row.WorkspaceID, &row.ChannelID, &row.ChannelName, &row.IsDM, &row.MessageID,
			&row.Body, &row.ReadAt, &row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return n, nil
}

func (s *Store) MarkNotificationRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, now())
		WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (s *Store) GetNotification(ctx context.Context, userID, notificationID uuid.UUID) (NotificationRow, error) {
	var row NotificationRow
	err := s.pool.QueryRow(ctx, `
		SELECT n.id, n.user_id, n.actor_id, COALESCE(u.display_name, ''), n.kind,
			n.workspace_id, n.channel_id, `+notificationChannelLabelSQL+`,
			COALESCE(c.kind, 'channel') = 'dm', n.message_id,
			n.body, n.read_at, n.created_at
		FROM notifications n
		LEFT JOIN users u ON u.id = n.actor_id
		`+notificationChannelJoinsSQL+`
		WHERE n.id = $1 AND n.user_id = $2
	`, notificationID, userID).Scan(
		&row.ID, &row.UserID, &row.ActorID, &row.ActorName, &row.Kind,
		&row.WorkspaceID, &row.ChannelID, &row.ChannelName, &row.IsDM, &row.MessageID,
		&row.Body, &row.ReadAt, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return NotificationRow{}, ErrNotFound
	}
	if err != nil {
		return NotificationRow{}, fmt.Errorf("get notification: %w", err)
	}
	return row, nil
}

func (s *Store) ListWorkspaceMembers(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceMemberInfo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+workspaceMemberSelectSQL+`
		FROM workspace_members wm
		INNER JOIN users u ON u.id = wm.user_id
		LEFT JOIN workspaces hw ON hw.id = wm.home_workspace_id
		LEFT JOIN federated_peers fp ON fp.server_id = wm.home_server_id
		WHERE wm.workspace_id = $1
		ORDER BY lower(wm.handle)
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	defer rows.Close()

	var out []WorkspaceMemberInfo
	index := map[uuid.UUID]int{}
	for rows.Next() {
		var m WorkspaceMemberInfo
		if err := rows.Scan(scanWorkspaceMember(&m)...); err != nil {
			return nil, fmt.Errorf("scan workspace member: %w", err)
		}
		m.StatusEmoji, m.StatusText, m.StatusExpiresAt = domain.EffectiveCustomStatus(
			m.StatusEmoji, m.StatusText, m.StatusExpiresAt,
		)
		index[m.UserID] = len(out)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	aliases, err := s.listHandleAliases(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for userID, handles := range aliases {
		i, ok := index[userID]
		if !ok {
			continue
		}
		out[i].FormerHandles = handles
	}
	return out, nil
}

func (s *Store) GetWorkspaceMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (WorkspaceMemberInfo, error) {
	var m WorkspaceMemberInfo
	err := s.pool.QueryRow(ctx, `
		SELECT `+workspaceMemberSelectSQL+`
		FROM workspace_members wm
		INNER JOIN users u ON u.id = wm.user_id
		LEFT JOIN workspaces hw ON hw.id = wm.home_workspace_id
		LEFT JOIN federated_peers fp ON fp.server_id = wm.home_server_id
		WHERE wm.workspace_id = $1 AND wm.user_id = $2
	`, workspaceID, userID).Scan(scanWorkspaceMember(&m)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceMemberInfo{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("get workspace member: %w", err)
	}
	m.StatusEmoji, m.StatusText, m.StatusExpiresAt = domain.EffectiveCustomStatus(
		m.StatusEmoji, m.StatusText, m.StatusExpiresAt,
	)
	aliases, err := s.listHandleAliasesForUser(ctx, workspaceID, userID)
	if err != nil {
		return WorkspaceMemberInfo{}, err
	}
	m.FormerHandles = aliases
	return m, nil
}

func (s *Store) listHandleAliases(
	ctx context.Context,
	workspaceID uuid.UUID,
) (map[uuid.UUID][]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, handle
		FROM workspace_member_handle_aliases
		WHERE workspace_id = $1
		ORDER BY created_at ASC, handle ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list handle aliases: %w", err)
	}
	defer rows.Close()

	out := map[uuid.UUID][]string{}
	for rows.Next() {
		var userID uuid.UUID
		var handle string
		if err := rows.Scan(&userID, &handle); err != nil {
			return nil, fmt.Errorf("scan handle alias: %w", err)
		}
		out[userID] = append(out[userID], handle)
	}
	return out, rows.Err()
}

func (s *Store) listHandleAliasesForUser(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT handle
		FROM workspace_member_handle_aliases
		WHERE workspace_id = $1 AND user_id = $2
		ORDER BY created_at ASC, handle ASC
	`, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list user handle aliases: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var handle string
		if err := rows.Scan(&handle); err != nil {
			return nil, fmt.Errorf("scan user handle alias: %w", err)
		}
		out = append(out, handle)
	}
	return out, rows.Err()
}

func (s *Store) MemberHandleExists(
	ctx context.Context,
	workspaceID uuid.UUID,
	handle string,
	excludeUserID uuid.UUID,
) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspace_members
			WHERE workspace_id = $1
			  AND handle = $2
			  AND user_id <> $3
		)
	`, workspaceID, handle, excludeUserID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("member handle exists: %w", err)
	}
	return exists, nil
}

func (s *Store) UpdateWorkspaceMemberHandle(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	handle string,
) (WorkspaceMemberInfo, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("begin update handle: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldHandle string
	var role string
	var displayName string
	var statusEmoji string
	var statusText string
	var statusExpiresAt *time.Time
	var hasAvatar bool
	var avatarUpdatedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT wm.handle, wm.role, u.display_name,
			COALESCE(u.status_emoji, ''), COALESCE(u.status_text, ''), u.status_expires_at,
			(u.avatar_bytes IS NOT NULL), u.avatar_updated_at
		FROM workspace_members wm
		INNER JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND wm.user_id = $2
	`, workspaceID, userID).Scan(
		&oldHandle, &role, &displayName, &statusEmoji, &statusText, &statusExpiresAt,
		&hasAvatar, &avatarUpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceMemberInfo{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("load workspace member handle: %w", err)
	}
	statusEmoji, statusText, statusExpiresAt = domain.EffectiveCustomStatus(
		statusEmoji, statusText, statusExpiresAt,
	)
	if oldHandle == handle {
		aliases, err := s.listHandleAliasesForUser(ctx, workspaceID, userID)
		if err != nil {
			return WorkspaceMemberInfo{}, err
		}
		return WorkspaceMemberInfo{
			UserID: userID, DisplayName: displayName, Handle: handle,
			FormerHandles: aliases, Role: role,
			StatusEmoji: statusEmoji, StatusText: statusText, StatusExpiresAt: statusExpiresAt,
			HasAvatar: hasAvatar, AvatarUpdatedAt: avatarUpdatedAt,
		}, nil
	}

	// Reclaim the new handle if it only lives as someone else's former alias.
	if _, err := tx.Exec(ctx, `
		DELETE FROM workspace_member_handle_aliases
		WHERE workspace_id = $1 AND handle = $2
	`, workspaceID, handle); err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("reclaim handle alias: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE workspace_members
		SET handle = $3
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID, handle)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceMemberInfo{}, ErrHandleConflict
		}
		return WorkspaceMemberInfo{}, fmt.Errorf("update workspace member handle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return WorkspaceMemberInfo{}, ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_member_handle_aliases (workspace_id, user_id, handle)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, handle) DO UPDATE
		SET user_id = EXCLUDED.user_id
	`, workspaceID, userID, oldHandle); err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("save handle alias: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("commit update handle: %w", err)
	}

	aliases, err := s.listHandleAliasesForUser(ctx, workspaceID, userID)
	if err != nil {
		return WorkspaceMemberInfo{}, err
	}
	return WorkspaceMemberInfo{
		UserID: userID, DisplayName: displayName, Handle: handle,
		FormerHandles: aliases, Role: role,
		StatusEmoji: statusEmoji, StatusText: statusText, StatusExpiresAt: statusExpiresAt,
		HasAvatar: hasAvatar, AvatarUpdatedAt: avatarUpdatedAt,
	}, nil
}

func (s *Store) GetChannelNotificationLevel(
	ctx context.Context,
	userID, channelID uuid.UUID,
) (domain.ChannelNotificationLevel, error) {
	var level string
	err := s.pool.QueryRow(ctx, `
		SELECT level FROM channel_notification_settings
		WHERE user_id = $1 AND channel_id = $2
	`, userID, channelID).Scan(&level)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ChannelNotifyMentions, nil
	}
	if err != nil {
		return "", fmt.Errorf("get channel notification level: %w", err)
	}
	parsed, ok := domain.ParseChannelNotificationLevel(level)
	if !ok {
		return domain.ChannelNotifyMentions, nil
	}
	return parsed, nil
}

func (s *Store) UpsertChannelNotificationLevel(
	ctx context.Context,
	userID, channelID uuid.UUID,
	level domain.ChannelNotificationLevel,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channel_notification_settings (user_id, channel_id, level, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id, channel_id) DO UPDATE
		SET level = EXCLUDED.level, updated_at = now()
	`, userID, channelID, string(level))
	if err != nil {
		return fmt.Errorf("upsert channel notification level: %w", err)
	}
	return nil
}

func (s *Store) ListChannelNotificationLevels(
	ctx context.Context,
	channelID uuid.UUID,
) (map[uuid.UUID]domain.ChannelNotificationLevel, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, level FROM channel_notification_settings WHERE channel_id = $1
	`, channelID)
	if err != nil {
		return nil, fmt.Errorf("list channel notification levels: %w", err)
	}
	defer rows.Close()

	out := map[uuid.UUID]domain.ChannelNotificationLevel{}
	for rows.Next() {
		var userID uuid.UUID
		var level string
		if err := rows.Scan(&userID, &level); err != nil {
			return nil, fmt.Errorf("scan channel notification level: %w", err)
		}
		parsed, ok := domain.ParseChannelNotificationLevel(level)
		if !ok {
			continue
		}
		out[userID] = parsed
	}
	return out, rows.Err()
}

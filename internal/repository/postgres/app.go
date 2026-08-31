package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

var ErrConflict = errors.New("already exists")

type AppRow struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	Description  string
	Origin       string
	OwnerUserID  *uuid.UUID
	Capabilities []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r AppRow) ToDomain() domain.App {
	out := domain.App{
		ID: r.ID.String(), Slug: r.Slug, Name: r.Name, Description: r.Description,
		Origin: r.Origin, Capabilities: r.Capabilities,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.OwnerUserID != nil {
		out.OwnerUserID = r.OwnerUserID.String()
	}
	if out.Capabilities == nil {
		out.Capabilities = []string{}
	}
	return out
}

type WorkspaceAppInstallRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	AppSlug     string
	AppName     string
	InstalledBy *uuid.UUID
	BotUserID   uuid.UUID
	CreatedAt   time.Time
}

func (r WorkspaceAppInstallRow) ToDomain() domain.WorkspaceAppInstall {
	out := domain.WorkspaceAppInstall{
		ID: r.ID.String(), WorkspaceID: r.WorkspaceID.String(), AppID: r.AppID.String(),
		AppSlug: r.AppSlug, AppName: r.AppName, BotUserID: r.BotUserID.String(),
		CreatedAt: r.CreatedAt,
	}
	if r.InstalledBy != nil {
		out.InstalledBy = r.InstalledBy.String()
	}
	return out
}

type ChannelAppInstallRow struct {
	ID                    uuid.UUID
	WorkspaceAppInstallID uuid.UUID
	ChannelID             uuid.UUID
	ChannelName           string
	ChannelIsPrivate      bool
	ChannelIsDM           bool
	CreatedAt             time.Time
}

type IncomingWebhookRow struct {
	ID                    uuid.UUID
	ChannelAppInstallID   uuid.UUID
	WorkspaceAppInstallID uuid.UUID
	ChannelID             uuid.UUID
	ChannelName           string
	ChannelIsPrivate      bool
	TokenHash             string
	TokenPrefix           string
	LastUsedAt            *time.Time
	CreatedAt             time.Time
}

func (r IncomingWebhookRow) ToDomain() domain.IncomingWebhook {
	return domain.IncomingWebhook{
		ID: r.ID.String(), ChannelID: r.ChannelID.String(),
		ChannelName: r.ChannelName, ChannelIsPrivate: r.ChannelIsPrivate,
		TokenPrefix: r.TokenPrefix, LastUsedAt: r.LastUsedAt, CreatedAt: r.CreatedAt,
	}
}

func (s *Store) ListApps(ctx context.Context) ([]AppRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, description, origin, owner_user_id, capabilities, created_at, updated_at
		FROM apps
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}
	defer rows.Close()
	var out []AppRow
	for rows.Next() {
		row, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetAppBySlug(ctx context.Context, slug string) (AppRow, error) {
	row, err := scanApp(s.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, origin, owner_user_id, capabilities, created_at, updated_at
		FROM apps WHERE slug = $1
	`, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return AppRow{}, ErrNotFound
	}
	if err != nil {
		return AppRow{}, fmt.Errorf("get app: %w", err)
	}
	return row, nil
}

func (s *Store) CountWorkspaceAppInstalls(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM workspace_app_installs WHERE workspace_id = $1
	`, workspaceID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count workspace apps: %w", err)
	}
	return n, nil
}

func (s *Store) GetWorkspaceAppInstall(
	ctx context.Context, workspaceID, appID uuid.UUID,
) (WorkspaceAppInstallRow, error) {
	var row WorkspaceAppInstallRow
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.workspace_id, i.app_id, a.slug, a.name, i.installed_by, i.bot_user_id, i.created_at
		FROM workspace_app_installs i
		INNER JOIN apps a ON a.id = i.app_id
		WHERE i.workspace_id = $1 AND i.app_id = $2
	`, workspaceID, appID).Scan(
		&row.ID, &row.WorkspaceID, &row.AppID, &row.AppSlug, &row.AppName,
		&row.InstalledBy, &row.BotUserID, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceAppInstallRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceAppInstallRow{}, fmt.Errorf("get workspace app install: %w", err)
	}
	return row, nil
}

func (s *Store) ListWorkspaceAppInstalls(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceAppInstallRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.workspace_id, i.app_id, a.slug, a.name, i.installed_by, i.bot_user_id, i.created_at
		FROM workspace_app_installs i
		INNER JOIN apps a ON a.id = i.app_id
		WHERE i.workspace_id = $1
		ORDER BY a.name ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace app installs: %w", err)
	}
	defer rows.Close()
	var out []WorkspaceAppInstallRow
	for rows.Next() {
		var row WorkspaceAppInstallRow
		if err := rows.Scan(
			&row.ID, &row.WorkspaceID, &row.AppID, &row.AppSlug, &row.AppName,
			&row.InstalledBy, &row.BotUserID, &row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace app install: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) CreateAppUser(ctx context.Context, displayName string) (UserRow, error) {
	var row UserRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (display_name, kind, email_verified)
		VALUES ($1, 'app', true)
		RETURNING `+userSelectColumns+`
	`, displayName).Scan(scanUserFields(&row)...)
	if err != nil {
		return UserRow{}, fmt.Errorf("create app user: %w", err)
	}
	return scanUserRow(&row, nil)
}

func (s *Store) AddAppWorkspaceMember(
	ctx context.Context, workspaceID, userID uuid.UUID,
) error {
	var displayName string
	err := s.pool.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, userID).Scan(&displayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load app user: %w", err)
	}
	for attempt := 0; attempt < 5; attempt++ {
		handle, err := allocateUniqueHandle(ctx, s.pool, workspaceID, displayName)
		if err != nil {
			return fmt.Errorf("allocate app handle: %w", err)
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role, handle, kind)
			VALUES ($1, $2, 'member', $3, 'app')
			ON CONFLICT (workspace_id, user_id) DO NOTHING
		`, workspaceID, userID, handle)
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return fmt.Errorf("add app workspace member: %w", err)
	}
	return fmt.Errorf("add app workspace member: handle conflict")
}

func (s *Store) InsertWorkspaceAppInstall(
	ctx context.Context, workspaceID, appID, installedBy, botUserID uuid.UUID,
) (WorkspaceAppInstallRow, error) {
	var row WorkspaceAppInstallRow
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO workspace_app_installs (workspace_id, app_id, installed_by, bot_user_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id, workspace_id, app_id, installed_by, bot_user_id, created_at
		)
		SELECT i.id, i.workspace_id, i.app_id, a.slug, a.name, i.installed_by, i.bot_user_id, i.created_at
		FROM inserted i
		INNER JOIN apps a ON a.id = i.app_id
	`, workspaceID, appID, installedBy, botUserID).Scan(
		&row.ID, &row.WorkspaceID, &row.AppID, &row.AppSlug, &row.AppName,
		&row.InstalledBy, &row.BotUserID, &row.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceAppInstallRow{}, ErrConflict
		}
		return WorkspaceAppInstallRow{}, fmt.Errorf("insert workspace app install: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteWorkspaceAppInstall(ctx context.Context, workspaceID, appID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM workspace_app_installs WHERE workspace_id = $1 AND app_id = $2
	`, workspaceID, appID)
	if err != nil {
		return fmt.Errorf("delete workspace app install: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertChannelAppInstall(
	ctx context.Context, workspaceInstallID, channelID uuid.UUID,
) (ChannelAppInstallRow, error) {
	var row ChannelAppInstallRow
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO channel_app_installs (workspace_app_install_id, channel_id)
			VALUES ($1, $2)
			RETURNING id, workspace_app_install_id, channel_id, created_at
		)
		SELECT i.id, i.workspace_app_install_id, i.channel_id, c.name, c.is_private,
			(c.kind = 'dm'), i.created_at
		FROM inserted i
		INNER JOIN channels c ON c.id = i.channel_id
	`, workspaceInstallID, channelID).Scan(
		&row.ID, &row.WorkspaceAppInstallID, &row.ChannelID, &row.ChannelName,
		&row.ChannelIsPrivate, &row.ChannelIsDM, &row.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ChannelAppInstallRow{}, ErrConflict
		}
		return ChannelAppInstallRow{}, fmt.Errorf("insert channel app install: %w", err)
	}
	return row, nil
}

func (s *Store) GetChannelAppInstallForApp(
	ctx context.Context, workspaceID, channelID, appID uuid.UUID,
) (ChannelAppInstallRow, error) {
	var row ChannelAppInstallRow
	err := s.pool.QueryRow(ctx, `
		SELECT cai.id, cai.workspace_app_install_id, cai.channel_id, c.name, c.is_private,
			(c.kind = 'dm'), cai.created_at
		FROM channel_app_installs cai
		INNER JOIN workspace_app_installs wai ON wai.id = cai.workspace_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
		WHERE wai.workspace_id = $1 AND cai.channel_id = $2 AND wai.app_id = $3
	`, workspaceID, channelID, appID).Scan(
		&row.ID, &row.WorkspaceAppInstallID, &row.ChannelID, &row.ChannelName,
		&row.ChannelIsPrivate, &row.ChannelIsDM, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChannelAppInstallRow{}, ErrNotFound
	}
	if err != nil {
		return ChannelAppInstallRow{}, fmt.Errorf("get channel app install: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteChannelAppInstall(ctx context.Context, installID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM channel_app_installs WHERE id = $1`, installID)
	if err != nil {
		return fmt.Errorf("delete channel app install: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertIncomingWebhook(
	ctx context.Context, channelInstallID uuid.UUID, tokenHash, tokenPrefix string,
) (IncomingWebhookRow, error) {
	var row IncomingWebhookRow
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO incoming_webhooks (channel_app_install_id, token_hash, token_prefix)
			VALUES ($1, $2, $3)
			RETURNING id, channel_app_install_id, token_hash, token_prefix, last_used_at, created_at
		)
		SELECT` + incomingWebhookSelectSQL + `
		FROM inserted i
		INNER JOIN channel_app_installs cai ON cai.id = i.channel_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
	`, channelInstallID, tokenHash, tokenPrefix).Scan(scanIncomingWebhookFields(&row)...)
	if err != nil {
		return IncomingWebhookRow{}, fmt.Errorf("insert incoming webhook: %w", err)
	}
	return row, nil
}

func (s *Store) GetIncomingWebhookByChannelInstall(
	ctx context.Context, channelInstallID uuid.UUID,
) (IncomingWebhookRow, error) {
	var row IncomingWebhookRow
	err := s.pool.QueryRow(ctx, `
		SELECT` + incomingWebhookSelectSQL + `
		FROM incoming_webhooks i
		INNER JOIN channel_app_installs cai ON cai.id = i.channel_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
		WHERE i.channel_app_install_id = $1
	`, channelInstallID).Scan(scanIncomingWebhookFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return IncomingWebhookRow{}, ErrNotFound
	}
	if err != nil {
		return IncomingWebhookRow{}, fmt.Errorf("get incoming webhook: %w", err)
	}
	return row, nil
}

func (s *Store) ListIncomingWebhooksForWorkspace(
	ctx context.Context, workspaceID uuid.UUID,
) ([]IncomingWebhookRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT` + incomingWebhookSelectSQL + `
		FROM incoming_webhooks i
		INNER JOIN channel_app_installs cai ON cai.id = i.channel_app_install_id
		INNER JOIN workspace_app_installs wai ON wai.id = cai.workspace_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
		WHERE wai.workspace_id = $1
		ORDER BY lower(c.name)
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list incoming webhooks: %w", err)
	}
	defer rows.Close()
	var out []IncomingWebhookRow
	for rows.Next() {
		var row IncomingWebhookRow
		if err := rows.Scan(scanIncomingWebhookFields(&row)...); err != nil {
			return nil, fmt.Errorf("scan incoming webhook: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetIncomingWebhookByTokenHash(ctx context.Context, tokenHash string) (IncomingWebhookRow, error) {
	var row IncomingWebhookRow
	err := s.pool.QueryRow(ctx, `
		SELECT` + incomingWebhookSelectSQL + `
		FROM incoming_webhooks i
		INNER JOIN channel_app_installs cai ON cai.id = i.channel_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
		WHERE i.token_hash = $1
	`, tokenHash).Scan(scanIncomingWebhookFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return IncomingWebhookRow{}, ErrNotFound
	}
	if err != nil {
		return IncomingWebhookRow{}, fmt.Errorf("get incoming webhook by hash: %w", err)
	}
	return row, nil
}

func (s *Store) GetIncomingWebhookBot(
	ctx context.Context, webhookID uuid.UUID,
) (uuid.UUID, uuid.UUID, error) {
	var botID, workspaceID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT wai.bot_user_id, wai.workspace_id
		FROM incoming_webhooks i
		INNER JOIN channel_app_installs cai ON cai.id = i.channel_app_install_id
		INNER JOIN workspace_app_installs wai ON wai.id = cai.workspace_app_install_id
		WHERE i.id = $1
	`, webhookID).Scan(&botID, &workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get webhook bot: %w", err)
	}
	return botID, workspaceID, nil
}

func (s *Store) RotateIncomingWebhook(
	ctx context.Context, webhookID uuid.UUID, tokenHash, tokenPrefix string,
) (IncomingWebhookRow, error) {
	var row IncomingWebhookRow
	err := s.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE incoming_webhooks
			SET token_hash = $2, token_prefix = $3, updated_at = now()
			WHERE id = $1
			RETURNING id, channel_app_install_id, token_hash, token_prefix, last_used_at, created_at
		)
		SELECT u.id, u.channel_app_install_id, cai.workspace_app_install_id, cai.channel_id, c.name, c.is_private,
			u.token_hash, u.token_prefix, u.last_used_at, u.created_at
		FROM updated u
		INNER JOIN channel_app_installs cai ON cai.id = u.channel_app_install_id
		INNER JOIN channels c ON c.id = cai.channel_id
	`, webhookID, tokenHash, tokenPrefix).Scan(scanIncomingWebhookFields(&row)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return IncomingWebhookRow{}, ErrNotFound
	}
	if err != nil {
		return IncomingWebhookRow{}, fmt.Errorf("rotate incoming webhook: %w", err)
	}
	return row, nil
}

func (s *Store) TouchIncomingWebhook(ctx context.Context, webhookID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE incoming_webhooks SET last_used_at = now(), updated_at = now() WHERE id = $1
	`, webhookID)
	if err != nil {
		return fmt.Errorf("touch incoming webhook: %w", err)
	}
	return nil
}

const incomingWebhookSelectSQL = `
		i.id, i.channel_app_install_id, cai.workspace_app_install_id, cai.channel_id, c.name, c.is_private,
			i.token_hash, i.token_prefix, i.last_used_at, i.created_at
`

func scanIncomingWebhookFields(row *IncomingWebhookRow) []any {
	return []any{
		&row.ID, &row.ChannelAppInstallID, &row.WorkspaceAppInstallID, &row.ChannelID,
		&row.ChannelName, &row.ChannelIsPrivate, &row.TokenHash, &row.TokenPrefix,
		&row.LastUsedAt, &row.CreatedAt,
	}
}

func scanApp(row pgx.Row) (AppRow, error) {
	var r AppRow
	var caps []byte
	err := row.Scan(
		&r.ID, &r.Slug, &r.Name, &r.Description, &r.Origin, &r.OwnerUserID, &caps, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return AppRow{}, err
	}
	if len(caps) > 0 {
		if err := json.Unmarshal(caps, &r.Capabilities); err != nil {
			return AppRow{}, fmt.Errorf("scan app capabilities: %w", err)
		}
	}
	return r, nil
}

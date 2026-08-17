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

var ErrInviteConflict = errors.New("invite already pending")

type WorkspaceInviteRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Token       string
	Role        string
	InvitedBy   uuid.UUID
	ExpiresAt   time.Time
	AcceptedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r WorkspaceInviteRow) ToDomain() domain.WorkspaceInvite {
	return domain.WorkspaceInvite{
		ID:          r.ID.String(),
		WorkspaceID: r.WorkspaceID.String(),
		Email:       r.Email,
		Token:       r.Token,
		Role:        domain.WorkspaceRole(r.Role),
		InvitedBy:   r.InvitedBy.String(),
		ExpiresAt:   r.ExpiresAt,
		AcceptedAt:  r.AcceptedAt,
		CreatedAt:   r.CreatedAt,
	}
}

func (s *Store) CreateWorkspaceInvite(
	ctx context.Context,
	workspaceID, invitedBy uuid.UUID,
	email, token string,
	role domain.WorkspaceRole,
	expiresAt time.Time,
) (WorkspaceInviteRow, error) {
	var row WorkspaceInviteRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_invites (workspace_id, email, token, role, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, workspace_id, email, token, role, invited_by, expires_at, accepted_at, created_at, updated_at
	`, workspaceID, email, token, role, invitedBy, expiresAt).Scan(
		&row.ID, &row.WorkspaceID, &row.Email, &row.Token, &row.Role, &row.InvitedBy,
		&row.ExpiresAt, &row.AcceptedAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceInviteRow{}, ErrInviteConflict
		}
		return WorkspaceInviteRow{}, fmt.Errorf("create invite: %w", err)
	}
	return row, nil
}

func (s *Store) ListPendingWorkspaceInvites(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceInviteRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, email, token, role, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM workspace_invites
		WHERE workspace_id = $1 AND accepted_at IS NULL
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var out []WorkspaceInviteRow
	for rows.Next() {
		var row WorkspaceInviteRow
		if err := rows.Scan(
			&row.ID, &row.WorkspaceID, &row.Email, &row.Token, &row.Role, &row.InvitedBy,
			&row.ExpiresAt, &row.AcceptedAt, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspaceInviteByToken(ctx context.Context, token string) (WorkspaceInviteRow, error) {
	var row WorkspaceInviteRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, email, token, role, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM workspace_invites WHERE token = $1
	`, token).Scan(
		&row.ID, &row.WorkspaceID, &row.Email, &row.Token, &row.Role, &row.InvitedBy,
		&row.ExpiresAt, &row.AcceptedAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceInviteRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceInviteRow{}, fmt.Errorf("get invite: %w", err)
	}
	return row, nil
}

func (s *Store) AcceptWorkspaceInvite(ctx context.Context, inviteID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE workspace_invites
		SET accepted_at = now(), updated_at = now()
		WHERE id = $1 AND accepted_at IS NULL
	`, inviteID)
	if err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

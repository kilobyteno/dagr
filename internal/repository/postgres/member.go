package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
)

func (s *Store) CountWorkspaceOwners(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_members
		WHERE workspace_id = $1 AND role = 'owner'
	`, workspaceID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count workspace owners: %w", err)
	}
	return n, nil
}

func (s *Store) UpdateWorkspaceMemberRole(
	ctx context.Context, workspaceID, userID uuid.UUID, role domain.WorkspaceRole,
) (WorkspaceMemberInfo, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE workspace_members SET role = $3
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID, role)
	if err != nil {
		return WorkspaceMemberInfo{}, fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return WorkspaceMemberInfo{}, ErrNotFound
	}
	return s.GetWorkspaceMember(ctx, workspaceID, userID)
}

func (s *Store) RemoveWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) TransferWorkspaceOwnership(
	ctx context.Context, workspaceID, fromOwner, toUser uuid.UUID,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transfer ownership: %w", err)
	}
	defer tx.Rollback(ctx)

	var fromRole, toRole string
	err = tx.QueryRow(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, fromOwner).Scan(&fromRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load current owner: %w", err)
	}
	if fromRole != string(domain.WorkspaceRoleOwner) {
		return ErrNotFound
	}
	err = tx.QueryRow(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, toUser).Scan(&toRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load new owner: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE workspace_members SET role = 'member'
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, fromOwner); err != nil {
		return fmt.Errorf("demote previous owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE workspace_members SET role = 'owner'
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, toUser); err != nil {
		return fmt.Errorf("promote new owner: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transfer ownership: %w", err)
	}
	return nil
}

func (s *Store) DeleteWorkspaceInvite(ctx context.Context, workspaceID, inviteID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM workspace_invites
		WHERE id = $1 AND workspace_id = $2 AND accepted_at IS NULL
	`, inviteID, workspaceID)
	if err != nil {
		return fmt.Errorf("delete workspace invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) FindHomeWorkspaceForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, string, error) {
	var id uuid.UUID
	var name string
	err := s.pool.QueryRow(ctx, `
		SELECT w.id, w.name
		FROM workspace_members wm
		INNER JOIN workspaces w ON w.id = wm.workspace_id
		WHERE wm.user_id = $1 AND wm.role = 'owner' AND wm.kind = 'member'
		ORDER BY wm.created_at ASC
		LIMIT 1
	`, userID).Scan(&id, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrNotFound
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("find home workspace: %w", err)
	}
	return id, name, nil
}

func (s *Store) SetWorkspaceMemberOrigin(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	kind string,
	homeWorkspaceID *uuid.UUID,
	homeWorkspaceName string,
	homeServerID, homeWorkspaceRemoteID, homeWorkspaceIconURL string,
) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workspace_members
		SET kind = $3,
			home_workspace_id = $4,
			home_workspace_name = NULLIF($5, ''),
			home_server_id = NULLIF($6, ''),
			home_workspace_remote_id = NULLIF($7, ''),
			home_workspace_icon_url = NULLIF($8, '')
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID, kind, homeWorkspaceID, homeWorkspaceName,
		homeServerID, homeWorkspaceRemoteID, homeWorkspaceIconURL)
	if err != nil {
		return fmt.Errorf("set member origin: %w", err)
	}
	return nil
}

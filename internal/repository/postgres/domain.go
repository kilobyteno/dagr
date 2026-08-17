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

var (
	ErrDomainConflict         = errors.New("domain already claimed in this workspace")
	ErrDomainVerifiedConflict = errors.New("domain already verified by another workspace")
)

type WorkspaceDomainRow struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	Domain            string
	VerificationToken string
	VerifiedAt        *time.Time
	AutoJoin          bool
	CreatedBy         uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r WorkspaceDomainRow) ToDomain() domain.WorkspaceDomain {
	return domain.WorkspaceDomain{
		ID:                r.ID.String(),
		WorkspaceID:       r.WorkspaceID.String(),
		Domain:            r.Domain,
		VerificationToken: r.VerificationToken,
		VerifiedAt:        r.VerifiedAt,
		AutoJoin:          r.AutoJoin,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (s *Store) CreateWorkspaceDomain(
	ctx context.Context,
	workspaceID, createdBy uuid.UUID,
	domainName, verificationToken string,
) (WorkspaceDomainRow, error) {
	var row WorkspaceDomainRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_domains (workspace_id, domain, verification_token, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workspace_id, domain, verification_token, verified_at, auto_join,
			created_by, created_at, updated_at
	`, workspaceID, domainName, verificationToken, createdBy).Scan(
		&row.ID, &row.WorkspaceID, &row.Domain, &row.VerificationToken, &row.VerifiedAt,
		&row.AutoJoin, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceDomainRow{}, ErrDomainConflict
		}
		return WorkspaceDomainRow{}, fmt.Errorf("insert workspace domain: %w", err)
	}
	return row, nil
}

func (s *Store) ListWorkspaceDomains(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceDomainRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, domain, verification_token, verified_at, auto_join,
			created_by, created_at, updated_at
		FROM workspace_domains
		WHERE workspace_id = $1
		ORDER BY domain ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace domains: %w", err)
	}
	defer rows.Close()

	var out []WorkspaceDomainRow
	for rows.Next() {
		var row WorkspaceDomainRow
		if err := rows.Scan(
			&row.ID, &row.WorkspaceID, &row.Domain, &row.VerificationToken, &row.VerifiedAt,
			&row.AutoJoin, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace domain: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) GetWorkspaceDomain(
	ctx context.Context,
	workspaceID, domainID uuid.UUID,
) (WorkspaceDomainRow, error) {
	var row WorkspaceDomainRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, domain, verification_token, verified_at, auto_join,
			created_by, created_at, updated_at
		FROM workspace_domains
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, domainID).Scan(
		&row.ID, &row.WorkspaceID, &row.Domain, &row.VerificationToken, &row.VerifiedAt,
		&row.AutoJoin, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceDomainRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceDomainRow{}, fmt.Errorf("get workspace domain: %w", err)
	}
	return row, nil
}

func (s *Store) MarkWorkspaceDomainVerified(
	ctx context.Context,
	workspaceID, domainID uuid.UUID,
) (WorkspaceDomainRow, error) {
	var row WorkspaceDomainRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspace_domains
		SET verified_at = now(), updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING id, workspace_id, domain, verification_token, verified_at, auto_join,
			created_by, created_at, updated_at
	`, workspaceID, domainID).Scan(
		&row.ID, &row.WorkspaceID, &row.Domain, &row.VerificationToken, &row.VerifiedAt,
		&row.AutoJoin, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceDomainRow{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return WorkspaceDomainRow{}, ErrDomainVerifiedConflict
		}
		return WorkspaceDomainRow{}, fmt.Errorf("mark domain verified: %w", err)
	}
	return row, nil
}

func (s *Store) UpdateWorkspaceDomainAutoJoin(
	ctx context.Context,
	workspaceID, domainID uuid.UUID,
	autoJoin bool,
) (WorkspaceDomainRow, error) {
	var row WorkspaceDomainRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspace_domains
		SET auto_join = $3, updated_at = now()
		WHERE workspace_id = $1 AND id = $2 AND verified_at IS NOT NULL
		RETURNING id, workspace_id, domain, verification_token, verified_at, auto_join,
			created_by, created_at, updated_at
	`, workspaceID, domainID, autoJoin).Scan(
		&row.ID, &row.WorkspaceID, &row.Domain, &row.VerificationToken, &row.VerifiedAt,
		&row.AutoJoin, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkspaceDomainRow{}, ErrNotFound
	}
	if err != nil {
		return WorkspaceDomainRow{}, fmt.Errorf("update domain auto join: %w", err)
	}
	return row, nil
}

func (s *Store) DeleteWorkspaceDomain(ctx context.Context, workspaceID, domainID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM workspace_domains WHERE workspace_id = $1 AND id = $2
	`, workspaceID, domainID)
	if err != nil {
		return fmt.Errorf("delete workspace domain: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DomainVerifiedElsewhere(
	ctx context.Context,
	domainName string,
	excludeWorkspaceID uuid.UUID,
) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspace_domains
			WHERE domain = $1 AND verified_at IS NOT NULL AND workspace_id <> $2
		)
	`, domainName, excludeWorkspaceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("domain verified elsewhere: %w", err)
	}
	return exists, nil
}

func (s *Store) ListAutoJoinWorkspaceIDsByDomain(ctx context.Context, domainName string) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT workspace_id
		FROM workspace_domains
		WHERE domain = $1 AND verified_at IS NOT NULL AND auto_join = true
	`, domainName)
	if err != nil {
		return nil, fmt.Errorf("list auto join workspaces: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan auto join workspace: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) AddWorkspaceMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
	role domain.WorkspaceRole,
) error {
	var displayName string
	err := s.pool.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, userID).Scan(&displayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load user for membership: %w", err)
	}
	for attempt := 0; attempt < 5; attempt++ {
		handle, err := allocateUniqueHandle(ctx, s.pool, workspaceID, displayName)
		if err != nil {
			return fmt.Errorf("allocate member handle: %w", err)
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role, handle)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (workspace_id, user_id) DO NOTHING
		`, workspaceID, userID, role, handle)
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return fmt.Errorf("add workspace member: %w", err)
	}
	return fmt.Errorf("add workspace member: handle conflict")
}

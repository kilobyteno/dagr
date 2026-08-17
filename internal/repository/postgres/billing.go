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

// BillingRow is a workspace_billing record.
type BillingRow struct {
	WorkspaceID          uuid.UUID
	Plan                 string
	Status               string
	Interval             *string
	BillableSeats        int
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
	EarlyAccessEndsAt    *time.Time
	MollieCustomerID     *string
	MollieMandateID      *string
	MollieSubscriptionID *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (r BillingRow) ToDomain() domain.WorkspaceBilling {
	out := domain.WorkspaceBilling{
		WorkspaceID:       r.WorkspaceID.String(),
		Plan:              domain.PlanID(r.Plan),
		Status:            domain.BillingStatus(r.Status),
		BillableSeats:     r.BillableSeats,
		CurrentPeriodEnd:  r.CurrentPeriodEnd,
		CancelAtPeriodEnd: r.CancelAtPeriodEnd,
		EarlyAccessEndsAt: r.EarlyAccessEndsAt,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
	if r.Interval != nil {
		out.Interval = domain.BillingInterval(*r.Interval)
	}
	if r.MollieCustomerID != nil {
		out.MollieCustomerID = *r.MollieCustomerID
	}
	if r.MollieMandateID != nil {
		out.MollieMandateID = *r.MollieMandateID
	}
	if r.MollieSubscriptionID != nil {
		out.MollieSubscriptionID = *r.MollieSubscriptionID
	}
	return out
}

const billingSelectColumns = `
	workspace_id, plan, status, billing_interval, billable_seats,
	current_period_end, cancel_at_period_end, early_access_ends_at,
	mollie_customer_id, mollie_mandate_id, mollie_subscription_id,
	created_at, updated_at
`

func scanBilling(row BillingRow, err error) (BillingRow, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingRow{}, ErrNotFound
	}
	if err != nil {
		return BillingRow{}, err
	}
	return row, nil
}

func (s *Store) EnsureWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID, seats int) (BillingRow, error) {
	if seats < 1 {
		seats = 1
	}
	var row BillingRow
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_billing (workspace_id, plan, status, billable_seats)
		VALUES ($1, 'free', 'active', $2)
		ON CONFLICT (workspace_id) DO UPDATE
			SET updated_at = workspace_billing.updated_at
		RETURNING `+billingSelectColumns+`
	`, workspaceID, seats).Scan(
		&row.WorkspaceID, &row.Plan, &row.Status, &row.Interval, &row.BillableSeats,
		&row.CurrentPeriodEnd, &row.CancelAtPeriodEnd, &row.EarlyAccessEndsAt,
		&row.MollieCustomerID, &row.MollieMandateID, &row.MollieSubscriptionID,
		&row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return BillingRow{}, fmt.Errorf("ensure workspace billing: %w", err)
	}
	return row, nil
}

func (s *Store) GetWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID) (BillingRow, error) {
	var row BillingRow
	err := s.pool.QueryRow(ctx, `
		SELECT `+billingSelectColumns+`
		FROM workspace_billing
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&row.WorkspaceID, &row.Plan, &row.Status, &row.Interval, &row.BillableSeats,
		&row.CurrentPeriodEnd, &row.CancelAtPeriodEnd, &row.EarlyAccessEndsAt,
		&row.MollieCustomerID, &row.MollieMandateID, &row.MollieSubscriptionID,
		&row.CreatedAt, &row.UpdatedAt,
	)
	out, scanErr := scanBilling(row, err)
	if scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return BillingRow{}, ErrNotFound
		}
		return BillingRow{}, fmt.Errorf("get workspace billing: %w", scanErr)
	}
	return out, nil
}

func (s *Store) CountBillableSeats(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM workspace_members
		WHERE workspace_id = $1 AND kind = 'member'
	`, workspaceID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count billable seats: %w", err)
	}
	if n < 1 {
		n = 1
	}
	return n, nil
}

// BillingPatch is a partial update for workspace_billing.
type BillingPatch struct {
	Plan                 *string
	Status               *string
	Interval             *string
	ClearInterval        bool
	BillableSeats        *int
	CurrentPeriodEnd     *time.Time
	ClearPeriodEnd       bool
	CancelAtPeriodEnd    *bool
	EarlyAccessEndsAt    *time.Time
	ClearEarlyAccess     bool
	MollieCustomerID     *string
	MollieMandateID      *string
	MollieSubscriptionID *string
	ClearSubscriptionID  bool
}

func (s *Store) UpdateWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID, patch BillingPatch) (BillingRow, error) {
	var row BillingRow
	err := s.pool.QueryRow(ctx, `
		UPDATE workspace_billing
		SET
			plan = COALESCE($2, plan),
			status = COALESCE($3, status),
			billing_interval = CASE
				WHEN $4 THEN NULL
				ELSE COALESCE($5, billing_interval)
			END,
			billable_seats = COALESCE($6, billable_seats),
			current_period_end = CASE
				WHEN $7 THEN NULL
				ELSE COALESCE($8, current_period_end)
			END,
			cancel_at_period_end = COALESCE($9, cancel_at_period_end),
			early_access_ends_at = CASE
				WHEN $10 THEN NULL
				ELSE COALESCE($11, early_access_ends_at)
			END,
			mollie_customer_id = COALESCE($12, mollie_customer_id),
			mollie_mandate_id = COALESCE($13, mollie_mandate_id),
			mollie_subscription_id = CASE
				WHEN $14 THEN NULL
				ELSE COALESCE($15, mollie_subscription_id)
			END,
			updated_at = now()
		WHERE workspace_id = $1
		RETURNING `+billingSelectColumns+`
	`,
		workspaceID,
		patch.Plan,
		patch.Status,
		patch.ClearInterval,
		patch.Interval,
		patch.BillableSeats,
		patch.ClearPeriodEnd,
		patch.CurrentPeriodEnd,
		patch.CancelAtPeriodEnd,
		patch.ClearEarlyAccess,
		patch.EarlyAccessEndsAt,
		patch.MollieCustomerID,
		patch.MollieMandateID,
		patch.ClearSubscriptionID,
		patch.MollieSubscriptionID,
	).Scan(
		&row.WorkspaceID, &row.Plan, &row.Status, &row.Interval, &row.BillableSeats,
		&row.CurrentPeriodEnd, &row.CancelAtPeriodEnd, &row.EarlyAccessEndsAt,
		&row.MollieCustomerID, &row.MollieMandateID, &row.MollieSubscriptionID,
		&row.CreatedAt, &row.UpdatedAt,
	)
	out, scanErr := scanBilling(row, err)
	if scanErr != nil {
		if errors.Is(scanErr, ErrNotFound) {
			return BillingRow{}, ErrNotFound
		}
		return BillingRow{}, fmt.Errorf("update workspace billing: %w", scanErr)
	}
	return out, nil
}

func (s *Store) InsertBillingEvent(
	ctx context.Context,
	workspaceID *uuid.UUID,
	molliePaymentID, eventType string,
	payload []byte,
) (bool, error) {
	if payload == nil {
		payload = []byte("{}")
	}
	if !json.Valid(payload) {
		payload = []byte("{}")
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO billing_events (workspace_id, mollie_payment_id, event_type, payload)
		VALUES ($1, NULLIF($2, ''), $3, $4::jsonb)
		ON CONFLICT (mollie_payment_id) WHERE mollie_payment_id IS NOT NULL DO NOTHING
	`, workspaceID, molliePaymentID, eventType, payload)
	if err != nil {
		return false, fmt.Errorf("insert billing event: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) ListFreeCloudWorkspaceIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT workspace_id
		FROM workspace_billing
		WHERE NOT (
			plan = 'pro' AND (
				status IN ('active', 'past_due')
				OR (status = 'cancelled' AND current_period_end IS NOT NULL AND current_period_end > $1)
			)
		)
	`, now)
	if err != nil {
		return nil, fmt.Errorf("list free cloud workspaces: %w", err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan free workspace: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) ListDueBillingWorkspaces(ctx context.Context, now time.Time) ([]BillingRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+billingSelectColumns+`
		FROM workspace_billing
		WHERE
			(cancel_at_period_end AND current_period_end IS NOT NULL AND current_period_end <= $1)
			OR (status = 'past_due' AND updated_at <= $1)
			OR (plan = 'pro' AND early_access_ends_at IS NOT NULL AND early_access_ends_at <= $1)
	`, now)
	if err != nil {
		return nil, fmt.Errorf("list due billing workspaces: %w", err)
	}
	defer rows.Close()
	var out []BillingRow
	for rows.Next() {
		var row BillingRow
		if err := rows.Scan(
			&row.WorkspaceID, &row.Plan, &row.Status, &row.Interval, &row.BillableSeats,
			&row.CurrentPeriodEnd, &row.CancelAtPeriodEnd, &row.EarlyAccessEndsAt,
			&row.MollieCustomerID, &row.MollieMandateID, &row.MollieSubscriptionID,
			&row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan due billing: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) PurgeExpiredMessages(
	ctx context.Context,
	olderThan time.Time,
	workspaceIDs []uuid.UUID,
	limit int,
) (int, error) {
	if len(workspaceIDs) == 0 {
		return 0, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM messages
		WHERE id IN (
			SELECT m.id
			FROM messages m
			INNER JOIN channels c ON c.id = m.channel_id
			WHERE c.workspace_id = ANY($1)
			  AND m.created_at < $2
			ORDER BY m.created_at ASC
			LIMIT $3
		)
	`, workspaceIDs, olderThan, limit)
	if err != nil {
		return 0, fmt.Errorf("purge expired messages: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kilobyteno/dagr-chat/internal/billing"
	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/repository/postgres"
)

var (
	ErrBillingDisabled      = errors.New("billing disabled")
	ErrBillingNotConfigured = errors.New("billing not configured")
	ErrAlreadySubscribed    = errors.New("already subscribed")
	ErrNotSubscribed        = errors.New("not subscribed")
)

type BillingStore interface {
	GetWorkspaceForUser(ctx context.Context, workspaceID, userID uuid.UUID) (postgres.WorkspaceRow, error)
	EnsureWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID, seats int) (postgres.BillingRow, error)
	GetWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID) (postgres.BillingRow, error)
	UpdateWorkspaceBilling(ctx context.Context, workspaceID uuid.UUID, patch postgres.BillingPatch) (postgres.BillingRow, error)
	CountBillableSeats(ctx context.Context, workspaceID uuid.UUID) (int, error)
	InsertBillingEvent(ctx context.Context, workspaceID *uuid.UUID, molliePaymentID, eventType string, payload []byte) (bool, error)
	ListFreeCloudWorkspaceIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error)
	ListDueBillingWorkspaces(ctx context.Context, now time.Time) ([]postgres.BillingRow, error)
	PurgeExpiredMessages(ctx context.Context, olderThan time.Time, workspaceIDs []uuid.UUID, limit int) (int, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.UserRow, error)
}

// BillingSnapshot is the API view of a workspace plan.
type BillingSnapshot struct {
	Enabled            bool
	Plan               domain.PlanID
	Status             domain.BillingStatus
	Interval           domain.BillingInterval
	BillableSeats      int
	CurrentPeriodEnd   *time.Time
	CancelAtPeriodEnd  bool
	EarlyAccessEndsAt  *time.Time
	Entitlements       domain.Entitlements
	MonthlyAmountCents int
	YearlyAmountCents  int
	NextAmountCents    int
	Currency           string
	CanManage          bool
}

type BillingService struct {
	store    BillingStore
	provider billing.Provider
	cfg      config.Config
	logger   *slog.Logger
}

func NewBillingService(store BillingStore, cfg config.Config, provider billing.Provider, logger *slog.Logger) *BillingService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BillingService{store: store, provider: provider, cfg: cfg, logger: logger}
}

func (s *BillingService) Enabled() bool {
	return s != nil && s.cfg.BillingEnabled()
}

func (s *BillingService) ForWorkspace(ctx context.Context, workspaceID string) domain.Entitlements {
	if s == nil || !s.cfg.IsCloud() {
		return domain.UnlimitedEntitlements()
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return domain.FreeCloudEntitlements()
	}
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return domain.FreeCloudEntitlements()
	}
	billingState := row.ToDomain()
	return domain.ResolveEntitlements(true, &billingState, time.Now().UTC())
}

func (s *BillingService) Get(
	ctx context.Context, userID, workspaceID string,
) (*BillingSnapshot, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !s.Enabled() {
		ents := domain.UnlimitedEntitlements()
		return &BillingSnapshot{
			Enabled:      false,
			Plan:         ents.Plan,
			Status:       domain.BillingActive,
			Entitlements: ents,
			Currency:     s.cfg.BillingCurrency,
			CanManage:    canManageWorkspace(domain.WorkspaceRole(ws.Role)),
		}, nil
	}
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return nil, err
	}
	state := row.ToDomain()
	ents := domain.ResolveEntitlements(true, &state, time.Now().UTC())
	snap := &BillingSnapshot{
		Enabled:            true,
		Plan:               ents.Plan,
		Status:             state.Status,
		Interval:           state.Interval,
		BillableSeats:      state.BillableSeats,
		CurrentPeriodEnd:   state.CurrentPeriodEnd,
		CancelAtPeriodEnd:  state.CancelAtPeriodEnd,
		EarlyAccessEndsAt:  state.EarlyAccessEndsAt,
		Entitlements:       ents,
		MonthlyAmountCents: billing.PeriodAmountCents(s.cfg, domain.IntervalMonthly, state.BillableSeats, false),
		YearlyAmountCents:  billing.YearlyListCents(s.cfg) * max(state.BillableSeats, 1),
		Currency:           s.cfg.BillingCurrency,
		CanManage:          canManageWorkspace(domain.WorkspaceRole(ws.Role)),
	}
	if ents.Plan == domain.PlanPro && state.Interval != "" {
		early := state.EarlyAccessEndsAt != nil && state.EarlyAccessEndsAt.After(time.Now().UTC())
		snap.NextAmountCents = billing.RecurringAmountCents(s.cfg, state.Interval, state.BillableSeats, early)
	}
	return snap, nil
}

type CheckoutResult struct {
	CheckoutURL string
	PaymentID   string
}

func (s *BillingService) StartCheckout(
	ctx context.Context, userID, workspaceID, interval string,
) (*CheckoutResult, error) {
	if !s.Enabled() {
		return nil, ErrBillingDisabled
	}
	if s.provider == nil || strings.TrimSpace(s.cfg.MollieAPIKey) == "" {
		return nil, ErrBillingNotConfigured
	}
	iv := domain.BillingInterval(strings.ToLower(strings.TrimSpace(interval)))
	if iv != domain.IntervalMonthly && iv != domain.IntervalYearly {
		return nil, ErrInvalidInput
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return nil, ErrForbidden
	}
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return nil, err
	}
	state := row.ToDomain()
	if state.HasPaidAccess(time.Now().UTC()) && !state.CancelAtPeriodEnd {
		return nil, ErrAlreadySubscribed
	}
	user, err := s.store.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	customerID := state.MollieCustomerID
	if customerID == "" {
		customer, err := s.provider.CreateCustomer(ctx, user.Email, ws.Name)
		if err != nil {
			return nil, fmt.Errorf("create mollie customer: %w", err)
		}
		customerID = customer.ID
		if _, err := s.store.UpdateWorkspaceBilling(ctx, wid, postgres.BillingPatch{
			MollieCustomerID: &customerID,
		}); err != nil {
			return nil, err
		}
	}
	seats := state.BillableSeats
	amount := billing.PeriodAmountCents(s.cfg, iv, seats, true)
	redirect := strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/billing/return?workspaceId=" + wid.String()
	webhook := s.webhookURL()
	payment, err := s.provider.CreateFirstPayment(ctx, billing.CreatePaymentInput{
		CustomerID:  customerID,
		AmountCents: amount,
		Currency:    s.cfg.BillingCurrency,
		Description: fmt.Sprintf("Dagr Pro (%s) for %s", iv, ws.Name),
		RedirectURL: redirect,
		WebhookURL:  webhook,
		Metadata: map[string]string{
			"workspaceId": wid.String(),
			"interval":    string(iv),
			"seats":       fmt.Sprintf("%d", seats),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create mollie payment: %w", err)
	}
	return &CheckoutResult{CheckoutURL: payment.CheckoutURL, PaymentID: payment.ID}, nil
}

func (s *BillingService) Cancel(ctx context.Context, userID, workspaceID string) error {
	if !s.Enabled() {
		return ErrBillingDisabled
	}
	uid, wid, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	_ = uid
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return err
	}
	state := row.ToDomain()
	if !state.HasPaidAccess(time.Now().UTC()) {
		return ErrNotSubscribed
	}
	if state.MollieCustomerID != "" && state.MollieSubscriptionID != "" && s.provider != nil {
		if err := s.provider.CancelSubscription(ctx, state.MollieCustomerID, state.MollieSubscriptionID); err != nil {
			s.logger.Warn("cancel mollie subscription", "error", err, "workspaceId", wid.String())
		}
	}
	cancel := true
	clearSub := true
	_, err = s.store.UpdateWorkspaceBilling(ctx, wid, postgres.BillingPatch{
		CancelAtPeriodEnd:   &cancel,
		ClearSubscriptionID: clearSub,
	})
	return err
}

func (s *BillingService) Resume(ctx context.Context, userID, workspaceID string) error {
	if !s.Enabled() {
		return ErrBillingDisabled
	}
	if s.provider == nil {
		return ErrBillingNotConfigured
	}
	_, wid, err := s.requireManager(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return err
	}
	state := row.ToDomain()
	if !state.CancelAtPeriodEnd || !state.HasPaidAccess(time.Now().UTC()) {
		return ErrNotSubscribed
	}
	if state.MollieCustomerID == "" || state.MollieMandateID == "" {
		return ErrBillingNotConfigured
	}
	early := state.EarlyAccessEndsAt != nil && state.EarlyAccessEndsAt.After(time.Now().UTC())
	amount := billing.RecurringAmountCents(s.cfg, state.Interval, state.BillableSeats, early)
	start := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02")
	if state.CurrentPeriodEnd != nil && state.CurrentPeriodEnd.After(time.Now().UTC()) {
		start = state.CurrentPeriodEnd.UTC().Format("2006-01-02")
	}
	sub, err := s.provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		CustomerID:  state.MollieCustomerID,
		AmountCents: amount,
		Currency:    s.cfg.BillingCurrency,
		Interval:    mollieInterval(state.Interval),
		StartDate:   start,
		Description: "Dagr Pro",
		WebhookURL:  s.webhookURL(),
		Metadata:    map[string]string{"workspaceId": wid.String()},
	})
	if err != nil {
		return err
	}
	cancel := false
	status := string(domain.BillingActive)
	_, err = s.store.UpdateWorkspaceBilling(ctx, wid, postgres.BillingPatch{
		Status:               &status,
		CancelAtPeriodEnd:    &cancel,
		MollieSubscriptionID: &sub.ID,
	})
	return err
}

func (s *BillingService) HandleWebhook(ctx context.Context, paymentID string) error {
	if !s.Enabled() || s.provider == nil {
		return nil
	}
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return ErrInvalidInput
	}
	payment, err := s.provider.GetPayment(ctx, paymentID)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]string{
		"id":     payment.ID,
		"status": payment.Status,
	})
	workspaceID := payment.Metadata["workspaceId"]
	var widPtr *uuid.UUID
	if parsed, err := uuid.Parse(workspaceID); err == nil {
		widPtr = &parsed
	}
	inserted, err := s.store.InsertBillingEvent(ctx, widPtr, payment.ID+":"+payment.Status, "payment."+payment.Status, raw)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}
	if widPtr == nil {
		return nil
	}
	switch payment.Status {
	case "paid":
		return s.applyPaid(ctx, *widPtr, payment)
	case "failed", "expired", "canceled":
		return s.applyFailed(ctx, *widPtr, payment)
	default:
		return nil
	}
}

func (s *BillingService) applyPaid(ctx context.Context, workspaceID uuid.UUID, payment billing.Payment) error {
	row, err := s.ensure(ctx, workspaceID)
	if err != nil {
		return err
	}
	state := row.ToDomain()
	interval := domain.BillingInterval(payment.Metadata["interval"])
	if interval != domain.IntervalMonthly && interval != domain.IntervalYearly {
		if state.Interval != "" {
			interval = state.Interval
		} else {
			interval = domain.IntervalMonthly
		}
	}
	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0)
	if interval == domain.IntervalYearly {
		periodEnd = now.AddDate(1, 0, 0)
	}
	plan := string(domain.PlanPro)
	status := string(domain.BillingActive)
	iv := string(interval)
	cancel := false
	patch := postgres.BillingPatch{
		Plan:              &plan,
		Status:            &status,
		Interval:          &iv,
		CurrentPeriodEnd:  &periodEnd,
		CancelAtPeriodEnd: &cancel,
	}
	if payment.CustomerID != "" {
		patch.MollieCustomerID = &payment.CustomerID
	}
	if payment.MandateID != "" {
		patch.MollieMandateID = &payment.MandateID
	}
	if payment.SequenceType == "first" && s.cfg.EarlyAccessEnabled {
		ends := now.AddDate(0, s.cfg.EarlyAccessMonths, 0)
		patch.EarlyAccessEndsAt = &ends
	}
	if payment.SequenceType == "first" && s.provider != nil && (state.MollieSubscriptionID == "" || state.CancelAtPeriodEnd) {
		customerID := payment.CustomerID
		if customerID == "" {
			customerID = state.MollieCustomerID
		}
		if customerID != "" {
			early := s.cfg.EarlyAccessEnabled && interval == domain.IntervalMonthly
			amount := billing.RecurringAmountCents(s.cfg, interval, state.BillableSeats, early)
			sub, err := s.provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
				CustomerID:  customerID,
				AmountCents: amount,
				Currency:    s.cfg.BillingCurrency,
				Interval:    mollieInterval(interval),
				StartDate:   periodEnd.Format("2006-01-02"),
				Description: "Dagr Pro",
				WebhookURL:  s.webhookURL(),
				Metadata:    map[string]string{"workspaceId": workspaceID.String(), "interval": string(interval)},
			})
			if err != nil {
				s.logger.Error("create mollie subscription", "error", err, "workspaceId", workspaceID.String())
			} else {
				patch.MollieSubscriptionID = &sub.ID
			}
		}
	}
	_, err = s.store.UpdateWorkspaceBilling(ctx, workspaceID, patch)
	return err
}

func (s *BillingService) applyFailed(ctx context.Context, workspaceID uuid.UUID, payment billing.Payment) error {
	if payment.SequenceType == "first" {
		return nil
	}
	row, err := s.ensure(ctx, workspaceID)
	if err != nil {
		return err
	}
	state := row.ToDomain()
	if state.Plan != domain.PlanPro {
		return nil
	}
	status := string(domain.BillingPastDue)
	_, err = s.store.UpdateWorkspaceBilling(ctx, workspaceID, postgres.BillingPatch{Status: &status})
	return err
}

func (s *BillingService) SyncSeats(ctx context.Context, workspaceID string) error {
	if s == nil || !s.Enabled() {
		return nil
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil
	}
	seats, err := s.store.CountBillableSeats(ctx, wid)
	if err != nil {
		return err
	}
	row, err := s.ensure(ctx, wid)
	if err != nil {
		return err
	}
	if _, err := s.store.UpdateWorkspaceBilling(ctx, wid, postgres.BillingPatch{BillableSeats: &seats}); err != nil {
		return err
	}
	state := row.ToDomain()
	state.BillableSeats = seats
	if !state.HasPaidAccess(time.Now().UTC()) || state.MollieSubscriptionID == "" || state.MollieCustomerID == "" || s.provider == nil {
		return nil
	}
	early := state.EarlyAccessEndsAt != nil && state.EarlyAccessEndsAt.After(time.Now().UTC())
	amount := billing.RecurringAmountCents(s.cfg, state.Interval, seats, early)
	_, err = s.provider.UpdateSubscriptionAmount(ctx, state.MollieCustomerID, state.MollieSubscriptionID, amount, s.cfg.BillingCurrency)
	if err != nil {
		s.logger.Warn("update mollie seat amount", "error", err, "workspaceId", workspaceID)
	}
	return nil
}

func (s *BillingService) OnWorkspaceCreated(ctx context.Context, workspaceID string) error {
	if s == nil {
		return nil
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil
	}
	_, err = s.ensure(ctx, wid)
	return err
}

func (s *BillingService) OnWorkspaceDeleting(ctx context.Context, workspaceID string) error {
	if s == nil || !s.Enabled() || s.provider == nil {
		return nil
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil
	}
	row, err := s.store.GetWorkspaceBilling(ctx, wid)
	if err != nil {
		return nil
	}
	state := row.ToDomain()
	if state.MollieCustomerID != "" && state.MollieSubscriptionID != "" {
		_ = s.provider.CancelSubscription(ctx, state.MollieCustomerID, state.MollieSubscriptionID)
	}
	return nil
}

func (s *BillingService) Reconcile(ctx context.Context, now time.Time) error {
	if s == nil || !s.Enabled() {
		return nil
	}
	rows, err := s.store.ListDueBillingWorkspaces(ctx, now)
	if err != nil {
		return err
	}
	for _, row := range rows {
		state := row.ToDomain()
		if state.CancelAtPeriodEnd && state.CurrentPeriodEnd != nil && !state.CurrentPeriodEnd.After(now) {
			if err := s.downgrade(ctx, row.WorkspaceID); err != nil {
				s.logger.Error("downgrade cancelled workspace", "error", err, "workspaceId", row.WorkspaceID.String())
			}
			continue
		}
		if state.Status == domain.BillingPastDue && now.Sub(state.UpdatedAt) >= s.cfg.BillingGracePeriod {
			if err := s.downgrade(ctx, row.WorkspaceID); err != nil {
				s.logger.Error("downgrade past due workspace", "error", err, "workspaceId", row.WorkspaceID.String())
			}
			continue
		}
		if state.Plan == domain.PlanPro && state.EarlyAccessEndsAt != nil && !state.EarlyAccessEndsAt.After(now) {
			if state.Interval == domain.IntervalMonthly && state.MollieSubscriptionID != "" && state.MollieCustomerID != "" && s.provider != nil {
				amount := billing.RecurringAmountCents(s.cfg, state.Interval, state.BillableSeats, false)
				if _, err := s.provider.UpdateSubscriptionAmount(ctx, state.MollieCustomerID, state.MollieSubscriptionID, amount, s.cfg.BillingCurrency); err != nil {
					s.logger.Warn("end early access amount", "error", err, "workspaceId", row.WorkspaceID.String())
				}
			}
			if _, err := s.store.UpdateWorkspaceBilling(ctx, row.WorkspaceID, postgres.BillingPatch{ClearEarlyAccess: true}); err != nil {
				s.logger.Error("clear early access", "error", err, "workspaceId", row.WorkspaceID.String())
			}
		}
	}
	return nil
}

func (s *BillingService) PurgeExpiredHistory(ctx context.Context, now time.Time, limit int) (int, error) {
	if s == nil || !s.cfg.IsCloud() {
		return 0, nil
	}
	ids, err := s.store.ListFreeCloudWorkspaceIDs(ctx, now)
	if err != nil {
		return 0, err
	}
	cutoff := now.AddDate(0, 0, -billing.FreeHistoryDays)
	return s.store.PurgeExpiredMessages(ctx, cutoff, ids, limit)
}

func (s *BillingService) downgrade(ctx context.Context, workspaceID uuid.UUID) error {
	plan := string(domain.PlanFree)
	status := string(domain.BillingActive)
	cancel := false
	_, err := s.store.UpdateWorkspaceBilling(ctx, workspaceID, postgres.BillingPatch{
		Plan:                &plan,
		Status:              &status,
		ClearInterval:       true,
		ClearPeriodEnd:      true,
		CancelAtPeriodEnd:   &cancel,
		ClearEarlyAccess:    true,
		ClearSubscriptionID: true,
	})
	return err
}

func (s *BillingService) ensure(ctx context.Context, workspaceID uuid.UUID) (postgres.BillingRow, error) {
	row, err := s.store.GetWorkspaceBilling(ctx, workspaceID)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, postgres.ErrNotFound) {
		return postgres.BillingRow{}, err
	}
	seats, err := s.store.CountBillableSeats(ctx, workspaceID)
	if err != nil {
		seats = 1
	}
	return s.store.EnsureWorkspaceBilling(ctx, workspaceID, seats)
}

func (s *BillingService) requireManager(ctx context.Context, userID, workspaceID string) (uuid.UUID, uuid.UUID, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrNotFound
	}
	ws, err := s.store.GetWorkspaceForUser(ctx, wid, uid)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return uuid.Nil, uuid.Nil, ErrNotFound
		}
		return uuid.Nil, uuid.Nil, err
	}
	if !canManageWorkspace(domain.WorkspaceRole(ws.Role)) {
		return uuid.Nil, uuid.Nil, ErrForbidden
	}
	return uid, wid, nil
}

func (s *BillingService) webhookURL() string {
	if strings.TrimSpace(s.cfg.MollieWebhookURL) != "" {
		return s.cfg.MollieWebhookURL
	}
	base := strings.TrimRight(s.cfg.ServerPublicURL, "/")
	if base == "" {
		base = strings.TrimRight(s.cfg.PublicBaseURL, "/")
	}
	return base + "/api/v1/billing/webhooks/mollie"
}

func mollieInterval(interval domain.BillingInterval) string {
	if interval == domain.IntervalYearly {
		return "12 months"
	}
	return "1 month"
}

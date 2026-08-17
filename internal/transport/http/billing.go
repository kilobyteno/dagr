package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kilobyteno/dagr-chat/internal/domain"
	"github.com/kilobyteno/dagr-chat/internal/service"
)

type billingEntitlementsJSON struct {
	Plan               string `json:"plan"`
	UnlimitedHistory   bool   `json:"unlimitedHistory"`
	MessageHistoryDays *int   `json:"messageHistoryDays,omitempty"`
	MaxApps            *int   `json:"maxApps,omitempty"`
	CrossWorkspaceDMs  string `json:"crossWorkspaceDms"`
}

type billingJSON struct {
	Enabled            bool                    `json:"enabled"`
	Plan               string                  `json:"plan"`
	Status             string                  `json:"status,omitempty"`
	Interval           string                  `json:"interval,omitempty"`
	BillableSeats      int                     `json:"billableSeats,omitempty"`
	CurrentPeriodEnd   *time.Time              `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd  bool                    `json:"cancelAtPeriodEnd,omitempty"`
	EarlyAccessEndsAt  *time.Time              `json:"earlyAccessEndsAt,omitempty"`
	MonthlyAmountCents int                     `json:"monthlyAmountCents,omitempty"`
	YearlyAmountCents  int                     `json:"yearlyAmountCents,omitempty"`
	NextAmountCents    int                     `json:"nextAmountCents,omitempty"`
	Currency           string                  `json:"currency,omitempty"`
	CanManage          bool                    `json:"canManage"`
	Entitlements       billingEntitlementsJSON `json:"entitlements"`
}

type checkoutRequest struct {
	Interval string `json:"interval"`
}

func toBillingJSON(snap service.BillingSnapshot) billingJSON {
	return billingJSON{
		Enabled:            snap.Enabled,
		Plan:               string(snap.Plan),
		Status:             string(snap.Status),
		Interval:           string(snap.Interval),
		BillableSeats:      snap.BillableSeats,
		CurrentPeriodEnd:   snap.CurrentPeriodEnd,
		CancelAtPeriodEnd:  snap.CancelAtPeriodEnd,
		EarlyAccessEndsAt:  snap.EarlyAccessEndsAt,
		MonthlyAmountCents: snap.MonthlyAmountCents,
		YearlyAmountCents:  snap.YearlyAmountCents,
		NextAmountCents:    snap.NextAmountCents,
		Currency:           snap.Currency,
		CanManage:          snap.CanManage,
		Entitlements: billingEntitlementsJSON{
			Plan:               string(snap.Entitlements.Plan),
			UnlimitedHistory:   snap.Entitlements.UnlimitedHistory,
			MessageHistoryDays: snap.Entitlements.MessageHistoryDays,
			MaxApps:            snap.Entitlements.MaxApps,
			CrossWorkspaceDMs:  string(snap.Entitlements.CrossWorkspaceDMs),
		},
	}
}

func (s *Server) handleGetWorkspaceBilling(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if s.billing == nil {
		ents := domain.UnlimitedEntitlements()
		writeJSON(w, http.StatusOK, map[string]any{
			"billing": toBillingJSON(service.BillingSnapshot{
				Enabled:      false,
				Plan:         ents.Plan,
				Status:       domain.BillingActive,
				Entitlements: ents,
				CanManage:    false,
			}),
		})
		return
	}
	snap, err := s.billing.Get(r.Context(), user.ID, chi.URLParam(r, "workspaceID"))
	if err != nil {
		s.writeBillingError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"billing": toBillingJSON(*snap)})
}

func (s *Server) handleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if s.billing == nil {
		s.writeError(w, r, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server", nil)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
		return
	}
	result, err := s.billing.StartCheckout(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Interval)
	if err != nil {
		s.writeBillingError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checkoutUrl": result.CheckoutURL,
		"paymentId":   result.PaymentID,
	})
}

func (s *Server) handleBillingCancel(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if s.billing == nil {
		s.writeError(w, r, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server", nil)
		return
	}
	if err := s.billing.Cancel(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		s.writeBillingError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBillingResume(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.writeError(w, r, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization", nil)
		return
	}
	if s.billing == nil {
		s.writeError(w, r, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server", nil)
		return
	}
	if err := s.billing.Resume(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		s.writeBillingError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMollieWebhook(w http.ResponseWriter, r *http.Request) {
	if s.billing == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := r.ParseForm(); err != nil {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(body, &struct{}{})
	}
	paymentID := strings.TrimSpace(r.FormValue("id"))
	if paymentID == "" {
		var payload struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		paymentID = strings.TrimSpace(payload.ID)
	}
	if err := s.billing.HandleWebhook(r.Context(), paymentID); err != nil {
		s.writeBillingError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) writeBillingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrBillingDisabled):
		s.writeError(w, r, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server", err)
	case errors.Is(err, service.ErrBillingNotConfigured):
		s.writeError(w, r, http.StatusServiceUnavailable, "billing_not_configured", "Payments are not configured yet", err)
	case errors.Is(err, service.ErrAlreadySubscribed):
		s.writeError(w, r, http.StatusConflict, "already_subscribed", "This workspace already has Pro", err)
	case errors.Is(err, service.ErrNotSubscribed):
		s.writeError(w, r, http.StatusConflict, "not_subscribed", "This workspace is not on Pro", err)
	case errors.Is(err, service.ErrForbidden):
		s.writeError(w, r, http.StatusForbidden, "forbidden", "You cannot manage billing for this workspace", err)
	case errors.Is(err, service.ErrNotFound):
		s.writeError(w, r, http.StatusNotFound, "not_found", "Workspace not found", err)
	case errors.Is(err, service.ErrInvalidInput):
		s.writeError(w, r, http.StatusBadRequest, "invalid_input", "Invalid billing request", err)
	default:
		s.writeError(w, r, http.StatusInternalServerError, "internal_error", "Could not process billing request", err)
	}
}

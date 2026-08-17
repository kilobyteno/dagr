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
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
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
		writeBillingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"billing": toBillingJSON(*snap)})
}

func (s *Server) handleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if s.billing == nil {
		writeError(w, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server")
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return
	}
	result, err := s.billing.StartCheckout(r.Context(), user.ID, chi.URLParam(r, "workspaceID"), req.Interval)
	if err != nil {
		writeBillingError(w, err)
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
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if s.billing == nil {
		writeError(w, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server")
		return
	}
	if err := s.billing.Cancel(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		writeBillingError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBillingResume(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid authorization")
		return
	}
	if s.billing == nil {
		writeError(w, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server")
		return
	}
	if err := s.billing.Resume(r.Context(), user.ID, chi.URLParam(r, "workspaceID")); err != nil {
		writeBillingError(w, err)
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
		writeBillingError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeBillingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBillingDisabled):
		writeError(w, http.StatusNotFound, "billing_disabled", "Billing is not enabled on this server")
	case errors.Is(err, service.ErrBillingNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "billing_not_configured", "Payments are not configured yet")
	case errors.Is(err, service.ErrAlreadySubscribed):
		writeError(w, http.StatusConflict, "already_subscribed", "This workspace already has Pro")
	case errors.Is(err, service.ErrNotSubscribed):
		writeError(w, http.StatusConflict, "not_subscribed", "This workspace is not on Pro")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You cannot manage billing for this workspace")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Workspace not found")
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid billing request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not process billing request")
	}
}

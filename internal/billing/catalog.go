package billing

import (
	"fmt"
	"math"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/domain"
)

const (
	// FreeHistoryDays is Cloud Free message retention.
	FreeHistoryDays = 90
	// FreeMaxApps is the future Cloud Free integration cap.
	FreeMaxApps = 10
)

// PlanCatalogue is the public price list for Cloud.
type PlanCatalogue struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	MonthlyPriceCents       int    `json:"monthlyPriceCents"`
	YearlyPriceCents        int    `json:"yearlyPriceCents"`
	Currency                string `json:"currency"`
	MessageHistoryDays      *int   `json:"messageHistoryDays"`
	UnlimitedHistory        bool   `json:"unlimitedHistory"`
	MaxApps                 *int   `json:"maxApps"`
	UnlimitedApps           bool   `json:"unlimitedApps"`
	CrossWorkspaceDMs       string `json:"crossWorkspaceDms"`
	GroupDMsAvailable       bool   `json:"groupDmsAvailable"`
	AppsLimitAvailable      bool   `json:"appsLimitAvailable"`
	CrossWorkspaceAvailable bool   `json:"crossWorkspaceAvailable"`
}

// EarlyAccessInfo describes the beta discount window.
type EarlyAccessInfo struct {
	Enabled    bool `json:"enabled"`
	Months     int  `json:"months"`
	PercentOff int  `json:"percentOff"`
}

// PublicPlans is the catalogue plus early-access flags for GET /public/config.
type PublicPlans struct {
	Currency    string          `json:"currency"`
	EarlyAccess EarlyAccessInfo `json:"earlyAccess"`
	Plans       []PlanCatalogue `json:"plans"`
}

// Catalog builds the public plan list from server config.
func Catalog(cfg config.Config) PublicPlans {
	freeDays := FreeHistoryDays
	freeApps := FreeMaxApps
	yearly := YearlyListCents(cfg)
	return PublicPlans{
		Currency: cfg.BillingCurrency,
		EarlyAccess: EarlyAccessInfo{
			Enabled:    cfg.EarlyAccessEnabled,
			Months:     cfg.EarlyAccessMonths,
			PercentOff: cfg.EarlyAccessPercentOff,
		},
		Plans: []PlanCatalogue{
			{
				ID:                      string(domain.PlanFree),
				Name:                    "Free",
				MonthlyPriceCents:       0,
				YearlyPriceCents:        0,
				Currency:                cfg.BillingCurrency,
				MessageHistoryDays:      &freeDays,
				MaxApps:                 &freeApps,
				CrossWorkspaceDMs:       string(domain.CrossWorkspaceDMOneToOne),
				AppsLimitAvailable:      false,
				CrossWorkspaceAvailable: false,
			},
			{
				ID:                      string(domain.PlanPro),
				Name:                    "Pro",
				MonthlyPriceCents:       cfg.ProMonthlyCents,
				YearlyPriceCents:        yearly,
				Currency:                cfg.BillingCurrency,
				UnlimitedHistory:        true,
				UnlimitedApps:           true,
				CrossWorkspaceDMs:       string(domain.CrossWorkspaceDMGroup),
				GroupDMsAvailable:       false,
				AppsLimitAvailable:      false,
				CrossWorkspaceAvailable: false,
			},
		},
	}
}

// YearlyListCents is the full-year Pro price after the yearly discount.
func YearlyListCents(cfg config.Config) int {
	full := cfg.ProMonthlyCents * 12
	if cfg.YearlyDiscountPercent <= 0 {
		return full
	}
	return applyPercentOff(full, cfg.YearlyDiscountPercent)
}

// MonthlyEarlyAccessCents is the per-seat monthly price during early access.
func MonthlyEarlyAccessCents(cfg config.Config) int {
	if !cfg.EarlyAccessEnabled {
		return cfg.ProMonthlyCents
	}
	return applyPercentOff(cfg.ProMonthlyCents, cfg.EarlyAccessPercentOff)
}

// YearlyFirstInvoiceCents is the first-year per-seat price with early access.
// Later years use YearlyListCents.
func YearlyFirstInvoiceCents(cfg config.Config) int {
	list := YearlyListCents(cfg)
	if !cfg.EarlyAccessEnabled || cfg.EarlyAccessMonths <= 0 {
		return list
	}
	yearlyUnit := list / 12
	credit := applyPercentOff(yearlyUnit*cfg.EarlyAccessMonths, cfg.EarlyAccessPercentOff)
	if credit > list {
		return 0
	}
	return list - credit
}

// PeriodAmountCents is the charge for seats for a given interval and whether
// this is the first invoice (early access) or a later renewal.
func PeriodAmountCents(cfg config.Config, interval domain.BillingInterval, seats int, firstInvoice bool) int {
	if seats < 1 {
		seats = 1
	}
	switch interval {
	case domain.IntervalYearly:
		unit := YearlyListCents(cfg)
		if firstInvoice {
			unit = YearlyFirstInvoiceCents(cfg)
		}
		return unit * seats
	default:
		unit := cfg.ProMonthlyCents
		if firstInvoice && cfg.EarlyAccessEnabled {
			unit = MonthlyEarlyAccessCents(cfg)
		}
		return unit * seats
	}
}

// RecurringAmountCents is the Mollie subscription amount after the first payment.
func RecurringAmountCents(cfg config.Config, interval domain.BillingInterval, seats int, earlyAccessActive bool) int {
	if seats < 1 {
		seats = 1
	}
	if interval == domain.IntervalYearly {
		return YearlyListCents(cfg) * seats
	}
	unit := cfg.ProMonthlyCents
	if earlyAccessActive {
		unit = MonthlyEarlyAccessCents(cfg)
	}
	return unit * seats
}

// FormatEUR formats euro cents as a Mollie amount string (e.g. "7.00").
func FormatEUR(cents int) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func applyPercentOff(cents, percent int) int {
	if percent <= 0 {
		return cents
	}
	if percent >= 100 {
		return 0
	}
	return int(math.Round(float64(cents) * (1 - float64(percent)/100)))
}

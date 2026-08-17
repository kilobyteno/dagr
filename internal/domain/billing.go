package domain

import "time"

// PlanID is a workspace subscription plan.
type PlanID string

const (
	PlanFree PlanID = "free"
	PlanPro  PlanID = "pro"
)

// BillingStatus is the local lifecycle of a workspace subscription.
type BillingStatus string

const (
	BillingActive    BillingStatus = "active"
	BillingPastDue   BillingStatus = "past_due"
	BillingCancelled BillingStatus = "cancelled"
)

// BillingInterval is how often Pro is charged.
type BillingInterval string

const (
	IntervalMonthly BillingInterval = "monthly"
	IntervalYearly  BillingInterval = "yearly"
)

// CrossWorkspaceDMPolicy is a future Connect entitlement.
type CrossWorkspaceDMPolicy string

const (
	CrossWorkspaceDMOneToOne CrossWorkspaceDMPolicy = "one_to_one"
	CrossWorkspaceDMGroup    CrossWorkspaceDMPolicy = "group"
)

// Entitlements are resolved features for a workspace on this deployment.
type Entitlements struct {
	Plan               PlanID
	UnlimitedHistory   bool
	MessageHistoryDays *int
	MaxApps            *int
	CrossWorkspaceDMs  CrossWorkspaceDMPolicy
}

// UnlimitedEntitlements is used for self-hosted servers.
func UnlimitedEntitlements() Entitlements {
	return Entitlements{
		Plan:              PlanPro,
		UnlimitedHistory:  true,
		CrossWorkspaceDMs: CrossWorkspaceDMGroup,
	}
}

// FreeCloudEntitlements is the forever-free Cloud plan.
func FreeCloudEntitlements() Entitlements {
	days := 90
	apps := 10
	return Entitlements{
		Plan:               PlanFree,
		UnlimitedHistory:   false,
		MessageHistoryDays: &days,
		MaxApps:            &apps,
		CrossWorkspaceDMs:  CrossWorkspaceDMOneToOne,
	}
}

// ProCloudEntitlements is Cloud Pro.
func ProCloudEntitlements() Entitlements {
	return Entitlements{
		Plan:              PlanPro,
		UnlimitedHistory:  true,
		CrossWorkspaceDMs: CrossWorkspaceDMGroup,
	}
}

// WorkspaceBilling is persisted plan state for a workspace.
type WorkspaceBilling struct {
	WorkspaceID          string
	Plan                 PlanID
	Status               BillingStatus
	Interval             BillingInterval
	BillableSeats        int
	CurrentPeriodEnd     *time.Time
	CancelAtPeriodEnd    bool
	EarlyAccessEndsAt    *time.Time
	MollieCustomerID     string
	MollieMandateID      string
	MollieSubscriptionID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// HasPaidAccess reports whether Pro entitlements still apply.
func (b WorkspaceBilling) HasPaidAccess(now time.Time) bool {
	if b.Plan != PlanPro {
		return false
	}
	switch b.Status {
	case BillingActive, BillingPastDue:
		return true
	case BillingCancelled:
		return b.CurrentPeriodEnd != nil && b.CurrentPeriodEnd.After(now)
	default:
		return false
	}
}

// ResolveEntitlements returns self-hosted unlimited, Cloud Pro, or Cloud Free.
func ResolveEntitlements(cloud bool, billing *WorkspaceBilling, now time.Time) Entitlements {
	if !cloud {
		return UnlimitedEntitlements()
	}
	if billing != nil && billing.HasPaidAccess(now) {
		return ProCloudEntitlements()
	}
	return FreeCloudEntitlements()
}

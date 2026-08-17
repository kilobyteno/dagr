package domain

import (
	"testing"
	"time"
)

func TestResolveEntitlements(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	selfHosted := ResolveEntitlements(false, nil, now)
	if !selfHosted.UnlimitedHistory || selfHosted.Plan != PlanPro {
		t.Fatalf("self-hosted entitlements = %+v", selfHosted)
	}

	free := ResolveEntitlements(true, nil, now)
	if free.UnlimitedHistory || free.MessageHistoryDays == nil || *free.MessageHistoryDays != 90 {
		t.Fatalf("cloud free = %+v", free)
	}

	active := &WorkspaceBilling{Plan: PlanPro, Status: BillingActive}
	if got := ResolveEntitlements(true, active, now); !got.UnlimitedHistory {
		t.Fatalf("active pro should be unlimited")
	}

	ended := now.Add(-time.Hour)
	cancelled := &WorkspaceBilling{Plan: PlanPro, Status: BillingCancelled, CurrentPeriodEnd: &ended}
	if got := ResolveEntitlements(true, cancelled, now); got.UnlimitedHistory {
		t.Fatalf("expired cancelled pro should be free")
	}

	future := now.Add(24 * time.Hour)
	grace := &WorkspaceBilling{Plan: PlanPro, Status: BillingCancelled, CurrentPeriodEnd: &future}
	if got := ResolveEntitlements(true, grace, now); !got.UnlimitedHistory {
		t.Fatalf("cancelled still in period should stay pro")
	}
}

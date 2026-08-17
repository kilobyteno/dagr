package billing

import (
	"testing"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/domain"
)

func testBillingConfig() config.Config {
	return config.Config{
		DeploymentMode:        config.DeploymentCloud,
		BillingCurrency:       "EUR",
		ProMonthlyCents:       700,
		YearlyDiscountPercent: 10,
		EarlyAccessEnabled:    true,
		EarlyAccessMonths:     3,
		EarlyAccessPercentOff: 50,
	}
}

func TestYearlyAndEarlyAccessAmounts(t *testing.T) {
	t.Parallel()
	cfg := testBillingConfig()
	if got := YearlyListCents(cfg); got != 7560 {
		t.Fatalf("yearly list = %d", got)
	}
	if got := MonthlyEarlyAccessCents(cfg); got != 350 {
		t.Fatalf("monthly early access = %d", got)
	}
	if got := YearlyFirstInvoiceCents(cfg); got != 6615 {
		t.Fatalf("yearly first invoice = %d", got)
	}
	if got := PeriodAmountCents(cfg, domain.IntervalMonthly, 2, true); got != 700 {
		t.Fatalf("first monthly 2 seats = %d", got)
	}
	if got := PeriodAmountCents(cfg, domain.IntervalMonthly, 2, false); got != 1400 {
		t.Fatalf("later monthly 2 seats = %d", got)
	}
	if got := PeriodAmountCents(cfg, domain.IntervalYearly, 1, true); got != 6615 {
		t.Fatalf("first yearly = %d", got)
	}
	if got := RecurringAmountCents(cfg, domain.IntervalYearly, 1, true); got != 7560 {
		t.Fatalf("yearly recurring = %d", got)
	}
}

func TestFormatEUR(t *testing.T) {
	t.Parallel()
	if got := FormatEUR(700); got != "7.00" {
		t.Fatalf("got %s", got)
	}
	if got := FormatEUR(6615); got != "66.15" {
		t.Fatalf("got %s", got)
	}
}

func TestCatalogCloudPlans(t *testing.T) {
	t.Parallel()
	catalog := Catalog(testBillingConfig())
	if catalog.Currency != "EUR" || len(catalog.Plans) != 2 {
		t.Fatalf("catalog = %+v", catalog)
	}
	if catalog.Plans[0].ID != "free" || catalog.Plans[1].MonthlyPriceCents != 700 {
		t.Fatalf("plans = %+v", catalog.Plans)
	}
}

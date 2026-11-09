package applicationview_test

import (
	"testing"

	"tei/internal/applicationview"
	"tei/internal/core"
)

func closingPeriodRef(day int) applicationview.CommandPeriodRef {
	return applicationview.CommandPeriodRef{
		ID:           core.CanonicalID("sales.month"),
		Data:         core.CanonicalID("sales.order"),
		BusinessDate: core.CanonicalID("sales.order.order_date"),
		ClosingDay:   day,
	}
}

// TestScopeForAWholeMonthPeriod checks that a period without a closing day is
// scoped by the period alone, so an existing state store keyed by period keeps
// working.
func TestScopeForAWholeMonthPeriod(t *testing.T) {
	scope, err := applicationview.ScopeFor(closingPeriodRef(0), "2026-05-21")
	if err != nil {
		t.Fatal(err)
	}
	if scope.Segment != 0 || scope.String() != "sales.month" {
		t.Fatalf("scope=%+v", scope)
	}
}

// TestScopeForAClosingDayUsesTheBusinessDate checks that the two halves of a month
// are different scopes, so closing one half does not refuse the other. X-TEA read
// one month sequence for both (getMSeqOfDate, D:Session.java L2471–2484).
func TestScopeForAClosingDayUsesTheBusinessDate(t *testing.T) {
	before, err := applicationview.ScopeFor(closingPeriodRef(20), "2026-05-20")
	if err != nil {
		t.Fatal(err)
	}
	after, err := applicationview.ScopeFor(closingPeriodRef(20), "2026-05-21")
	if err != nil {
		t.Fatal(err)
	}
	if before.String() != "sales.month#1" || after.String() != "sales.month#2" {
		t.Fatalf("before=%q after=%q", before.String(), after.String())
	}
}

// TestScopeForAClosingDayRequiresADate checks that a split period without a
// business date is refused instead of being read as one half.
func TestScopeForAClosingDayRequiresADate(t *testing.T) {
	if _, err := applicationview.ScopeFor(closingPeriodRef(20), ""); err == nil {
		t.Fatal("a split period without a business date was accepted")
	}
	if _, err := applicationview.ScopeFor(closingPeriodRef(20), "20/05/2026"); err == nil {
		t.Fatal("a business date in the wrong form was accepted")
	}
}

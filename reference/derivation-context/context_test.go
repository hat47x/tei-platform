package derivationcontext

import (
	"errors"
	"testing"
	"time"

	"tei/internal/semantic/derivationref"
)

var jst = time.FixedZone("JST", 9*60*60)

func mustDate(t *testing.T, year int, month time.Month, day int) time.Time {
	t.Helper()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// In JST, 2026-10-13 00:30 is still 2026-10-12 15:30 UTC: the same instant has
// a different calendar date per timezone.
func crossingInstant(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 10, 13, 0, 30, 0, 0, jst)
}

func TestD12ElapsedDaysIsFixedByRequestClock(t *testing.T) {
	start := mustDate(t, 2026, time.October, 12)

	day1, err := ElapsedDays(start, Context{Clock: FixedClock{Instant: crossingInstant(t)}, Location: jst})
	if err != nil || day1 != 1 {
		t.Fatalf("2026-10-13 JST: got %d, %v", day1, err)
	}
	day2, err := ElapsedDays(start, Context{
		Clock:    FixedClock{Instant: time.Date(2026, 10, 14, 0, 30, 0, 0, jst)},
		Location: jst,
	})
	if err != nil || day2 != 2 {
		t.Fatalf("2026-10-14 JST: got %d, %v", day2, err)
	}
	// The same fixed clock and inputs must reproduce the same result.
	again, err := ElapsedDays(start, Context{Clock: FixedClock{Instant: crossingInstant(t)}, Location: jst})
	if err != nil || again != day1 {
		t.Fatalf("not deterministic: %d != %d (%v)", again, day1, err)
	}
}

func TestD12TimezoneChangesTheDateBoundary(t *testing.T) {
	start := mustDate(t, 2026, time.October, 12)
	instant := crossingInstant(t)

	inJST, err := ElapsedDays(start, Context{Clock: FixedClock{Instant: instant}, Location: jst})
	if err != nil {
		t.Fatal(err)
	}
	inUTC, err := ElapsedDays(start, Context{Clock: FixedClock{Instant: instant}, Location: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	if inJST != 1 || inUTC != 0 {
		t.Fatalf("timezone was not honoured: JST=%d UTC=%d", inJST, inUTC)
	}
}

func TestD12MissingContextFailsClosed(t *testing.T) {
	start := mustDate(t, 2026, time.October, 12)
	if _, err := ElapsedDays(start, Context{Clock: UnfixedClock{}, Location: jst}); !errors.Is(err, ErrClockUnfixed) {
		t.Fatalf("unfixed clock was used: %v", err)
	}
	if _, err := ElapsedDays(start, Context{Location: jst}); !errors.Is(err, ErrContextMissing) {
		t.Fatalf("missing clock accepted: %v", err)
	}
	if _, err := ElapsedDays(start, Context{Clock: FixedClock{Instant: crossingInstant(t)}}); !errors.Is(err, ErrContextMissing) {
		t.Fatalf("missing timezone accepted: %v", err)
	}
}

func TestD12OneRequestInstantForEveryRow(t *testing.T) {
	// One request fixes one instant; rows differ only by their own business
	// start date, so the whole page is consistent even across a date boundary.
	clock := FixedClock{Instant: crossingInstant(t)}
	starts := []time.Time{
		mustDate(t, 2026, time.October, 11),
		mustDate(t, 2026, time.October, 12),
		mustDate(t, 2026, time.October, 13),
	}
	want := []int{2, 1, 0}
	for pass := 0; pass < 2; pass++ {
		for i, start := range starts {
			got, err := ElapsedDays(start, Context{Clock: clock, Location: jst})
			if err != nil || got != want[i] {
				t.Fatalf("pass %d row %d: got %d, want %d (%v)", pass, i, got, want[i], err)
			}
		}
	}
}

func TestD12PureRuleNeedsNoContext(t *testing.T) {
	if err := Check(Requirement{}, Context{}); err != nil {
		t.Fatalf("pure rule demanded context: %v", err)
	}
	amount, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: "2"},
		derivationref.Input{State: derivationref.Present, Decimal: "300"},
	)
	if err != nil || amount.Decimal != "600" {
		t.Fatalf("pure rule: %+v, %v", amount, err)
	}
}

func TestD12LocaleOnlyRequiredWhenDeclared(t *testing.T) {
	if err := Check(Requirement{NeedsLocale: true}, Context{}); !errors.Is(err, ErrContextMissing) {
		t.Fatalf("declared locale not required: %v", err)
	}
	if err := Check(Requirement{NeedsLocale: true}, Context{Locale: "ja-JP"}); err != nil {
		t.Fatalf("supplied locale rejected: %v", err)
	}
	if err := Check(Requirement{}, Context{Locale: ""}); err != nil {
		t.Fatalf("locale forced onto a rule that does not need it: %v", err)
	}
}

func TestD12EvaluationDateBeforeStartIsRejected(t *testing.T) {
	start := mustDate(t, 2026, time.October, 20)
	if _, err := ElapsedDays(start, Context{
		Clock:    FixedClock{Instant: crossingInstant(t)},
		Location: jst,
	}); !errors.Is(err, ErrContextMissing) {
		t.Fatalf("negative elapsed days accepted: %v", err)
	}
}

package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
)

func closingPeriod(t *testing.T, day int) businesscalendar.ClosingPeriod {
	t.Helper()
	return businesscalendar.ClosingPeriod{Calendar: testCalendar(t), ClosingDay: day}
}

// TestClosingPeriodKeepsTheMonth checks that a date after the closing day stays in
// the month it names. X-TEA moved it into the next month's sequence
// (getMSeqOfDate, D:Session.java L2471–2484), which also moved the fiscal year.
func TestClosingPeriodKeepsTheMonth(t *testing.T) {
	period := closingPeriod(t, 20)
	before, err := period.KeyFor(day(t, "2026-05-20"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := period.KeyFor(day(t, "2026-05-21"))
	if err != nil {
		t.Fatal(err)
	}
	if before.FiscalYear != after.FiscalYear || before.MonthSequence != after.MonthSequence {
		t.Fatalf("before=%+v after=%+v", before, after)
	}
	if before.CalendarMonth != 5 || after.CalendarMonth != 5 {
		t.Fatalf("before=%+v after=%+v", before, after)
	}
	if before.ClosingSegment != businesscalendar.SegmentBefore {
		t.Fatalf("before=%+v", before)
	}
	if after.ClosingSegment != businesscalendar.SegmentAfter {
		t.Fatalf("after=%+v", after)
	}
}

// TestClosingPeriodSeparatesTheHalves checks that the two halves of one month get
// different keys, so a closure can name one of them.
func TestClosingPeriodSeparatesTheHalves(t *testing.T) {
	period := closingPeriod(t, 20)
	before, err := period.KeyFor(day(t, "2026-05-10"))
	if err != nil {
		t.Fatal(err)
	}
	after, err := period.KeyFor(day(t, "2026-05-25"))
	if err != nil {
		t.Fatal(err)
	}
	if before.String() == after.String() {
		t.Fatalf("both halves render as %q", before.String())
	}
	// The fixture calendar starts its fiscal year in April, so May is sequence 2
	// of fiscal year 2027.
	if before.String() != "2027-02-1" || after.String() != "2027-02-2" {
		t.Fatalf("before=%q after=%q", before.String(), after.String())
	}
}

// TestClosingPeriodRefusesAnImpossibleDay checks that days 29–31 are refused,
// because February has no such day and a boundary that does not exist every month
// is not a boundary.
func TestClosingPeriodRefusesAnImpossibleDay(t *testing.T) {
	for _, dayValue := range []int{0, 29, 30, 31} {
		period := businesscalendar.ClosingPeriod{Calendar: testCalendar(t), ClosingDay: dayValue}
		if err := period.Validate(); err == nil {
			t.Fatalf("closing day %d was accepted", dayValue)
		}
	}
}

// TestMonthPeriodKeyHasNoSegment checks that a whole-month period keeps the
// previous key form, so an existing declaration is unchanged.
func TestMonthPeriodKeyHasNoSegment(t *testing.T) {
	key, err := testCalendar(t).PeriodKeyFor(day(t, "2026-05-21"))
	if err != nil {
		t.Fatal(err)
	}
	if key.ClosingSegment != 0 {
		t.Fatalf("key=%+v", key)
	}
	if key.String() != "2027-02" {
		t.Fatalf("key=%q", key.String())
	}
}

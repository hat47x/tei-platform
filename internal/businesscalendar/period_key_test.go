package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
)

// TestPeriodKeyForFollowsTheDeclaredStart checks that the month sequence starts
// at the declared fiscal year start, so the key a closure refers to is the key
// the declaration names.
func TestPeriodKeyForFollowsTheDeclaredStart(t *testing.T) {
	calendar := testCalendar(t)
	april, err := calendar.PeriodKeyFor(day(t, "2026-04-15"))
	if err != nil {
		t.Fatal(err)
	}
	if april.FiscalYear != 2027 || april.MonthSequence != 1 || april.CalendarMonth != 4 {
		t.Fatalf("april=%+v", april)
	}
	march, err := calendar.PeriodKeyFor(day(t, "2027-03-15"))
	if err != nil {
		t.Fatal(err)
	}
	if march.FiscalYear != 2027 || march.MonthSequence != 12 || march.CalendarMonth != 3 {
		t.Fatalf("march=%+v", march)
	}
	if march.String() != "2027-12" {
		t.Fatalf("string=%q", march.String())
	}
}

// TestPeriodKeyForJanuaryStart checks that a calendar starting in January gives
// the sequence equal to the calendar month.
func TestPeriodKeyForJanuaryStart(t *testing.T) {
	calendar := testCalendar(t)
	calendar.FiscalYearStartMonth = 1
	key, err := calendar.PeriodKeyFor(day(t, "2026-01-10"))
	if err != nil {
		t.Fatal(err)
	}
	if key.MonthSequence != 1 || key.FiscalYear != 2027 {
		t.Fatalf("key=%+v", key)
	}
}

// TestPeriodStartInvertsTheKey checks that the period's first date is derivable
// from the key, so a closure record can be turned back into a range.
func TestPeriodStartInvertsTheKey(t *testing.T) {
	calendar := testCalendar(t)
	for _, month := range []string{"2026-05-20", "2026-04-01", "2027-03-31"} {
		date := day(t, month)
		key, err := calendar.PeriodKeyFor(date)
		if err != nil {
			t.Fatal(err)
		}
		start, err := calendar.PeriodStart(key)
		if err != nil {
			t.Fatal(err)
		}
		keyOfStart, err := calendar.PeriodKeyFor(start)
		if err != nil {
			t.Fatal(err)
		}
		if keyOfStart != key {
			t.Fatalf("date=%s key=%+v start=%s startKey=%+v", month, key, start.Format("2006-01-02"), keyOfStart)
		}
		if start.Month() == 0 {
			t.Fatalf("empty start for %s", month)
		}
	}
}

// TestPeriodStartRefusesAnImpossibleSequence checks that a sequence outside
// 1..12 is refused instead of wrapping into another year.
func TestPeriodStartRefusesAnImpossibleSequence(t *testing.T) {
	calendar := testCalendar(t)
	for _, sequence := range []int{0, 13, -1} {
		if _, err := calendar.PeriodStart(businesscalendar.PeriodKey{FiscalYear: 2027, MonthSequence: sequence}); err == nil {
			t.Fatalf("sequence %d was accepted", sequence)
		}
	}
}

// TestPeriodKeyDoesNotShiftAcrossAClosingDay checks the difference from X-TEA:
// a date always belongs to the month it names, so a "last day" setting cannot
// move it into the next month's period.
func TestPeriodKeyDoesNotShiftAcrossAClosingDay(t *testing.T) {
	calendar := testCalendar(t)
	last, err := calendar.PeriodKeyFor(day(t, "2026-04-30"))
	if err != nil {
		t.Fatal(err)
	}
	if last.MonthSequence != 1 {
		t.Fatalf("last day of start month=%+v", last)
	}
	next, err := calendar.PeriodKeyFor(day(t, "2026-05-01"))
	if err != nil {
		t.Fatal(err)
	}
	if next.MonthSequence != 2 {
		t.Fatalf("next day=%+v", next)
	}
}

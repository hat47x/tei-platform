package businesscalendar_test

import (
	"errors"
	"testing"
	"time"

	"tei/internal/businesscalendar"
)

func testLocale(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load time zone: %v", err)
	}
	return location
}

func testCalendar(t *testing.T, holidays ...time.Time) businesscalendar.Calendar {
	t.Helper()
	return businesscalendar.Calendar{
		FiscalYearStartMonth: 4,
		WeekStarts:           businesscalendar.WeekStartsMonday,
		Holidays:             holidays,
		Locale:               testLocale(t),
	}
}

func day(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, testLocale(t))
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed
}

// TestDaysBetweenNamesTheRule checks that the count is chosen by name, not by an
// unexplained integer, and that the three rules give the results X-TEA's
// countType gave.
func TestDaysBetweenNamesTheRule(t *testing.T) {
	calendar := testCalendar(t, day(t, "2026-04-29"), day(t, "2026-05-05"))
	from, thru := day(t, "2026-04-27"), day(t, "2026-05-08")

	elapsed, err := calendar.DaysBetween(from, thru, businesscalendar.DayCountElapsed)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed != 11 {
		t.Fatalf("elapsed=%d", elapsed)
	}
	span, err := calendar.DaysBetween(from, thru, businesscalendar.DayCountCalendar)
	if err != nil {
		t.Fatal(err)
	}
	if span != 11 {
		t.Fatalf("calendar=%d", span)
	}
	business, err := calendar.DaysBetween(from, thru, businesscalendar.DayCountBusiness)
	if err != nil {
		t.Fatal(err)
	}
	if business != 9 {
		t.Fatalf("business=%d", business)
	}
}

// TestDaysBetweenCountsNegativeSpans checks that the direction is the caller's
// span, not an error: a caller does not have to swap the arguments.
func TestDaysBetweenCountsNegativeSpans(t *testing.T) {
	calendar := testCalendar(t)
	from, thru := day(t, "2026-05-08"), day(t, "2026-04-27")
	count, err := calendar.DaysBetween(from, thru, businesscalendar.DayCountElapsed)
	if err != nil {
		t.Fatal(err)
	}
	if count != -11 {
		t.Fatalf("count=%d", count)
	}
}

// TestDaysBetweenIgnoresTheClockTime checks that a date difference is a date
// difference: an hour of daylight saving or a time of day does not change it.
func TestDaysBetweenIgnoresTheClockTime(t *testing.T) {
	calendar := testCalendar(t)
	from := day(t, "2026-04-27").Add(23*time.Hour + 30*time.Minute)
	thru := day(t, "2026-04-29").Add(time.Hour)
	count, err := calendar.DaysBetween(from, thru, businesscalendar.DayCountElapsed)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count=%d", count)
	}
}

// TestAddBusinessDaysStopsOnABusinessDay checks that moving over a holiday
// lands on a business day, and that zero lands on a business day when the start
// is one.
func TestAddBusinessDaysStopsOnABusinessDay(t *testing.T) {
	calendar := testCalendar(t, day(t, "2026-05-02"))
	start := day(t, "2026-05-01")
	result, err := calendar.AddBusinessDays(start, 1)
	if err != nil {
		t.Fatal(err)
	}
	// X-TEA counted only holidays out of business days, never a weekend
	// (D:Session.java L1803), so the day after a Friday holiday is the Sunday.
	if !result.Equal(day(t, "2026-05-03")) {
		t.Fatalf("result=%s", result.Format("2006-01-02"))
	}
	zero, err := calendar.AddBusinessDays(start, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !zero.Equal(start) {
		t.Fatalf("zero=%s", zero.Format("2006-01-02"))
	}
	// A start on a holiday moves forward to the next business day.
	onHoliday, err := calendar.AddBusinessDays(day(t, "2026-05-02"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !onHoliday.Equal(day(t, "2026-05-03")) {
		t.Fatalf("holiday start=%s", onHoliday.Format("2006-01-02"))
	}
}

// TestAddBusinessDaysCountsBackwards checks the negative direction.
func TestAddBusinessDaysCountsBackwards(t *testing.T) {
	calendar := testCalendar(t)
	result, err := calendar.AddBusinessDays(day(t, "2026-05-08"), -2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Equal(day(t, "2026-05-06")) {
		t.Fatalf("result=%s", result.Format("2006-01-02"))
	}
}

// TestFiscalYearFollowsTheDeclaredStart checks that the fiscal year label comes
// from the declared start month, not from the calendar year.
func TestFiscalYearFollowsTheDeclaredStart(t *testing.T) {
	calendar := testCalendar(t)
	march, err := calendar.FiscalYear(day(t, "2026-03-31"))
	if err != nil {
		t.Fatal(err)
	}
	if march != 2026 {
		t.Fatalf("march label=%d", march)
	}
	april, err := calendar.FiscalYear(day(t, "2026-04-01"))
	if err != nil {
		t.Fatal(err)
	}
	if april != 2027 {
		t.Fatalf("april label=%d", april)
	}
	start, err := calendar.FiscalYearStart(day(t, "2026-05-01"))
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(day(t, "2026-04-01")) {
		t.Fatalf("start=%s", start.Format("2006-01-02"))
	}
}

// TestCalendarWithoutABoundaryIsRefused checks that silence is not defaulted: a
// calendar without a fiscal year start or a week start does not derive a date.
func TestCalendarWithoutABoundaryIsRefused(t *testing.T) {
	cases := []businesscalendar.Calendar{
		{FiscalYearStartMonth: 0, WeekStarts: businesscalendar.WeekStartsMonday, Locale: testLocale(t)},
		{FiscalYearStartMonth: 13, WeekStarts: businesscalendar.WeekStartsMonday, Locale: testLocale(t)},
		{FiscalYearStartMonth: 4, Locale: testLocale(t)},
		{FiscalYearStartMonth: 4, WeekStarts: businesscalendar.WeekStartsMonday},
	}
	for index, calendar := range cases {
		if _, err := calendar.DaysBetween(day(t, "2026-04-01"), day(t, "2026-04-02"), businesscalendar.DayCountElapsed); !errors.Is(err, businesscalendar.ErrInvalidCalendar) {
			t.Fatalf("case %d err=%v", index, err)
		}
	}
}

// TestDaysBetweenRefusesAnUnnamedRule checks that an unexplained count is
// refused instead of treated as one of the rules.
func TestDaysBetweenRefusesAnUnnamedRule(t *testing.T) {
	calendar := testCalendar(t)
	if _, err := calendar.DaysBetween(day(t, "2026-04-01"), day(t, "2026-04-02"), "1"); !errors.Is(err, businesscalendar.ErrInvalidCalendar) {
		t.Fatalf("err=%v", err)
	}
}

// TestHolidaySetReportsTheSetUsed checks that the holiday set is readable in a
// stable order, so a record can say which set a derivation used.
func TestHolidaySetReportsTheSetUsed(t *testing.T) {
	calendar := testCalendar(t, day(t, "2026-05-05"), day(t, "2026-04-29"))
	set := calendar.HolidaySet()
	if len(set) != 2 || !set[0].Equal(day(t, "2026-04-29")) {
		t.Fatalf("set=%v", set)
	}
}

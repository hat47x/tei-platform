package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// holidayPeriodWorld declares a period whose calendar shares one holiday Data
// with another calendar. The identity type closes the set of calendars.
func holidayPeriodWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "std",
				Types: []semantic.Type{
					{ID: "std.code", Symbol: "code", Kind: semantic.TypePrimitive},
					{ID: "std.date", Symbol: "date", Kind: semantic.TypePrimitive},
					{ID: "sales.calendar_id", Symbol: "calendar_id", Kind: semantic.TypeAlias, Underlying: "std.code", Values: []string{"sales.tokyo", "sales.osaka"}},
				},
			},
			{
				ID: "sales",
				Data: []semantic.Data{
					{ID: "sales.order", Attributes: []semantic.Attribute{{ID: "sales.order.order_date", TypeRef: "std.date"}}},
					{
						ID: "sales.holiday",
						Attributes: []semantic.Attribute{
							{ID: "sales.holiday.calendar", TypeRef: "sales.calendar_id", Role: businesscalendar.RoleHolidayCalendar},
							{ID: "sales.holiday.holiday_on", TypeRef: "std.date", Role: businesscalendar.RoleHolidayDate},
						},
					},
				},
				Calendars: []semantic.Calendar{
					{ID: "sales.tokyo", FiscalYearStartMonth: 4, WeekStarts: "monday", BusinessDayExcludes: "sales.holiday"},
					{ID: "sales.osaka", FiscalYearStartMonth: 4, WeekStarts: "monday", BusinessDayExcludes: "sales.holiday"},
				},
				Periods: []semantic.Period{
					{ID: "sales.month", Data: "sales.order", BusinessDate: "sales.order.order_date", Calendar: "sales.tokyo", States: []string{"open", "closing", "closed"}},
				},
			},
		},
	}
}

func holidayPeriodRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"sales.holiday.calendar": "sales.tokyo", "sales.holiday.holiday_on": "2026-05-02"},
		{"sales.holiday.calendar": "sales.osaka", "sales.holiday.holiday_on": "2026-05-06"},
	}
}

// TestHolidayResolveJoinsTheSteps checks that one call resolves the declaration,
// binds the holiday Data, filters by calendar, and checks the identities.
func TestHolidayResolveJoinsTheSteps(t *testing.T) {
	dates, err := businesscalendar.HolidayResolve(holidayPeriodWorld(), "sales.tokyo", holidayPeriodRows(), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 1 || !dates[0].Equal(day(t, "2026-05-02")) {
		t.Fatalf("dates=%v", dates)
	}
}

// TestHolidayResolveReportsAMisspeltIdentity checks that a row whose identity is
// outside the declared values is reported even when the row belongs to another
// calendar. X-TEA compared codes as plain strings (D:Session.java L786), so that
// row was skipped without a word and the holiday disappeared silently.
func TestHolidayResolveReportsAMisspeltIdentity(t *testing.T) {
	rows := holidayPeriodRows()
	rows[1]["sales.holiday.calendar"] = "sales.kyoto"
	if _, err := businesscalendar.HolidayResolve(holidayPeriodWorld(), "sales.tokyo", rows, testLocale(t)); err == nil {
		t.Fatal("a misspelt identity on another calendar row was accepted")
	}
}

// TestHolidayResolveLeavesASingleCalendarDataAlone checks that a Data without
// identities keeps working through the joined call.
func TestHolidayResolveLeavesASingleCalendarDataAlone(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	rows := []map[core.CanonicalID]string{
		{"sales.holiday.holiday_on": "2026-05-02"},
		{"sales.holiday.holiday_on": "2026-04-29"},
	}
	dates, err := businesscalendar.HolidayResolve(world, "sales.fiscal", rows, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 2 {
		t.Fatalf("dates=%v", dates)
	}
}

// TestCalendarForPeriodBuildsTheCalendarThePeriodFollows checks the joined path
// from a period to a calendar a derivation can use.
func TestCalendarForPeriodBuildsTheCalendarThePeriodFollows(t *testing.T) {
	calendar, err := businesscalendar.CalendarForPeriod(holidayPeriodWorld(), "sales.month", holidayPeriodRows(), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := calendar.Excludes(day(t, "2026-05-02"))
	if err != nil {
		t.Fatal(err)
	}
	if !excluded {
		t.Fatal("the period's calendar did not exclude its own holiday")
	}
	// The other calendar's holiday is not this calendar's holiday.
	notExcluded, err := calendar.Excludes(day(t, "2026-05-06"))
	if err != nil {
		t.Fatal(err)
	}
	if notExcluded {
		t.Fatal("the period's calendar took another calendar's holiday")
	}
}

// TestCalendarForPeriodRefusesAPeriodWithoutACalendar checks that a period that
// declares no calendar is refused, so a caller does not silently use another
// period's boundary.
func TestCalendarForPeriodRefusesAPeriodWithoutACalendar(t *testing.T) {
	world := holidayPeriodWorld()
	world.Semantics[1].Periods[0].Calendar = ""
	if _, err := businesscalendar.CalendarForPeriod(world, "sales.month", holidayPeriodRows(), testLocale(t)); err == nil {
		t.Fatal("a period without a calendar was accepted")
	}
}

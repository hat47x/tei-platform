package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func holidayWorld(role string) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.holiday",
						Attributes: []semantic.Attribute{
							{ID: "sales.holiday.calendar_id", TypeRef: "std.identifier"},
							{ID: "sales.holiday.holiday_on", TypeRef: "std.date", Role: role},
						},
					},
				},
				Calendars: []semantic.Calendar{
					{
						ID:                   "sales.fiscal",
						FiscalYearStartMonth: 4,
						WeekStarts:           "monday",
						BusinessDayExcludes:  "sales.holiday",
					},
				},
			},
		},
	}
}

func holidayRows(dates ...string) []map[core.CanonicalID]string {
	rows := make([]map[core.CanonicalID]string, 0, len(dates))
	for _, date := range dates {
		rows = append(rows, map[core.CanonicalID]string{"sales.holiday.holiday_on": date})
	}
	return rows
}

// TestBindHolidaysFindsTheDeclaredDate checks that the calendar names the
// holiday Data and the Data names the date attribute by role.
func TestBindHolidaysFindsTheDeclaredDate(t *testing.T) {
	declaration, err := businesscalendar.Resolve(holidayWorld(businesscalendar.RoleHolidayDate), "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(holidayWorld(businesscalendar.RoleHolidayDate))
	if err != nil {
		t.Fatal(err)
	}
	if binding.DataID != "sales.holiday" || binding.Date != "sales.holiday.holiday_on" {
		t.Fatalf("binding=%+v", binding)
	}
}

// TestBindHolidaysRefusesAnUnroledDate checks that a Data without the role is
// reported instead of binding a guessed column.
func TestBindHolidaysRefusesAnUnroledDate(t *testing.T) {
	world := holidayWorld("")
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := declaration.BindHolidays(world); err == nil {
		t.Fatal("a holiday data without the date role was accepted")
	}
}

// TestBindHolidaysRefusesAnUndeclaredData checks that the named Data must exist.
func TestBindHolidaysRefusesAnUndeclaredData(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	declaration.HolidayData = "sales.missing"
	if _, err := declaration.BindHolidays(world); err == nil {
		t.Fatal("an undeclared holiday data was accepted")
	}
}

// TestHolidayRowsReadsTheDates checks that rows become dates and that the
// resulting calendar excludes them.
func TestHolidayRowsReadsTheDates(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	dates, err := binding.HolidayRows(holidayRows("2026-05-02", "2026-04-29"), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	calendar, err := declaration.Build(dates, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if set := calendar.HolidaySet(); len(set) != 2 || !set[0].Equal(day(t, "2026-04-29")) {
		t.Fatalf("holiday set=%v", set)
	}
	excluded, err := calendar.Excludes(day(t, "2026-05-02"))
	if err != nil {
		t.Fatal(err)
	}
	if !excluded {
		t.Fatal("the bound holiday was not excluded")
	}
	business, err := calendar.IsBusinessDay(day(t, "2026-05-01"))
	if err != nil {
		t.Fatal(err)
	}
	if !business {
		t.Fatal("a day that is not a holiday was excluded")
	}
}

// TestHolidayRowsReportABadDate checks that an unreadable date names the row and
// the attribute. X-TEA read a date string straight into its list, so a
// differently formatted holiday was not excluded and nothing reported it
// (D:Session.java L1691).
func TestHolidayRowsReportABadDate(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	for _, dates := range [][]string{{""}, {"02/05/2026"}} {
		if _, err := binding.HolidayRows(holidayRows(dates...), testLocale(t)); err == nil {
			t.Fatalf("dates %v were accepted", dates)
		}
	}
	if _, err := binding.HolidayRows(holidayRows("2026-05-02"), nil); err == nil {
		t.Fatal("a bind without a zone was accepted")
	}
}

// TestCalendarWithoutHolidayDataHasNoHolidays checks that a calendar that names
// no exclusion data excludes nothing, and that this is not an error.
func TestCalendarWithoutHolidayDataHasNoHolidays(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	world.Semantics[0].Calendars[0].BusinessDayExcludes = ""
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	if binding.DataID != "" || binding.Date != "" {
		t.Fatalf("binding=%+v", binding)
	}
}

package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// multiCalendarWorld declares one holiday Data shared by two calendars. The Data
// carries a calendar identity so a row belongs to one calendar only.
func multiCalendarWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.holiday",
						Attributes: []semantic.Attribute{
							{ID: "sales.holiday.calendar", TypeRef: "std.code", Role: businesscalendar.RoleHolidayCalendar},
							{ID: "sales.holiday.holiday_on", TypeRef: "std.date", Role: businesscalendar.RoleHolidayDate},
						},
					},
				},
				Calendars: []semantic.Calendar{
					{ID: "sales.tokyo", FiscalYearStartMonth: 4, WeekStarts: "monday", BusinessDayExcludes: "sales.holiday"},
					{ID: "sales.osaka", FiscalYearStartMonth: 4, WeekStarts: "monday", BusinessDayExcludes: "sales.holiday"},
				},
			},
		},
	}
}

func multiCalendarRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"sales.holiday.calendar": "sales.tokyo", "sales.holiday.holiday_on": "2026-05-02"},
		{"sales.holiday.calendar": "sales.osaka", "sales.holiday.holiday_on": "2026-05-06"},
		{"sales.holiday.calendar": "sales.tokyo", "sales.holiday.holiday_on": "2026-04-29"},
	}
}

// TestBindHolidaysFindsTheCalendarIdentity checks that the optional calendar role
// is resolved when the Data declares it.
func TestBindHolidaysFindsTheCalendarIdentity(t *testing.T) {
	declaration, err := businesscalendar.Resolve(multiCalendarWorld(), "sales.tokyo")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(multiCalendarWorld())
	if err != nil {
		t.Fatal(err)
	}
	if binding.Calendar != "sales.holiday.calendar" || binding.Date != "sales.holiday.holiday_on" {
		t.Fatalf("binding=%+v", binding)
	}
}

// TestHolidayRowsForCalendarTakesOnlyItsOwnHolidays checks that two calendars can
// share one Data without taking each other's holidays.
func TestHolidayRowsForCalendarTakesOnlyItsOwnHolidays(t *testing.T) {
	world := multiCalendarWorld()
	declaration, err := businesscalendar.Resolve(world, "sales.tokyo")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	tokyo, err := binding.HolidayRowsForCalendar("sales.tokyo", multiCalendarRows(), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(tokyo) != 2 {
		t.Fatalf("tokyo=%v", tokyo)
	}
	osaka, err := binding.HolidayRowsForCalendar("sales.osaka", multiCalendarRows(), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(osaka) != 1 || !osaka[0].Equal(day(t, "2026-05-06")) {
		t.Fatalf("osaka=%v", osaka)
	}
}

// TestHolidayRowsRefusesAnUnnamedCalendar checks that a shared Data cannot be
// read without saying which calendar, so a caller does not silently take every
// calendar's holidays.
func TestHolidayRowsRefusesAnUnnamedCalendar(t *testing.T) {
	world := multiCalendarWorld()
	declaration, err := businesscalendar.Resolve(world, "sales.tokyo")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := binding.HolidayRows(multiCalendarRows(), testLocale(t)); err == nil {
		t.Fatal("a shared holiday data was read without naming a calendar")
	}
	if _, err := binding.HolidayRowsForCalendar("", multiCalendarRows(), testLocale(t)); err == nil {
		t.Fatal("an empty calendar was accepted")
	}
}

// TestHolidayRowsForCalendarIgnoresTheFilterForASingleCalendar checks that a
// Data without a calendar identity keeps its previous behaviour: every row
// belongs to the one calendar.
func TestHolidayRowsForCalendarIgnoresTheFilterForASingleCalendar(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Calendar != "" {
		t.Fatalf("binding=%+v", binding)
	}
	dates, err := binding.HolidayRowsForCalendar("sales.fiscal", holidayRows("2026-05-02"), testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 1 {
		t.Fatalf("dates=%v", dates)
	}
	// The plain path still works for a single-calendar Data.
	if _, err := binding.HolidayRows(holidayRows("2026-05-02"), testLocale(t)); err != nil {
		t.Fatal(err)
	}
}

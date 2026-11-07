package businesscalendar_test

import (
	"testing"
	"time"

	"tei/internal/businesscalendar"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func resolveWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID:         "sales.holiday",
						Attributes: []semantic.Attribute{{ID: "sales.holiday.holiday_on", TypeRef: "std.date"}},
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
				Periods: []semantic.Period{
					{
						ID:       "sales.month",
						Data:     "sales.order",
						Calendar: "sales.fiscal",
					},
				},
			},
		},
	}
}

// TestResolveReadsTheDeclaration checks that the runtime calendar comes from the
// declaration: the fiscal year start, the week start, and the holiday Data.
func TestResolveReadsTheDeclaration(t *testing.T) {
	declaration, err := businesscalendar.Resolve(resolveWorld(), "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	if declaration.FiscalYearStartMonth != 4 || declaration.WeekStarts != businesscalendar.WeekStartsMonday {
		t.Fatalf("declaration=%+v", declaration)
	}
	if declaration.HolidayData != "sales.holiday" {
		t.Fatalf("holiday data=%q", declaration.HolidayData)
	}
}

// TestResolveRefusesAnUndeclaredCalendar checks that a missing calendar is
// reported instead of defaulted.
func TestResolveRefusesAnUndeclaredCalendar(t *testing.T) {
	if _, err := businesscalendar.Resolve(resolveWorld(), "sales.missing"); err == nil {
		t.Fatal("an undeclared calendar was resolved")
	}
}

// TestBuildCombinesDeclarationAndHolidays checks that the calendar is built from
// the declaration and the holiday dates.
func TestBuildCombinesDeclarationAndHolidays(t *testing.T) {
	declaration, err := businesscalendar.Resolve(resolveWorld(), "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	calendar, err := declaration.Build([]time.Time{day(t, "2026-05-02")}, testLocale(t))
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := calendar.Excludes(day(t, "2026-05-02"))
	if err != nil {
		t.Fatal(err)
	}
	if !excluded {
		t.Fatal("the declared holiday was not excluded")
	}
	if len(calendar.HolidaySet()) != 1 {
		t.Fatalf("holiday set=%v", calendar.HolidaySet())
	}
}

// TestPeriodCalendarRequiresADeclaration checks that a period without a calendar
// is refused rather than resolved to another period's boundary.
func TestPeriodCalendarRequiresADeclaration(t *testing.T) {
	world := resolveWorld()
	world.Semantics[0].Periods[0].Calendar = ""
	if _, err := businesscalendar.PeriodCalendar(world, "sales.month"); err == nil {
		t.Fatal("a period without a calendar was resolved")
	}
	declared, err := businesscalendar.PeriodCalendar(resolveWorld(), "sales.month")
	if err != nil {
		t.Fatal(err)
	}
	if declared != "sales.fiscal" {
		t.Fatalf("calendar=%q", declared)
	}
}

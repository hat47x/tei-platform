package businesscalendar_test

import (
	"testing"

	"tei/internal/businesscalendar"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// calendarDomainWorld declares one holiday Data whose calendar identity type
// closes the set of calendars. A second calendar Data uses a type without values,
// so the open case is covered too.
func calendarDomainWorld(values []string) ir.World {
	types := []semantic.Type{
		{ID: "std.code", Symbol: "code", Kind: semantic.TypePrimitive},
		{ID: "std.date", Symbol: "date", Kind: semantic.TypePrimitive},
	}
	if values != nil {
		types = append(types, semantic.Type{ID: "sales.calendar_id", Symbol: "calendar_id", Kind: semantic.TypeAlias, Underlying: "std.code", Values: values})
	} else {
		types = append(types, semantic.Type{ID: "sales.calendar_id", Symbol: "calendar_id", Kind: semantic.TypeAlias, Underlying: "std.code"})
	}
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID:    "std",
				Types: types,
			},
			{
				ID: "sales",
				Data: []semantic.Data{
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
				},
			},
		},
	}
}

func calendarDomainBinding(t *testing.T, world ir.World) businesscalendar.HolidayBinding {
	t.Helper()
	declaration, err := businesscalendar.Resolve(world, "sales.tokyo")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

// TestCalendarDomainReadsTheDeclaredValues checks that the domain comes from the
// identity attribute's type, so a declaration can close the set of calendars.
func TestCalendarDomainReadsTheDeclaredValues(t *testing.T) {
	world := calendarDomainWorld([]string{"sales.tokyo", "sales.osaka"})
	domain, err := calendarDomainBinding(t, world).CalendarDomain(world)
	if err != nil {
		t.Fatal(err)
	}
	if !domain.Declared || len(domain.Values) != 2 {
		t.Fatalf("domain=%+v", domain)
	}
}

// TestCalendarDomainLeavesAnOpenTypeOpen checks that a type without declared values
// is not read as "no calendar is allowed".
func TestCalendarDomainLeavesAnOpenTypeOpen(t *testing.T) {
	world := calendarDomainWorld(nil)
	domain, err := calendarDomainBinding(t, world).CalendarDomain(world)
	if err != nil {
		t.Fatal(err)
	}
	if domain.Declared {
		t.Fatalf("domain=%+v", domain)
	}
}

// TestCalendarDomainReportsAnUnknownIdentity checks that a row whose identity is
// outside the declared values is reported. X-TEA compared a code string against a
// "code;date" list entry (D:Session.java L786), so a code no calendar declared
// still matched and the mistake was invisible.
func TestCalendarDomainReportsAnUnknownIdentity(t *testing.T) {
	world := calendarDomainWorld([]string{"sales.tokyo", "sales.osaka"})
	domain, err := calendarDomainBinding(t, world).CalendarDomain(world)
	if err != nil {
		t.Fatal(err)
	}
	if err := domain.Validate(0, "sales.tokyo"); err != nil {
		t.Fatalf("a declared identity was refused: %v", err)
	}
	if err := domain.Validate(1, "sales.kyoto"); err == nil {
		t.Fatal("an undeclared identity was accepted")
	}
	if err := domain.Validate(2, ""); err == nil {
		t.Fatal("an empty identity was accepted")
	}
}

// TestCalendarDomainAcceptsAnyIdentityWhenOpen checks that an open domain
// validates nothing, so the check does not invent a closed set.
func TestCalendarDomainAcceptsAnyIdentityWhenOpen(t *testing.T) {
	world := calendarDomainWorld(nil)
	domain, err := calendarDomainBinding(t, world).CalendarDomain(world)
	if err != nil {
		t.Fatal(err)
	}
	if err := domain.Validate(0, "sales.anything"); err != nil {
		t.Fatalf("an open domain refused an identity: %v", err)
	}
}

// TestCalendarDomainIsEmptyForASingleCalendarData checks that a Data without a
// calendar identity has no domain and no error.
func TestCalendarDomainIsEmptyForASingleCalendarData(t *testing.T) {
	world := holidayWorld(businesscalendar.RoleHolidayDate)
	declaration, err := businesscalendar.Resolve(world, "sales.fiscal")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := binding.CalendarDomain(world)
	if err != nil {
		t.Fatal(err)
	}
	if domain.Declared || len(domain.Values) != 0 {
		t.Fatalf("domain=%+v", domain)
	}
	if err := domain.Validate(0, ""); err != nil {
		t.Fatalf("single calendar data refused a row: %v", err)
	}
}

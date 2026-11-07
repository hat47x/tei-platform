package businesscalendar

import (
	"fmt"
	"time"

	"tei/internal/core"
	"tei/internal/ir"
)

// Declaration is the part of the runtime calendar that comes from the semantic
// declaration. The holiday set comes from the Data the declaration names, and a
// caller supplies it, because a holiday is a business record and not a
// declaration.
type Declaration struct {
	CalendarID           core.CanonicalID
	FiscalYearStartMonth int
	WeekStarts           WeekStarts
	HolidayData          core.CanonicalID
}

// Resolve reads the calendar declaration from the world. It refuses a missing
// declaration rather than substituting a default: a derivation that cannot name
// the calendar it follows cannot answer a date question.
func Resolve(world ir.World, calendarID core.CanonicalID) (Declaration, error) {
	for _, namespace := range world.Semantics {
		for _, calendar := range namespace.Calendars {
			if calendar.ID != calendarID {
				continue
			}
			return Declaration{
				CalendarID:           calendar.ID,
				FiscalYearStartMonth: calendar.FiscalYearStartMonth,
				WeekStarts:           WeekStarts(calendar.WeekStarts),
				HolidayData:          calendar.BusinessDayExcludes,
			}, nil
		}
	}
	return Declaration{}, fmt.Errorf("calendar %s is not declared", calendarID)
}

// Build combines the declaration with the holiday dates read from the Data it
// names. The locale is required: without it a date could be read in the host's
// zone, and the answer would depend on where the program ran.
func (d Declaration) Build(holidays []time.Time, locale *time.Location) (Calendar, error) {
	calendar := Calendar{
		FiscalYearStartMonth: d.FiscalYearStartMonth,
		WeekStarts:           d.WeekStarts,
		Holidays:             append([]time.Time(nil), holidays...),
		Locale:               locale,
	}
	if err := calendar.validate(); err != nil {
		return Calendar{}, err
	}
	return calendar, nil
}

// CalendarFor resolves a calendar by ID and builds it from the given holidays.
func CalendarFor(world ir.World, calendarID core.CanonicalID, holidays []time.Time, locale *time.Location) (Calendar, error) {
	declaration, err := Resolve(world, calendarID)
	if err != nil {
		return Calendar{}, err
	}
	return declaration.Build(holidays, locale)
}

// PeriodCalendar returns the calendar a period declares. A period without a
// declared calendar is refused here, so a caller does not silently fall back to
// another period's boundary.
func PeriodCalendar(world ir.World, periodID core.CanonicalID) (core.CanonicalID, error) {
	for _, namespace := range world.Semantics {
		for _, period := range namespace.Periods {
			if period.ID != periodID {
				continue
			}
			if period.Calendar == "" {
				return "", fmt.Errorf("period %s does not declare a calendar", periodID)
			}
			return period.Calendar, nil
		}
	}
	return "", fmt.Errorf("period %s is not declared", periodID)
}

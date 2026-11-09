package businesscalendar

import (
	"time"

	"tei/internal/core"
	"tei/internal/ir"
)

// HolidayResolve reads the holiday dates one calendar declares, from the world
// and the stored rows.
//
// It joins the three steps a caller otherwise has to combine by hand: resolve
// the declaration, bind the holiday Data, and read the rows for this calendar.
// The calendar identity's declared values are checked here, so a row whose
// identity is outside the declaration is reported instead of silently losing the
// holiday.
//
// X-TEA did the whole of this in one place at start-up (D:Session.java L786,
// L1691, L1882): it read a table keyed by "code;date" into a list and compared
// codes as plain strings. The joined function keeps the same outcome for a
// correct declaration and makes an incorrect one visible.
func HolidayResolve(world ir.World, calendarID core.CanonicalID, rows []map[core.CanonicalID]string, locale *time.Location) ([]time.Time, error) {
	declaration, err := Resolve(world, calendarID)
	if err != nil {
		return nil, err
	}
	binding, err := declaration.BindHolidays(world)
	if err != nil {
		return nil, err
	}
	dates, err := binding.HolidayRowsForCalendar(calendarID, rows, locale)
	if err != nil {
		return nil, err
	}
	if binding.Calendar == "" {
		return dates, nil
	}
	domain, err := binding.CalendarDomain(world)
	if err != nil {
		return nil, err
	}
	// The domain check runs over the whole row set, not only the rows this
	// calendar took, so a misspelt identity on another calendar row is also
	// reported. Skipping that row silently is what made X-TEA code typos
	// invisible.
	for index, row := range rows {
		if err := domain.Validate(index, row[binding.Calendar]); err != nil {
			return nil, err
		}
	}
	return dates, nil
}

// CalendarForPeriod is the common case: resolve the calendar a period declares,
// read its holidays, and build the calendar a derivation uses.
func CalendarForPeriod(world ir.World, periodID core.CanonicalID, rows []map[core.CanonicalID]string, locale *time.Location) (Calendar, error) {
	calendarID, err := PeriodCalendar(world, periodID)
	if err != nil {
		return Calendar{}, err
	}
	dates, err := HolidayResolve(world, calendarID, rows, locale)
	if err != nil {
		return Calendar{}, err
	}
	declaration, err := Resolve(world, calendarID)
	if err != nil {
		return Calendar{}, err
	}
	return declaration.Build(dates, locale)
}

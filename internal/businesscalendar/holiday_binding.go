package businesscalendar

import (
	"fmt"
	"time"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// RoleHolidayDate is the attribute role a holiday Data must declare. The
// calendar names the Data; the Data names the date. Without the role the binder
// cannot tell which attribute holds the holiday, and guessing one would bind the
// wrong column.
const RoleHolidayDate = "holiday-date"

// RoleHolidayCalendar is the optional role that tells several calendars apart in
// one holiday Data. X-TEA did this with a calendar code column and a list keyed
// by "code;date" (D:Session.java L786, L1691). Without the role, the Data holds
// holidays for a single calendar and every row belongs to it.
const RoleHolidayCalendar = "holiday-calendar"

// HolidayBinding is the holiday Data a calendar names, resolved to the attributes
// the dates and the calendar identity come from.
type HolidayBinding struct {
	DataID core.CanonicalID
	Date   core.CanonicalID
	// Calendar is empty when the Data holds holidays for one calendar. When it
	// is set, rows are filtered by the calendar a declaration names, so two
	// calendars can share a table without taking each other's holidays.
	Calendar core.CanonicalID
}

// BindHolidays resolves the holiday Data a calendar declares and finds the
// attributes the dates and the calendar identity come from.
//
// X-TEA kept holidays in a table keyed by a calendar code and a date string, and
// read them into a list at start-up (D:Session.java L786, L1691, L1882). The
// calendar code was a string from the data and the date was whatever the table
// held, so a holiday in another format was not excluded and nothing reported it.
// Here the calendar names the Data, the Data names the date attribute by role,
// and a date that cannot be read is reported with its row.
//
// A calendar without a declared holiday Data has no holidays. That is not a
// default: the calendar says a business day excludes nothing, which is a
// declaration a reader can see.
func (d Declaration) BindHolidays(world ir.World) (HolidayBinding, error) {
	if d.HolidayData == "" {
		return HolidayBinding{}, nil
	}
	data, ok := dataByID(world, d.HolidayData)
	if !ok {
		return HolidayBinding{}, fmt.Errorf("holiday data %s is not declared", d.HolidayData)
	}
	binding := HolidayBinding{DataID: d.HolidayData}
	for _, attribute := range data.Attributes {
		switch attribute.Role {
		case RoleHolidayDate:
			binding.Date = attribute.ID
		case RoleHolidayCalendar:
			binding.Calendar = attribute.ID
		}
	}
	if binding.Date == "" {
		return HolidayBinding{}, fmt.Errorf("holiday data %s does not declare %s", d.HolidayData, RoleHolidayDate)
	}
	return binding, nil
}

// HolidayRows turns stored rows into holiday dates. Each row is keyed by
// attribute ID, as the reference store keeps it.
//
// A binding whose Data declares a calendar identity needs the calendar to read;
// this method refuses then, so a caller does not silently take every calendar's
// holidays. Use HolidayRowsForCalendar in that case.
func (b HolidayBinding) HolidayRows(rows []map[core.CanonicalID]string, locale *time.Location) ([]time.Time, error) {
	if b.Calendar != "" {
		return nil, fmt.Errorf("%w: holiday data %s holds several calendars; name the calendar to read", ErrInvalidCalendar, b.DataID)
	}
	return b.readRows(rows, locale)
}

// HolidayRowsForCalendar turns the stored rows of one calendar into holiday
// dates. A row for another calendar is skipped, because it is that calendar's
// holiday and not this one's.
func (b HolidayBinding) HolidayRowsForCalendar(calendarID core.CanonicalID, rows []map[core.CanonicalID]string, locale *time.Location) ([]time.Time, error) {
	if b.Calendar == "" {
		return b.readRows(rows, locale)
	}
	if calendarID == "" {
		return nil, fmt.Errorf("%w: the calendar to read is not named", ErrInvalidCalendar)
	}
	filtered := make([]map[core.CanonicalID]string, 0, len(rows))
	for _, row := range rows {
		if row[b.Calendar] == string(calendarID) {
			filtered = append(filtered, row)
		}
	}
	return b.readRows(filtered, locale)
}

func (b HolidayBinding) readRows(rows []map[core.CanonicalID]string, locale *time.Location) ([]time.Time, error) {
	if locale == nil {
		return nil, fmt.Errorf("%w: the time zone is not declared", ErrInvalidCalendar)
	}
	out := make([]time.Time, 0, len(rows))
	for index, row := range rows {
		value := row[b.Date]
		if value == "" {
			return nil, fmt.Errorf("holiday row %d: %s is empty", index+1, b.Date)
		}
		parsed, err := time.ParseInLocation("2006-01-02", value, locale)
		if err != nil {
			return nil, fmt.Errorf("holiday row %d: %s: %q is not a date", index+1, b.Date, value)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func dataByID(world ir.World, dataID core.CanonicalID) (semantic.Data, bool) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID == dataID {
				return data, true
			}
		}
	}
	return semantic.Data{}, false
}

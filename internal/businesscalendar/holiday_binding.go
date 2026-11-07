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

// HolidayBinding is the holiday Data a calendar names, resolved to the
// attribute the dates come from.
type HolidayBinding struct {
	DataID core.CanonicalID
	Date   core.CanonicalID
}

// BindHolidays resolves the holiday Data a calendar declares and finds the
// attribute the dates come from.
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
	for _, attribute := range data.Attributes {
		if attribute.Role == RoleHolidayDate {
			return HolidayBinding{DataID: d.HolidayData, Date: attribute.ID}, nil
		}
	}
	return HolidayBinding{}, fmt.Errorf("holiday data %s does not declare %s", d.HolidayData, RoleHolidayDate)
}

// HolidayRows turns stored rows into holiday dates. Each row is keyed by
// attribute ID, as the reference store keeps it.
func (b HolidayBinding) HolidayRows(rows []map[core.CanonicalID]string, locale *time.Location) ([]time.Time, error) {
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

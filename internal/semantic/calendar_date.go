package semantic

import (
	"fmt"
	"time"
)

// CalendarDate is a day of the calendar without a time of day and without a
// time zone: the value of std.date. Two calendar dates are ordered the same
// way everywhere, which a moment in time is not until a zone is stated.
//
// The zero value is not a date.
type CalendarDate struct {
	Year  int
	Month int
	Day   int
}

// IsZero reports whether d is the zero value, which is not a date.
func (d CalendarDate) IsZero() bool { return d == CalendarDate{} }

// ParseCalendarDate reads a date written as YYYY-MM-DD and nothing else: no
// time, no zone, no other separator, no missing leading zero. A day the
// calendar does not have, such as 2026-02-30, is refused instead of being
// moved to a nearby day.
func ParseCalendarDate(text string) (CalendarDate, error) {
	refuse := func() (CalendarDate, error) {
		return CalendarDate{}, fmt.Errorf("%q is not a calendar date; write it as YYYY-MM-DD, for example 2026-04-01", text)
	}
	if len(text) != len("2006-01-02") {
		return refuse()
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		if i == 4 || i == 7 {
			if c != '-' {
				return refuse()
			}
		} else if c < '0' || c > '9' {
			return refuse()
		}
	}
	parsed, err := time.Parse("2006-01-02", text)
	if err != nil || parsed.Year() == 0 {
		return refuse()
	}
	return CalendarDate{Year: parsed.Year(), Month: int(parsed.Month()), Day: parsed.Day()}, nil
}

// Compare returns -1 when d is before other, 0 when they are the same day and
// 1 when d is after other.
func (d CalendarDate) Compare(other CalendarDate) int {
	for _, pair := range [][2]int{{d.Year, other.Year}, {d.Month, other.Month}, {d.Day, other.Day}} {
		switch {
		case pair[0] < pair[1]:
			return -1
		case pair[0] > pair[1]:
			return 1
		}
	}
	return 0
}

func (d CalendarDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

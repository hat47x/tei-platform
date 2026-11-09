// Package businesscalendar derives dates from a declared calendar. It is a
// reference implementation: it computes from the declaration and the holiday set
// it is given, and it does not read a database, a clock, or a locale.
//
// X-TEA's Session answered these questions from system variables and a holiday
// table read at run time (D:Session.java L1444–1980), and getDaysBetweenDates
// chose the counting rule with an integer argument (同 L1761). Here the
// declaration and the named rule are the inputs, so the same declaration and the
// same holiday set give the same answer on every run.
package businesscalendar

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// DayCount names how a span between two dates is counted. X-TEA's countType
// integer decided the meaning of the count (0 = elapsed, 1 = business, 2 =
// calendar, D:Session.java L1761–1830); here the count is the rule's name, so a
// caller does not pass an unexplained number.
type DayCount string

const (
	// DayCountElapsed is the difference between the two dates as a date count.
	DayCountElapsed DayCount = "elapsed"
	// DayCountCalendar counts every date from one to the other, both included.
	DayCountCalendar DayCount = "calendar"
	// DayCountBusiness counts the dates from one to the other, both included,
	// that the calendar does not exclude. X-TEA excluded only holidays, never a
	// weekend (同 L1803: offDateList だけを除く), so this rule does the same.
	DayCountBusiness DayCount = "business"
)

// WeekStarts names the first day of the week. The reference calendar only
// records it; date derivation does not depend on it, because X-TEA's business
// day count excluded holidays and not weekends.
type WeekStarts string

const (
	WeekStartsSunday WeekStarts = "sunday"
	WeekStartsMonday WeekStarts = "monday"
)

// Calendar is the runtime form of a declared calendar. It is resolved from the
// semantic declaration by the caller, so a derivation cannot invent a fiscal
// year start or a holiday set.
type Calendar struct {
	// FiscalYearStartMonth is 1..12. Zero is refused, not defaulted to April or
	// January: a boundary the declaration does not name is not derivable
	// (CALENDAR_FISCAL_YEAR_START_REQUIRED).
	FiscalYearStartMonth int
	WeekStarts           WeekStarts
	// Holidays are the dates the calendar excludes from business days.
	Holidays []time.Time
	// Locale is the time zone the dates are read in. An unset locale is refused,
	// so a run does not depend on the host's zone.
	Locale *time.Location
}

// ErrInvalidCalendar reports a calendar that cannot derive a date. It is a
// configuration failure, not a business result.
var ErrInvalidCalendar = errors.New("calendar cannot derive a date")

func (c Calendar) validate() error {
	if c.FiscalYearStartMonth < 1 || c.FiscalYearStartMonth > 12 {
		return fmt.Errorf("%w: fiscal year start month %d is outside 1..12", ErrInvalidCalendar, c.FiscalYearStartMonth)
	}
	switch c.WeekStarts {
	case WeekStartsSunday, WeekStartsMonday:
	case "":
		return fmt.Errorf("%w: the week start is not declared", ErrInvalidCalendar)
	default:
		return fmt.Errorf("%w: unsupported week start %q", ErrInvalidCalendar, c.WeekStarts)
	}
	if c.Locale == nil {
		return fmt.Errorf("%w: the time zone is not declared", ErrInvalidCalendar)
	}
	return nil
}

// normalize returns the date at midnight in the calendar's zone, so a caller's
// clock time never changes a date-only answer.
func (c Calendar) normalize(date time.Time) time.Time {
	year, month, day := date.In(c.Locale).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, c.Locale)
}

func (c Calendar) holidaySet() map[string]bool {
	out := make(map[string]bool, len(c.Holidays))
	for _, holiday := range c.Holidays {
		out[c.normalize(holiday).Format("2006-01-02")] = true
	}
	return out
}

// Excludes reports whether a date is excluded as a holiday. It is separate from
// IsBusinessDay so a caller can tell "a holiday" from "a configuration fault".
func (c Calendar) Excludes(date time.Time) (bool, error) {
	if err := c.validate(); err != nil {
		return false, err
	}
	return c.holidaySet()[c.normalize(date).Format("2006-01-02")], nil
}

// IsBusinessDay reports whether the date is not excluded by the calendar.
func (c Calendar) IsBusinessDay(date time.Time) (bool, error) {
	excluded, err := c.Excludes(date)
	if err != nil {
		return false, err
	}
	return !excluded, nil
}

// DaysBetween counts the span from one date to another under the named rule.
// The count is inclusive of both dates for the calendar and business rules, and
// a negative span returns a negative count, so a caller does not swap arguments.
func (c Calendar) DaysBetween(from, thru time.Time, count DayCount) (int, error) {
	if err := c.validate(); err != nil {
		return 0, err
	}
	switch count {
	case DayCountElapsed, DayCountCalendar, DayCountBusiness:
	default:
		return 0, fmt.Errorf("%w: unsupported day count %q", ErrInvalidCalendar, count)
	}
	start, end := c.normalize(from), c.normalize(thru)
	if start.Equal(end) {
		return 0, nil
	}
	step := 1
	if end.Before(start) {
		step = -1
	}
	if count == DayCountElapsed {
		// Subtract dates, not clock instants: a daylight-saving hour would
		// otherwise change the answer.
		days := 0
		for current := start; !current.Equal(end); current = current.AddDate(0, 0, step) {
			days++
		}
		return days * step, nil
	}
	days := 0
	for current := start; !current.Equal(end); current = current.AddDate(0, 0, step) {
		if count == DayCountBusiness {
			excluded, err := c.Excludes(current)
			if err != nil {
				return 0, err
			}
			if excluded {
				continue
			}
		}
		days++
	}
	return days * step, nil
}

// AddBusinessDays returns the date reached by moving the given number of business
// days from the start. Zero returns the start when it is a business day, and the
// next business day otherwise, so the result is always a business day.
func (c Calendar) AddBusinessDays(from time.Time, days int) (time.Time, error) {
	if err := c.validate(); err != nil {
		return time.Time{}, err
	}
	current := c.normalize(from)
	if days == 0 {
		for {
			excluded, err := c.Excludes(current)
			if err != nil {
				return time.Time{}, err
			}
			if !excluded {
				return current, nil
			}
			current = current.AddDate(0, 0, 1)
		}
	}
	step := 1
	if days < 0 {
		step, days = -1, -days
	}
	for days > 0 {
		current = current.AddDate(0, 0, step)
		excluded, err := c.Excludes(current)
		if err != nil {
			return time.Time{}, err
		}
		if !excluded {
			days--
		}
	}
	return current, nil
}

// FiscalYear returns the label of the fiscal year the date belongs to. The label
// is the calendar year the fiscal year ends in, so a date in March under an
// April start belongs to the following year's label.
func (c Calendar) FiscalYear(date time.Time) (int, error) {
	if err := c.validate(); err != nil {
		return 0, err
	}
	normalized := c.normalize(date)
	year, month, _ := normalized.Date()
	if int(month) >= c.FiscalYearStartMonth {
		return year + 1, nil
	}
	return year, nil
}

// FiscalYearStart returns the first date of the fiscal year the date belongs to.
func (c Calendar) FiscalYearStart(date time.Time) (time.Time, error) {
	label, err := c.FiscalYear(date)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(label-1, time.Month(c.FiscalYearStartMonth), 1, 0, 0, 0, 0, c.Locale), nil
}

// HolidaySet returns the holiday dates sorted by date, so a diagnostic can report
// the set a derivation used without reading the declaration again.
func (c Calendar) HolidaySet() []time.Time {
	out := make([]time.Time, 0, len(c.Holidays))
	for _, holiday := range c.Holidays {
		out = append(out, c.normalize(holiday))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

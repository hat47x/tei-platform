package businesscalendar

import (
	"fmt"
	"time"
)

// PeriodKey names the period a business date belongs to. The key is the pair a
// closure refers to: a fiscal year label and a month sequence inside it. X-TEA
// answered the same question with getFYearOfDate and getMSeqOfDate
// (D:Session.java L2371–2519), whose month sequence used a start month and a
// last-day setting read from system variables. Here the start month comes from
// the declared calendar and the last day is not a setting: a date belongs to the
// month its date field names.
type PeriodKey struct {
	// FiscalYear is the label of the fiscal year the date belongs to, as
	// Calendar.FiscalYear returns it.
	FiscalYear int
	// MonthSequence is 1..12 inside the fiscal year: 1 is the start month.
	MonthSequence int
	// CalendarMonth is 1..12 of the civil calendar month the date names. It is
	// kept beside the sequence because the sequence alone is ambiguous when the
	// fiscal year does not start in January.
	CalendarMonth int
	// ClosingSegment is 0 for a whole-month period, and 1 or 2 when a period
	// declares a closing day. A zero value keeps the key of a month period
	// unchanged, so an existing declaration does not have to say "no closing".
	ClosingSegment ClosingSegment
}

// String renders the key in the form a period declaration can use as a value. A
// month period renders as "YYYY-NN"; a period with a closing day adds the segment,
// so the two halves of one month are distinguishable.
func (k PeriodKey) String() string {
	if k.ClosingSegment == 0 {
		return fmt.Sprintf("%04d-%02d", k.FiscalYear, k.MonthSequence)
	}
	return fmt.Sprintf("%04d-%02d-%d", k.FiscalYear, k.MonthSequence, k.ClosingSegment)
}

// PeriodKeyFor returns the period key of a business date under the declared
// calendar.
//
// X-TEA shifted the month sequence by one when the date fell after the last day
// of a closing month (getMSeqOfDate, D:Session.java L2476–2484). That rule made a
// date belong to the next month when a "last day" setting said so, and the same
// function changed the fiscal year for the same reason. This derivation does not
// carry that rule: a date belongs to the month it names, and a period boundary
// narrower than a month has to be declared as its own period.
func (c Calendar) PeriodKeyFor(date time.Time) (PeriodKey, error) {
	if err := c.validate(); err != nil {
		return PeriodKey{}, err
	}
	normalized := c.normalize(date)
	_, month, _ := normalized.Date()
	fiscalYear, err := c.FiscalYear(normalized)
	if err != nil {
		return PeriodKey{}, err
	}
	sequence := int(month) - c.FiscalYearStartMonth + 1
	if sequence <= 0 {
		sequence += 12
	}
	return PeriodKey{
		FiscalYear:    fiscalYear,
		MonthSequence: sequence,
		CalendarMonth: int(month),
	}, nil
}

// PeriodStart returns the first date of the period the key names. It is the
// inverse of PeriodKeyFor for keys the calendar can produce, which lets a caller
// turn a closure record back into a date range.
func (c Calendar) PeriodStart(key PeriodKey) (time.Time, error) {
	if err := c.validate(); err != nil {
		return time.Time{}, err
	}
	if key.MonthSequence < 1 || key.MonthSequence > 12 {
		return time.Time{}, fmt.Errorf("%w: month sequence %d is outside 1..12", ErrInvalidCalendar, key.MonthSequence)
	}
	month := c.FiscalYearStartMonth + key.MonthSequence - 1
	year := key.FiscalYear - 1
	if month > 12 {
		month -= 12
		year++
	}
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, c.Locale), nil
}

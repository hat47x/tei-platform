package businesscalendar

import (
	"fmt"
	"time"
)

// ClosingPeriod is a period narrower than a month: a declared closing day splits
// every month into two segments, and a business date belongs to the segment its
// day falls in.
//
// X-TEA read a last-day setting and moved a date after that day into the next
// month's sequence (getMSeqOfDate, D:Session.java L2471–2484), which also moved
// the fiscal year for the same reason. That made a date belong to a month it did
// not name, and nothing in the model said the boundary existed. Here the closing
// day is a declaration, the month is not moved, and the segment says which side
// of the closing the date is on.
type ClosingPeriod struct {
	Calendar   Calendar
	ClosingDay int
}

// MaxClosingDay is the largest closing day a period may declare. Days 29–31 are
// refused because February and the 30-day months have no such day, so a boundary
// on them would not exist in every month.
const MaxClosingDay = 28

// Validate checks the closing day against the declared calendar.
func (p ClosingPeriod) Validate() error {
	if p.ClosingDay < 1 || p.ClosingDay > MaxClosingDay {
		return fmt.Errorf("%w: closing day %d is outside 1..%d", ErrInvalidCalendar, p.ClosingDay, MaxClosingDay)
	}
	return p.Calendar.validate()
}

// ClosingSegment is which side of the closing day a date is on.
type ClosingSegment int

const (
	// SegmentBefore is the part of the month up to and including the closing day.
	SegmentBefore ClosingSegment = 1
	// SegmentAfter is the part of the month after the closing day.
	SegmentAfter ClosingSegment = 2
)

// KeyFor returns the period key of a business date, with the segment the date
// falls in. The fiscal year and the month sequence are the ones the date names:
// a date after the closing day is still in the same month.
func (p ClosingPeriod) KeyFor(date time.Time) (PeriodKey, error) {
	if err := p.Validate(); err != nil {
		return PeriodKey{}, err
	}
	key, err := p.Calendar.PeriodKeyFor(date)
	if err != nil {
		return PeriodKey{}, err
	}
	normalized := p.Calendar.normalize(date)
	if normalized.Day() <= p.ClosingDay {
		key.ClosingSegment = SegmentBefore
	} else {
		key.ClosingSegment = SegmentAfter
	}
	return key, nil
}

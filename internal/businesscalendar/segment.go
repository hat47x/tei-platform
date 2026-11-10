package businesscalendar

import (
	"fmt"
	"time"
)

// SegmentFor returns which half of the month a date falls in when the period
// declares a closing day.
//
// The segment is decided by the day of the month alone. X-TEA moved a date after
// the closing day into the next month's sequence (getMSeqOfDate, D:Session.java
// L2471–2484), so the same question also changed the month and the fiscal year.
// Here the month does not move, so the day of the date is the whole answer and a
// caller does not need the calendar to decide the segment.
func SegmentFor(date time.Time, closingDay int) (ClosingSegment, error) {
	if closingDay < 1 || closingDay > MaxClosingDay {
		return 0, fmt.Errorf("%w: closing day %d is outside 1..%d", ErrInvalidCalendar, closingDay, MaxClosingDay)
	}
	if date.Day() <= closingDay {
		return SegmentBefore, nil
	}
	return SegmentAfter, nil
}

// SegmentOf is SegmentFor on an already-declared period.
func (p ClosingPeriod) SegmentOf(date time.Time) (ClosingSegment, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	return SegmentFor(date, p.ClosingDay)
}

// String renders the segment as the digit a period key uses.
func (s ClosingSegment) String() string {
	switch s {
	case SegmentBefore:
		return "1"
	case SegmentAfter:
		return "2"
	default:
		return ""
	}
}

// Valid reports whether the segment is one of the declared halves.
func (s ClosingSegment) Valid() bool {
	return s == SegmentBefore || s == SegmentAfter
}

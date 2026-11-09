package applicationview

import (
	"fmt"
	"time"
)

// PeriodScope is the unit a closure applies to: a whole month, or one half of a
// month when the period declares a closing day.
//
// X-TEA decided a closure by a month sequence that a closing day could shift
// (getMSeqOfDate, D:Session.java L2471–2484), so the closed unit and the date's
// month disagreed after the closing day. Here the month does not move and the
// scope names the half explicitly, so a closure covers exactly what its key says.
type PeriodScope struct {
	PeriodID string
	// Segment is 1 for the first half, 2 for the second half, and 0 for a
	// whole-month period.
	Segment int
}

// String renders the scope. A whole-month scope is the period ID alone, so an
// existing state store keyed by period ID keeps working.
func (s PeriodScope) String() string {
	if s.Segment == 0 {
		return s.PeriodID
	}
	return fmt.Sprintf("%s#%d", s.PeriodID, s.Segment)
}

// ScopeFor returns the scope a period closure applies to for one business date.
//
// A period without a closing day is closed as a whole month, whatever the date.
// A period with a closing day is closed by half, so closing the first half does
// not refuse a command whose date is in the second half.
func ScopeFor(period CommandPeriodRef, businessDate string) (PeriodScope, error) {
	scope := PeriodScope{PeriodID: string(period.ID)}
	if period.ClosingDay == 0 {
		return scope, nil
	}
	if businessDate == "" {
		return PeriodScope{}, fmt.Errorf("period %s declares closing day %d but the business date is not named", period.ID, period.ClosingDay)
	}
	date, err := time.Parse("2006-01-02", businessDate)
	if err != nil {
		return PeriodScope{}, fmt.Errorf("period %s business date %q is not a date", period.ID, businessDate)
	}
	if date.Day() <= period.ClosingDay {
		scope.Segment = 1
	} else {
		scope.Segment = 2
	}
	return scope, nil
}

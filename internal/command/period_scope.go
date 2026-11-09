package command

import (
	"context"

	"tei/internal/applicationview"
	"tei/internal/core"
)

// ScopePeriodState is the optional extension a period state implements when
// periods can be narrower than a month.
//
// The base PeriodState answers for a whole period; this one answers for one
// scope, which is the period and half a closing day splits it into. A provider
// that keeps a closure per month implements only the base interface, and a period
// with a closing day then refuses rather than being read as a whole month: a
// half-month closure is not the same meaning as a month closure.
type ScopePeriodState interface {
	ClosedScope(ctx context.Context, scope string) (bool, error)
}

// scopeClosed asks the configured state whether the scope is closed. It reports
// whether the state could answer at all, so a caller does not mistake "no such
// state" for "open".
func scopeClosed(ctx context.Context, state PeriodState, scope applicationview.PeriodScope) (closed bool, answered bool, err error) {
	if scope.Segment == 0 {
		closed, err := state.Closed(ctx, core.CanonicalID(scope.PeriodID))
		return closed, true, err
	}
	scoped, ok := state.(ScopePeriodState)
	if !ok {
		return false, false, nil
	}
	closed, err = scoped.ClosedScope(ctx, scope.String())
	return closed, true, err
}

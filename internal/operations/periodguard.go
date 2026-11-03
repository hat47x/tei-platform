package operations

import (
	"context"

	"tei/internal/core"
)

// GuardCapability reports whether the capability a write is bound to may run in
// the given period states. A refusal is returned as a value, not as an error: it
// is a business outcome the caller reports to the user.
//
// The check lives here, not in the driver, because the operation surface owns the
// binding between a standard operation and the capability that executes it. A
// capability without a derived operation is not guarded.
func GuardCapability(surface OperationSurface, states map[string]PeriodState, capabilityID core.CanonicalID) (PeriodRejection, bool) {
	for _, operation := range surface.Operations {
		if operation.Binding != string(capabilityID) || operation.Period == "" {
			continue
		}
		rejections := CheckPeriods(OperationSurface{Operations: []Operation{operation}}, states)
		if len(rejections) == 0 {
			continue
		}
		return rejections[0], true
	}
	return PeriodRejection{}, false
}

// PeriodStateSource supplies the runtime state of the periods an application
// uses. The state is not accepted meaning, so the caller provides the source:
// an operator record today, an external system later. A source that returns no
// state for a period leaves that period unguarded.
type PeriodStateSource interface {
	PeriodStates(context.Context, string) (map[string]PeriodState, error)
}

// GuardCapabilityWith asks the source for the states and then guards the
// capability. A source error is returned as an error: the guard could not be
// evaluated, which is different from a refusal.
func GuardCapabilityWith(ctx context.Context, source PeriodStateSource, surface OperationSurface, applicationID string, capabilityID core.CanonicalID) (PeriodRejection, bool, error) {
	if source == nil {
		return PeriodRejection{}, false, nil
	}
	states, err := source.PeriodStates(ctx, applicationID)
	if err != nil {
		return PeriodRejection{}, false, err
	}
	rejection, refused := GuardCapability(surface, states, capabilityID)
	return rejection, refused, nil
}

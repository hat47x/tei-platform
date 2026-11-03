package driver

import (
	"tei/internal/core"
	"tei/internal/operations"
)

// GuardPeriods reports whether a capability may run in the given period states.
// A refusal is returned as a value, not as an error: it is a business outcome
// the caller reports to the user, not an execution failure.
//
// The check reads the operation surface, because the surface is where a write
// operation records the period that governs it. A capability without a derived
// operation is not guarded: the surface does not know it.
func GuardPeriods(surface operations.OperationSurface, states map[string]operations.PeriodState, capabilityID core.CanonicalID) (operations.PeriodRejection, bool) {
	for _, operation := range surface.Operations {
		if operation.Binding != string(capabilityID) || operation.Period == "" {
			continue
		}
		rejections := operations.CheckPeriods(operationSurfaceOf(operation), states)
		if len(rejections) == 0 {
			continue
		}
		return rejections[0], true
	}
	return operations.PeriodRejection{}, false
}

// operationSurfaceOf wraps one operation so the shared period check can be used
// without rebuilding the surface.
func operationSurfaceOf(operation operations.Operation) operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{operation}}
}

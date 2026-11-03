package operations

import "tei/internal/core"

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

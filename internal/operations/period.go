package operations

import "fmt"

// PeriodState is the runtime state of one period. The state is not part of the
// accepted meaning: a period is opened and closed by execution, so the caller
// supplies it.
type PeriodState string

const (
	PeriodOpen    PeriodState = "open"
	PeriodClosing PeriodState = "closing"
	PeriodClosed  PeriodState = "closed"
)

// PeriodRejection records an operation that may not run because its period is
// not open. It is a refusal, not an execution error.
type PeriodRejection struct {
	Operation string      `json:"operation"`
	Period    string      `json:"period"`
	State     PeriodState `json:"state"`
	Detail    string      `json:"detail"`
}

// CheckPeriods reports the write operations of a surface that may not run in
// the given period states. An operation is refused when its period is not open
// and the period declares that it rejects that operation once closed. The check
// does not execute anything and does not mutate the surface.
func CheckPeriods(surface OperationSurface, states map[string]PeriodState) []PeriodRejection {
	var rejections []PeriodRejection
	for _, operation := range surface.Operations {
		if operation.Period == "" || len(operation.PeriodRejects) == 0 {
			continue
		}
		state, ok := states[operation.Period]
		if !ok {
			continue
		}
		if state == PeriodOpen {
			continue
		}
		if !periodRejectsOperation(operation.PeriodRejects, operation.Kind) {
			continue
		}
		rejections = append(rejections, PeriodRejection{
			Operation: operation.ID,
			Period:    operation.Period,
			State:     state,
			Detail:    fmt.Sprintf("the period %s is %s, so %s is refused", operation.Period, state, operation.Kind),
		})
	}
	return rejections
}

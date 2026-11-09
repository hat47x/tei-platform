package hook

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// applyReachableRows applies a row-scoped hook to each row of the runner's row set.
//
// The phase is recorded as executed only when every row passed. A refusal stops at
// the row that refused and the phase is recorded as refused with the row named in
// the reason, so a partly-applied page never reads as a passed phase.
//
// A row-scoped hook that also declares `uses` is refused here: the declared call
// would have to be made once per row and this path does not do that yet. Refusing
// is the fail-closed choice, because calling once for the whole page would keep the
// promise for one row and not the others.
func (r Runner) applyReachableRows(ctx context.Context, report *PhaseReport, applicationID, patternID core.CanonicalID, phase string, declaration *application.Hook) error {
	if len(declaration.Uses) > 0 {
		return &Error{
			Code:    "HOOK_SCOPE_ROW_USES_UNSUPPORTED",
			Message: fmt.Sprintf("hook %s declares scope row with uses, which this path does not call per row", phase),
		}
	}
	for index, row := range r.Rows {
		// The row's own values are used, and only the reads the declaration names.
		// A pattern-level value is not substituted for a read the row lacks: that
		// value belongs to no row.
		reads := make(map[core.CanonicalID]string, len(declaration.Reads))
		for _, attribute := range declaration.Reads {
			if value, ok := row[attribute]; ok {
				reads[attribute] = value
			}
		}
		invocation := Invocation{
			Application: applicationID,
			Pattern:     patternID,
			Phase:       phase,
			Reads:       reads,
		}
		produced, err := r.RunWithDeclaredValues(ctx, invocation)
		if err == nil {
			if len(produced) > 0 {
				if report.Produced == nil {
					report.Produced = map[string]Produced{}
				}
				report.Produced[phase] = produced
			}
			continue
		}
		var warning *Warning
		if errors.As(err, &warning) {
			// A warn row does not stop the page; the phase is recorded as warned so
			// a reader sees that part of it failed.
			report.Warned = append(report.Warned, phase)
			continue
		}
		report.Refused = phase
		var notSatisfied *NotSatisfied
		if errors.As(err, &notSatisfied) {
			report.Outcome = notSatisfied.Outcome
		}
		report.Reasons[phase] = fmt.Sprintf("row %d: %v", index, err)
		return &RowRefusal{Row: index, Err: err}
	}
	report.Executed = append(report.Executed, phase)
	return nil
}

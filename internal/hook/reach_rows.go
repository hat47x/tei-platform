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
// A row-scoped hook that also declares `uses` makes its calls once per row,
// through the same per-row path RunRows uses. The promise is the same in both
// entry points; only the bookkeeping differs.
func (r Runner) applyReachableRows(ctx context.Context, report *PhaseReport, applicationID, patternID core.CanonicalID, phase string, declaration *application.Hook) error {
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
		// A `uses` promise is one call per row, made with that row's values.
		if len(declaration.Uses) > 0 {
			if _, callErr := r.RunRowCalls(ctx, invocation, declaration, r.Rows[index:index+1]); callErr != nil {
				report.Refused = phase
				report.Reasons[phase] = fmt.Sprintf("row %d: %v", index, callErr)
				return &RowRefusal{Row: index, Err: callErr}
			}
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
		if outcome, ok := OutcomeOf(err); ok {
			report.Outcome = outcome
		}
		report.Reasons[phase] = fmt.Sprintf("row %d: %v", index, err)
		return &RowRefusal{Row: index, Err: err}
	}
	report.Executed = append(report.Executed, phase)
	return nil
}

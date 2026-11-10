package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// RunRowCalls makes the calls a row-scoped hook declares in `uses`, once for the
// row that is being decided.
//
// The record path keeps a `uses` promise through RunDeclaredCalls, once per
// phase, because a record hook runs once. A row-scoped hook runs once per row, so
// its promise is one call per row: calling once for the page would keep the
// promise for no particular row, and calling not at all would leave the
// declaration inert.
//
// The inputs for each row come from that row's values, through the binding the
// interaction command already declares. A parameter whose attribute the row did
// not supply is left out, so the Command boundary reports the missing input
// rather than receiving a value that belongs to another row.
//
// The returned outcomes are the record of what happened, in the declaration's
// order. A refused or failed call is returned as an error after the outcomes are
// collected, so a caller sees which rows and which capabilities were affected.
func (r Runner) RunRowCalls(ctx context.Context, invocation Invocation, declared *application.Hook, rows []map[core.CanonicalID]string) ([]RowCallOutcome, error) {
	if declared == nil || len(declared.Uses) == 0 {
		return nil, nil
	}
	pattern, err := r.findPatternForCalls(invocation.Application, invocation.Pattern)
	if err != nil {
		return nil, err
	}
	restricted := Declared{Inner: r.Caller, Hook: declared}
	outcomes := make([]RowCallOutcome, 0, len(rows))
	var firstErr error
	for index, row := range rows {
		reads := narrowTo(row, declared.Reads)
		for _, capability := range declared.Uses {
			outcome := RowCallOutcome{Row: index, Capability: capability}
			if _, resolveErr := CommandPatternFor(r.World, invocation.Application, capability); resolveErr != nil {
				outcome.Reason = resolveErr.Error()
				outcomes = append(outcomes, outcome)
				if firstErr == nil {
					firstErr = resolveErr
				}
				continue
			}
			bound := BindInputsFromReads(pattern, capability, reads)
			if len(bound) == 0 {
				outcome.Reason = ReasonNoInputs
				outcomes = append(outcomes, outcome)
				continue
			}
			if r.Caller == nil {
				outcome.Reason = ReasonNoCaller
				outcomes = append(outcomes, outcome)
				continue
			}
			if _, callErr := restricted.Call(ctx, capability, bound); callErr != nil {
				outcome.Error = callErr.Error()
				outcomes = append(outcomes, outcome)
				if firstErr == nil {
					firstErr = fmt.Errorf("row %d: capability %s: %w", index, capability, callErr)
				}
				continue
			}
			outcome.Called = true
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes, firstErr
}

// RowCallOutcome records one declared call made for one row.
type RowCallOutcome struct {
	// Row is the zero-based position of the row the call was made for.
	Row int `json:"row"`
	// Capability is the capability the hook declared.
	Capability core.CanonicalID `json:"capability"`
	// Called is true when the boundary was reached.
	Called bool `json:"called"`
	// Reason states why a declared capability was not called.
	Reason string `json:"reason,omitempty"`
	// Error is the boundary's refusal or failure.
	Error string `json:"error,omitempty"`
}

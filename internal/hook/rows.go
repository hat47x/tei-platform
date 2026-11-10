package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// RowInvocation carries the values of one row, so a hook declared with scope row
// is applied to each row rather than once to the whole result.
//
// X-TEA's event script received no row set: a hook written to decide per row ran
// once and the script author had to loop inside it. That put the repetition in the
// script, where the declaration could not see it. Here the repetition is the
// runner's, so the declared scope and the run agree.
type RowInvocation struct {
	Invocation
	// Row is the zero-based position of the row in the result. It is passed so a
	// refusal can name the row it stopped on.
	Row int
}

// RunRows applies a row-scoped hook to each row, in order, and stops at the first
// refusal.
//
// The values of each row are narrowed to the reads the declaration names, so a row
// never shows the hook an attribute it did not declare. A refusal names the row it
// stopped at and the rows after it are not visited, which is the same "stop where
// it failed" rule the sequence uses for phases.
//
// A hook whose scope is not row is refused: running a record-scoped hook per row
// would apply it more often than it was declared for.
func (r Runner) RunRows(ctx context.Context, invocation Invocation, rows []map[core.CanonicalID]string) error {
	_, declared, err := r.declaredHook(invocation)
	if err != nil {
		return err
	}
	// Only a row-scoped hook is applied per row. A record-scoped hook runs once,
	// through Run, and an unapplied scope is refused there rather than here: this
	// method exists for the scope that needs repetition.
	if declared.Scope != "row" {
		return &Error{
			Code:    "HOOK_SCOPE_NOT_ROW",
			Message: fmt.Sprintf("hook %s declares scope %q, so it is not applied per row", invocation.Phase, declared.Scope),
		}
	}
	if len(rows) == 0 {
		// No rows means nothing to apply the hook to. That is not a failure: a
		// browse result can be empty, and refusing it would turn an empty page
		// into an error.
		return nil
	}
	implementation := r.implementationFor(invocation.Phase)
	if implementation == nil {
		return &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no implementation is configured for this hook boundary"}
	}
	for index, row := range rows {
		rowInvocation := invocation
		rowInvocation.Reads = narrowTo(row, declared.Reads)
		// A `uses` promise is one call per row, made with that row's values. The
		// record path keeps the same promise through RunDeclaredCalls; rows differ
		// only in how many times the promise is kept.
		if len(declared.Uses) > 0 {
			if _, callErr := r.RunRowCalls(ctx, rowInvocation, declared, rows[index:index+1]); callErr != nil {
				return callErr
			}
		}
		var rowErr error
		if calling, ok := implementation.(CallingImplementation); ok {
			rowErr = calling.RunWithCaller(ctx, rowInvocation, declaredCaller(r.Caller, declared))
		} else {
			rowErr = implementation.Run(ctx, rowInvocation)
		}
		if rowErr == nil {
			continue
		}
		rowErr = wrapFailure(rowErr, declared)
		// The row is named so a caller can point at the record that refused rather
		// than re-running the whole page to find it.
		if _, ok := OutcomeOf(rowErr); ok {
			return &RowRefusal{Row: index, Err: rowErr}
		}
		return &Error{
			Code:    "HOOK_ROW_FAILED",
			Message: fmt.Sprintf("row %d: %v", index, rowErr),
			Cause:   rowErr,
		}
	}
	return nil
}

// RowRefusal is a business refusal at one row. It keeps the outcome name, so a
// caller branches on the business answer while still learning which row it came
// from.
type RowRefusal struct {
	Row int
	Err error
}

func (r *RowRefusal) Error() string { return fmt.Sprintf("row %d: %v", r.Row, r.Err) }
func (r *RowRefusal) Unwrap() error { return r.Err }

// narrowTo returns the values the declaration names, and only those. A declared
// read with no value is left out so ReadsFor still refuses it: narrowing must not
// turn "not observed" into "observed as empty".
func narrowTo(row map[core.CanonicalID]string, declared []core.CanonicalID) map[core.CanonicalID]string {
	out := map[core.CanonicalID]string{}
	for _, attribute := range declared {
		if value, ok := row[attribute]; ok {
			out[attribute] = value
		}
	}
	return out
}

// declaredHook resolves the pattern and the hook declaration for an invocation, so
// the row path shares the lookup instead of repeating it.
func (r Runner) declaredHook(invocation Invocation) (*application.Pattern, *application.Hook, error) {
	var pattern *application.Pattern
	for i := range r.World.Applications {
		if r.World.Applications[i].ID != invocation.Application {
			continue
		}
		for j := range r.World.Applications[i].Patterns {
			if r.World.Applications[i].Patterns[j].ID == invocation.Pattern {
				pattern = &r.World.Applications[i].Patterns[j]
				break
			}
		}
		break
	}
	if pattern == nil {
		return nil, nil, &Error{Code: "HOOK_PATTERN_NOT_FOUND", Message: fmt.Sprintf("pattern %s is not defined in application %s", invocation.Pattern, invocation.Application)}
	}
	// A row-scoped hook is declared on a pattern that looks at a row set, which
	// is a browse (or a header-detail) pattern rather than a record pattern. The
	// record form is accepted too: Run needs it, and declaredHook serves both.
	if pattern.Interaction == nil {
		return nil, nil, &Error{Code: "HOOK_PATTERN_NOT_INTERACTIVE", Message: fmt.Sprintf("pattern %s has no interaction to declare hooks", invocation.Pattern)}
	}
	for i := range pattern.Interaction.Hooks {
		if pattern.Interaction.Hooks[i].Phase == invocation.Phase {
			return pattern, &pattern.Interaction.Hooks[i], nil
		}
	}
	return nil, nil, &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s does not declare a %s hook", invocation.Pattern, invocation.Phase)}
}

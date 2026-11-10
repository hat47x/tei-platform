package hook_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

func rowScopeWorld(scope string) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{{
					Phase: "before-update",
					Scope: scope,
					Reads: []core.CanonicalID{"sales.order.amount"},
				}}},
			}},
		}},
	}
}

// rowImplementation records the values it was given per call, and can refuse at a
// chosen call index.
type rowImplementation struct {
	seen    []map[core.CanonicalID]string
	refuse  int
	outcome string
}

func (i *rowImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.seen = append(i.seen, invocation.Reads)
	if i.refuse > 0 && len(i.seen) == i.refuse {
		return &hook.NotSatisfied{Reason: "the amount exceeds the limit", Outcome: i.outcome}
	}
	return nil
}

func rowsOf(values ...string) []map[core.CanonicalID]string {
	out := make([]map[core.CanonicalID]string, 0, len(values))
	for _, value := range values {
		out = append(out, map[core.CanonicalID]string{"sales.order.amount": value})
	}
	return out
}

// TestRunRowsAppliesTheHookToEachRow checks that a row-scoped hook runs once per row,
// in order. X-TEA's script looped inside one call, so the declaration could not see
// the repetition.
func TestRunRowsAppliesTheHookToEachRow(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{World: rowScopeWorld("row"), Implementation: implementation}
	err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, rowsOf("10", "20", "30"))
	if err != nil {
		t.Fatal(err)
	}
	if len(implementation.seen) != 3 {
		t.Fatalf("calls=%d", len(implementation.seen))
	}
	// Each call saw its own row's value, so the values did not leak between rows.
	for index, want := range []string{"10", "20", "30"} {
		if got := implementation.seen[index]["sales.order.amount"]; got != want {
			t.Fatalf("row %d saw %q, want %q", index, got, want)
		}
	}
}

// TestRunRowsNarrowsToDeclaredReads checks that a row value the declaration does not
// name is not passed to the hook.
func TestRunRowsNarrowsToDeclaredReads(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{World: rowScopeWorld("row"), Implementation: implementation}
	rows := []map[core.CanonicalID]string{{
		"sales.order.amount": "10",
		"sales.order.secret": "x",
	}}
	if err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, rows); err != nil {
		t.Fatal(err)
	}
	if _, leaked := implementation.seen[0]["sales.order.secret"]; leaked {
		t.Fatal("an undeclared attribute reached the hook")
	}
}

// TestRunRowsStopsAtTheRefusedRow checks that a refusal names the row it stopped at
// and the later rows are not visited.
func TestRunRowsStopsAtTheRefusedRow(t *testing.T) {
	implementation := &rowImplementation{refuse: 2, outcome: hook.OutcomeValueRejected}
	runner := hook.Runner{World: rowScopeWorld("row"), Implementation: implementation}
	err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, rowsOf("10", "20", "30"))
	if err == nil {
		t.Fatal("the refusal was reported as success")
	}
	if len(implementation.seen) != 2 {
		t.Fatalf("the rows after the refusal were visited: %d calls", len(implementation.seen))
	}
	var refusal *hook.RowRefusal
	if !errors.As(err, &refusal) || refusal.Row != 1 {
		t.Fatalf("err=%v", err)
	}
	// The business outcome survives, so a caller branches on it.
	if outcome, ok := hook.OutcomeOf(refusal); !ok || outcome != hook.OutcomeValueRejected {
		t.Fatalf("outcome=%q ok=%v", outcome, ok)
	}
}

// TestRunRowsRefusesANonRowScope checks that a record-scoped hook is not applied per
// row: doing so would run it more often than it was declared for.
func TestRunRowsRefusesANonRowScope(t *testing.T) {
	for _, scope := range []string{"", "record", "header"} {
		implementation := &rowImplementation{}
		runner := hook.Runner{World: rowScopeWorld(scope), Implementation: implementation}
		err := runner.RunRows(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		}, rowsOf("10"))
		if err == nil {
			t.Fatalf("scope %q was applied per row", scope)
		}
		if len(implementation.seen) != 0 {
			t.Fatalf("scope %q ran anyway", scope)
		}
	}
}

// TestRunRowsAcceptsAnEmptyResult checks that an empty page is not an error: there
// is simply nothing to apply the hook to.
func TestRunRowsAcceptsAnEmptyResult(t *testing.T) {
	implementation := &rowImplementation{}
	runner := hook.Runner{World: rowScopeWorld("row"), Implementation: implementation}
	if err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	}, nil); err != nil {
		t.Fatalf("an empty result was refused: %v", err)
	}
	if len(implementation.seen) != 0 {
		t.Fatalf("the hook ran for an empty result: %d", len(implementation.seen))
	}
}

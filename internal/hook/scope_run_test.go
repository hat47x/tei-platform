package hook_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/hook"
	"tei/internal/ir"
)

func scopeWorld(scope string) ir.World {
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
				}}},
			}},
		}},
	}
}

type scopeImplementation struct {
	ran []string
}

func (i *scopeImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.ran = append(i.ran, invocation.Phase)
	return nil
}

// TestRunRefusesAnUnappliedScope checks that a scope the runner cannot apply is
// refused instead of the hook running as if the scope did not matter. X-TEA's D-9
// was exactly this: a definition present in the source and inert at run time.
func TestRunRefusesAnUnappliedScope(t *testing.T) {
	// A row-scoped hook needs the row set, which Run does not carry: it is pointed
	// at RunRows instead of being run as a single record.
	for _, scope := range []string{"row"} {
		implementation := &scopeImplementation{}
		runner := hook.Runner{World: scopeWorld(scope), Implementation: implementation}
		err := runner.Run(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		})
		if err == nil || !strings.Contains(err.Error(), "HOOK_SCOPE_NEEDS_ROWS") {
			t.Fatalf("scope %s: err=%v", scope, err)
		}
		if len(implementation.ran) != 0 {
			t.Fatalf("scope %s: the hook ran anyway: %v", scope, implementation.ran)
		}
	}
	// The scopes nothing applies are refused as before.
	for _, scope := range []string{"item", "header"} {
		implementation := &scopeImplementation{}
		runner := hook.Runner{World: scopeWorld(scope), Implementation: implementation}
		err := runner.Run(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		})
		if err == nil {
			t.Fatalf("scope %s ran without being applied", scope)
		}
		if !strings.Contains(err.Error(), "HOOK_SCOPE_UNAPPLIED") {
			t.Fatalf("scope %s: unexpected error %v", scope, err)
		}
		if len(implementation.ran) != 0 {
			t.Fatalf("scope %s: the hook ran anyway: %v", scope, implementation.ran)
		}
	}
}

// TestRunAppliesTheRecordScope checks that the scope the runner does apply still
// runs, so the refusal is narrow.
func TestRunAppliesTheRecordScope(t *testing.T) {
	for _, scope := range []string{"", "record"} {
		implementation := &scopeImplementation{}
		runner := hook.Runner{World: scopeWorld(scope), Implementation: implementation}
		if err := runner.Run(context.Background(), hook.Invocation{
			Application: "order_entry",
			Pattern:     "order_entry.order_record",
			Phase:       "before-update",
		}); err != nil {
			t.Fatalf("scope %q was refused: %v", scope, err)
		}
		if len(implementation.ran) != 1 {
			t.Fatalf("scope %q did not run: %v", scope, implementation.ran)
		}
	}
}

// TestRunSequenceRefusesAnUnappliedScope checks the same refusal on the sequence
// path, so both entry points apply one rule.
func TestRunSequenceRefusesAnUnappliedScope(t *testing.T) {
	implementation := &scopeImplementation{}
	runner := hook.Runner{World: scopeWorld("row"), Implementation: implementation}
	_, err := runner.RunSequence(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
	})
	// A row-scoped hook is pointed at RunRows rather than run once for the record.
	if err == nil || !strings.Contains(err.Error(), "HOOK_SCOPE_NEEDS_ROWS") {
		t.Fatalf("err=%v", err)
	}
	if len(implementation.ran) != 0 {
		t.Fatalf("the sequence ran the hook anyway: %v", implementation.ran)
	}
}

// TestRunReachableSkipsAnUnappliedScopeWithAReason checks that the reachable path
// reports the phase as skipped with a reason instead of running it, so a phase that
// did not run never reads as a phase that passed.
func TestRunReachableSkipsAnUnappliedScopeWithAReason(t *testing.T) {
	implementation := &scopeImplementation{}
	runner := hook.Runner{World: scopeWorld("row"), Implementation: implementation}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, hook.ReferenceReach{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, phase := range report.Skipped {
		if phase == "before-update" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the phase was not reported as skipped: %+v", report)
	}
	if !strings.Contains(report.Reasons["before-update"], "scope row") {
		t.Fatalf("the reason does not name the scope: %q", report.Reasons["before-update"])
	}
	if len(implementation.ran) != 0 {
		t.Fatalf("the reachable run ran the hook anyway: %v", implementation.ran)
	}
}

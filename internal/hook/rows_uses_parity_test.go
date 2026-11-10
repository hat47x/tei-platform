package hook_test

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// rowUsesWorld declares a row-scoped hook that also names a capability to call.
// Whether that is allowed must not depend on which entry point runs it: the
// declaration says the same thing either way.
func rowUsesWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{Hooks: []application.Hook{{
					Phase: "before-update",
					Scope: "row",
					Reads: []core.CanonicalID{"sales.order.amount"},
					Uses:  []core.CanonicalID{"sales.check"},
				}}},
			}},
		}},
	}
}

// rowUsesImplementation records the phases it was asked to run and returns nil,
// so a test can see whether the declaration reached an implementation at all.
type rowUsesImplementation struct {
	phases []string
}

func (i *rowUsesImplementation) Run(_ context.Context, invocation hook.Invocation) error {
	i.phases = append(i.phases, invocation.Phase)
	return nil
}

// TestRowScopeWithUsesIsJudgedTheSameByRunAndRunReachable pins the inconsistency
// this test exists for.
//
// Run routes a row-scoped hook to RunRows, which hands the declared call to the
// implementation through declaredCaller. RunReachable routes the same hook to
// applyReachableRows, which refuses any row-scoped hook that declares `uses`.
// So the same declaration is callable through one entry point and refused
// through the other.
func TestRowScopeWithUsesIsJudgedTheSameByRunAndRunReachable(t *testing.T) {
	called := &rowUsesImplementation{}

	// Path 1: Run with a row set routes to RunRows, which calls the capability.
	runRunner := hook.Runner{
		World:          rowUsesWorld(),
		Implementation: called,
		Rows:           []map[core.CanonicalID]string{{"sales.order.amount": "10"}},
	}
	runErr := runRunner.Run(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
	})

	// Path 2: RunReachable with the same row set goes through
	// applyReachableRows, which refuses the same declaration.
	reachableCalled := &rowUsesImplementation{}
	reachableRunner := hook.Runner{
		World:          rowUsesWorld(),
		Implementation: reachableCalled,
		Rows:           []map[core.CanonicalID]string{{"sales.order.amount": "10"}},
	}
	report, reachableErr := reachableRunner.RunReachable(context.Background(),
		"order_entry", "order_entry.order_record",
		map[core.CanonicalID]string{"sales.order.amount": "10"}, hook.ReferenceReach{})

	if (runErr == nil) != (reachableErr == nil) {
		t.Fatalf("the same declaration got two answers: Run err=%v, RunReachable err=%v (report=%+v)",
			runErr, reachableErr, report)
	}
}

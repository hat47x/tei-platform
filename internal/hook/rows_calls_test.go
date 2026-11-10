package hook_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// rowCallWorld declares a browse pattern whose row-scoped hook also names one
// capability to call, plus the command pattern that capability resolves to. A
// row-scoped `uses` has somewhere to go in this world, so a test can check that
// the call happens once per row rather than being refused.
func rowCallWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", TypeRef: "std.identifier"},
					{ID: "sales.order.amount", TypeRef: "std.decimal"},
				},
			}},
			Capabilities: []semantic.Capability{{
				ID:       "sales.check",
				Symbol:   "check",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
				Outcomes: []semantic.Outcome{{Name: "checked", Payload: "sales.order"}},
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{
				{
					ID:   "order_entry.order_list",
					Kind: application.PatternBrowse,
					Data: "sales.order",
					Interaction: &application.Interaction{
						Hooks: []application.Hook{{
							Phase: "before-read",
							Scope: "row",
							Reads: []core.CanonicalID{"sales.order.order_id", "sales.order.amount"},
							Uses:  []core.CanonicalID{"sales.check"},
						}},
						Commands: []application.InteractionCommand{{
							Capability: "sales.check",
							Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
						}},
					},
				},
				{
					ID:         "order_entry.check",
					Kind:       application.PatternCommand,
					Data:       "sales.order",
					Capability: "sales.check",
				},
			},
		}},
	}
}

// rowRecordingCaller counts the calls and keeps the inputs each one received.
type rowRecordingCaller struct {
	inputs []map[string]string
}

func (c *rowRecordingCaller) Call(_ context.Context, _ core.CanonicalID, inputs map[string]string) (any, error) {
	copied := make(map[string]string, len(inputs))
	for key, value := range inputs {
		copied[key] = value
	}
	c.inputs = append(c.inputs, copied)
	return nil, nil
}

// rowRule records the reads each row handed it.
type rowRule struct {
	seen []map[core.CanonicalID]string
}

func (r *rowRule) Run(_ context.Context, invocation hook.Invocation) error {
	r.seen = append(r.seen, invocation.Reads)
	return nil
}

// TestRunRowsCallsTheDeclaredCapabilityOncePerRow checks the promise a row-scoped
// `uses` makes.
//
// A row-scoped hook runs once per row, so its `uses` is one call per row. Before
// this, a row-scoped hook with `uses` was refused outright: the record path kept
// the promise through RunDeclaredCalls but no per-row equivalent existed, so
// calling once for the page would have kept the promise for no particular row.
func TestRunRowsCallsTheDeclaredCapabilityOncePerRow(t *testing.T) {
	rule := &rowRule{}
	caller := &rowRecordingCaller{}
	runner := hook.Runner{World: rowCallWorld(), Implementation: rule, Caller: caller}
	if err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_list",
		Phase:       "before-read",
	}, []map[core.CanonicalID]string{
		{"sales.order.order_id": "ORD-1", "sales.order.amount": "10"},
		{"sales.order.order_id": "ORD-2", "sales.order.amount": "20"},
	}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(caller.inputs) != 2 {
		t.Fatalf("calls=%d, want one per row: %v", len(caller.inputs), caller.inputs)
	}
	// Each call receives that row's own value, bound through the interaction
	// command's `input order sales.order.order_id`.
	if caller.inputs[0]["order"] != "ORD-1" || caller.inputs[1]["order"] != "ORD-2" {
		t.Fatalf("calls=%v", caller.inputs)
	}
	if len(rule.seen) != 2 {
		t.Fatalf("the rule ran %d times", len(rule.seen))
	}
}

// TestRunRowsRefusesARowScopeThatCannotResolveItsCall checks the fail-closed
// side: a declared capability with no command pattern to resolve is refused
// rather than silently skipped.
func TestRunRowsRefusesARowScopeThatCannotResolveItsCall(t *testing.T) {
	world := rowCallWorld()
	// Drop the command pattern that resolves sales.check.
	world.Applications[0].Patterns = world.Applications[0].Patterns[:1]
	rule := &rowRule{}
	caller := &rowRecordingCaller{}
	runner := hook.Runner{World: world, Implementation: rule, Caller: caller}
	err := runner.RunRows(context.Background(), hook.Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_list",
		Phase:       "before-read",
	}, []map[core.CanonicalID]string{{"sales.order.order_id": "ORD-1"}})
	if err == nil {
		t.Fatal("an unresolvable declared call was accepted")
	}
	if !strings.Contains(err.Error(), "HOOK_PATTERN_NOT_DECLARED") {
		t.Fatalf("err=%v", err)
	}
	if len(caller.inputs) != 0 {
		t.Fatalf("the call was made anyway: %v", caller.inputs)
	}
}

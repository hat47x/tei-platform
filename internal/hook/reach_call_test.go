package hook

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// reachCallWorld declares a before-update hook that reads the status and
// declares the capability it calls, plus the command pattern that states which
// attribute feeds the capability's parameter.
func reachCallWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{
			{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase: "before-update",
						Reads: []core.CanonicalID{"sales.order.status"},
						Uses:  []core.CanonicalID{"sales.submit_order"},
					}},
				},
			},
			{
				ID:         "order_entry.submit_order",
				Kind:       application.PatternCommand,
				Capability: "sales.submit_order",
				Data:       "sales.order",
				Interaction: &application.Interaction{Commands: []application.InteractionCommand{{
					Capability: "sales.submit_order",
					Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.status"}},
				}}},
			},
		},
	}}}
}

// reachCountingCaller records the capabilities that reached it.
type reachCountingCaller struct {
	calls []core.CanonicalID
}

func (c *reachCountingCaller) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	c.calls = append(c.calls, capability)
	return nil, nil
}

// TestRunReachableCallsTheDeclaredCapability checks that the reachable path
// actually calls a hook's `uses` when a caller and inputs are supplied, and that
// the phase is then reported as executed with the call recorded.
func TestRunReachableCallsTheDeclaredCapability(t *testing.T) {
	boundary := &reachCountingCaller{}
	runner := Runner{
		World:          reachCallWorld(),
		Implementation: ReferenceRule{},
		Caller:         boundary,
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record",
		map[core.CanonicalID]string{"sales.order.status": "open"}, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(boundary.calls) != 1 || boundary.calls[0] != "sales.submit_order" {
		t.Fatalf("the declared capability did not reach the boundary: %v", boundary.calls)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Uncalled) != 0 {
		t.Fatalf("uncalled=%v", report.Uncalled)
	}
	calls := report.Calls["before-update"]
	if len(calls) != 1 || !calls[0].Called || calls[0].Capability != "sales.submit_order" {
		t.Fatalf("calls=%+v", calls)
	}
}

// TestRunReachableStopsAtACallRefusal checks that a refused call stops the
// sequence, so the rule does not run after a call the boundary rejected.
func TestRunReachableStopsAtACallRefusal(t *testing.T) {
	runner := Runner{
		World:          reachCallWorld(),
		Implementation: ReferenceRule{},
		Caller:         refusingCaller{},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record",
		map[core.CanonicalID]string{"sales.order.status": "open"}, ReferenceReach{})
	if err == nil {
		t.Fatal("a refused call did not stop the sequence")
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// refusingCaller refuses every call.
type refusingCaller struct{}

func (refusingCaller) Call(context.Context, core.CanonicalID, map[string]string) (any, error) {
	return nil, &Error{Code: "HOOK_CALL_REFUSED", Message: "the boundary refused"}
}

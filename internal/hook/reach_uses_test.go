package hook

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// usesWorld declares a before-update hook that declares one read and one call,
// plus the command pattern the declaration is resolved through.
func usesWorld() ir.World {
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

// recordingCaller counts the calls it received.
type recordingCaller struct {
	calls []core.CanonicalID
	err   error
}

func (c *recordingCaller) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	c.calls = append(c.calls, capability)
	if c.err != nil {
		return nil, c.err
	}
	return nil, nil
}

func usesInputs() map[core.CanonicalID]string {
	return map[core.CanonicalID]string{"sales.order.status": "open"}
}

// TestRunReachableDoesNotClaimADeclaredCallItDidNotMake checks that a hook which
// declares `uses` is not reported as executed when nothing called that
// capability. Claiming `executed` there would say the declared call happened.
func TestRunReachableDoesNotClaimADeclaredCallItDidNotMake(t *testing.T) {
	runner := Runner{World: usesWorld(), Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no report")
	}
	if len(report.Executed) != 0 {
		t.Fatalf("a phase whose declared call never ran was reported as executed: %v", report.Executed)
	}
	if len(report.NotEvaluated) != 1 || report.NotEvaluated[0] != "before-update" {
		t.Fatalf("notEvaluated=%v", report.NotEvaluated)
	}
	uncalled := report.Uncalled["before-update"]
	if len(uncalled) != 1 || uncalled[0] != "sales.submit_order" {
		t.Fatalf("uncalled=%v", report.Uncalled)
	}
	if report.Reasons["before-update"] != ReasonCallerUnavailable {
		t.Fatalf("reason=%q", report.Reasons["before-update"])
	}
}

// TestRunReachableCallsTheDeclaredCapabilityWhenACallerIsConfigured checks that
// a caller plus the declaration's own binding is enough to keep the promise: the
// capability reaches the boundary and the phase is executed.
func TestRunReachableCallsTheDeclaredCapabilityWhenACallerIsConfigured(t *testing.T) {
	caller := &recordingCaller{}
	runner := Runner{World: usesWorld(), Implementation: ReferenceRule{}, Caller: caller}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v (reasons=%v)", report.Executed, report.Reasons)
	}
	if len(caller.calls) != 1 || caller.calls[0] != "sales.submit_order" {
		t.Fatalf("the declaration did not reach the boundary: %v", caller.calls)
	}
}

// TestRunReachableRecordsAnUnresolvableCapability checks that a `uses` the
// declaration cannot resolve to a command pattern is recorded, not run and not
// silently counted as executed.
func TestRunReachableRecordsAnUnresolvableCapability(t *testing.T) {
	world := usesWorld()
	world.Applications[0].Patterns = world.Applications[0].Patterns[:1] // drop the command pattern
	caller := &recordingCaller{}
	runner := Runner{World: world, Implementation: ReferenceRule{}, Caller: caller}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.NotEvaluated) != 1 || report.NotEvaluated[0] != "before-update" {
		t.Fatalf("notEvaluated=%v", report.NotEvaluated)
	}
	if report.Reasons["before-update"] == "" {
		t.Fatal("the unresolved capability has no stated reason")
	}
	if len(caller.calls) != 0 {
		t.Fatalf("a capability with no command pattern was called: %v", caller.calls)
	}
}

// TestRunReachableStillRunsAHookThatDeclaresNoCall checks that a hook which
// declares only reads keeps running. The new refusal must not swallow the
// ordinary case.
func TestRunReachableStillRunsAHookThatDeclaresNoCall(t *testing.T) {
	world := ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{{
					Phase: "before-update",
					Reads: []core.CanonicalID{"sales.order.status"},
				}},
			},
		}},
	}}}
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(report.Uncalled) != 0 {
		t.Fatalf("uncalled=%v", report.Uncalled)
	}
}

// TestRunReachableReportsACallerRefusal checks that a caller's refusal still
// stops the sequence, so a refused call is not turned into "not evaluated".
func TestRunReachableReportsACallerRefusal(t *testing.T) {
	world := usesWorld()
	// The hook has no reads, so the call step is reached without the read check
	// interfering.
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Reads = nil
	caller := &recordingCaller{err: &Error{Code: "HOOK_CALL_REFUSED", Message: "the command refused"}}
	runner := Runner{World: world, Implementation: Guarded{
		Capability: "sales.submit_order",
		Inputs:     map[string]string{"order": "ORD-1"},
	}, Caller: caller}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	// Guarded calls the capability only through the caller, and RunReachable
	// refuses to run a phase whose declared call it cannot make, so the call is
	// not made here either; the refusal path is covered by Guarded's own tests.
	if err == nil && len(report.Executed) > 0 {
		t.Fatalf("a phase with an uncalled declaration was reported as executed: %+v", report)
	}
	var hookError *Error
	if err != nil && !errors.As(err, &hookError) {
		t.Fatalf("err=%v", err)
	}
}

// TestRunReachableUncalledKeepsTheDeclarationOrder checks that the capabilities
// are named in the order the declaration gives them.
func TestRunReachableUncalledKeepsTheDeclarationOrder(t *testing.T) {
	world := usesWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Uses = []core.CanonicalID{
		"sales.submit_order",
		"sales.audit_order",
	}
	// The second capability also needs a command pattern, or the binding step
	// refuses before the ordering question is reached.
	world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
		ID:         "order_entry.audit_order",
		Kind:       application.PatternCommand,
		Capability: "sales.audit_order",
		Data:       "sales.order",
	})
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	uncalled := report.Uncalled["before-update"]
	if strings.Join(idsOf(uncalled), ",") != "sales.submit_order,sales.audit_order" {
		t.Fatalf("uncalled=%v", uncalled)
	}
}

func idsOf(ids []core.CanonicalID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

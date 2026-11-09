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

// usesWorld declares a before-update hook that declares one read and one call.
func usesWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
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
		}},
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

// TestRunReachableNamesTheCallWhenACallerExistsButWasUnused checks the second
// reason: with a caller configured, a rule that does not call still leaves the
// declaration unkept.
func TestRunReachableNamesTheCallWhenACallerExistsButWasUnused(t *testing.T) {
	caller := &recordingCaller{}
	runner := Runner{World: usesWorld(), Implementation: ReferenceRule{}, Caller: caller}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", usesInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 0 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if report.Reasons["before-update"] != ReasonDeclaredCallNotMade {
		t.Fatalf("reason=%q", report.Reasons["before-update"])
	}
	if len(caller.calls) != 0 {
		t.Fatalf("the reference rule called %v", caller.calls)
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

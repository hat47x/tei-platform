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

// guardedWorld declares a record pattern whose hook reads the quantity and
// declares the capability the guarded rule calls, so the declaration permits the
// call the test expects.
func guardedWorld(failure string, capability core.CanonicalID) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase:   "before-update",
						Reads:   []core.CanonicalID{"sales.order.quantity"},
						Uses:    []core.CanonicalID{capability},
						Failure: failure,
					}},
				},
			}},
		}},
	}
}

// inputCaller records the capability and inputs that reached the boundary.
type inputCaller struct {
	called []core.CanonicalID
	inputs []map[string]string
}

func (c *inputCaller) Call(_ context.Context, capability core.CanonicalID, inputs map[string]string) (any, error) {
	c.called = append(c.called, capability)
	c.inputs = append(c.inputs, inputs)
	return nil, nil
}

// guardedRunner runs a guarded rule through the Runner with an explicit
// boundary, so the test can see whether the capability reached the boundary.
// declared is the capability the hook's declaration permits; called is the one
// the guarded rule asks for. They differ in the undeclared-call case.
func guardedRunner(failure string, declared, called core.CanonicalID, inner hook.Caller) hook.Runner {
	return hook.Runner{
		World: guardedWorld(failure, declared),
		Implementation: hook.Guarded{
			Rule:       hook.ReferenceRule{Positive: []core.CanonicalID{"sales.order.quantity"}},
			Capability: called,
			Inputs:     map[string]string{"order": "ORD-1"},
		},
		Caller: inner,
	}
}

// TestGuardedRunsTheCapabilityWhenTheRuleHolds checks that a satisfied rule
// reaches the boundary exactly once with the configured inputs.
func TestGuardedRunsTheCapabilityWhenTheRuleHolds(t *testing.T) {
	inner := &inputCaller{}
	runner := guardedRunner("reject", "sales.submit", "sales.submit", inner)
	if err := runner.Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{"sales.order.quantity": "2"})); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(inner.called) != 1 || inner.called[0] != "sales.submit" {
		t.Fatalf("calls=%v", inner.called)
	}
	if inner.inputs[0]["order"] != "ORD-1" {
		t.Fatalf("inputs=%v", inner.inputs)
	}
}

// TestGuardedDoesNotCallWhenTheRuleFails checks that a business refusal stops
// the call: the capability must not run after the hook said no.
func TestGuardedDoesNotCallWhenTheRuleFails(t *testing.T) {
	inner := &inputCaller{}
	runner := guardedRunner("reject", "sales.submit", "sales.submit", inner)
	err := runner.Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{"sales.order.quantity": "0"}))
	var notSatisfied *hook.NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
	if len(inner.called) != 0 {
		t.Fatalf("capability ran after a refusal: %v", inner.called)
	}
}

// TestGuardedCannotCallAnUndeclaredCapability checks that the declaration is the
// limit even for a guarded rule: the Runner passes a Declared caller, so an
// undeclared capability never reaches the boundary.
func TestGuardedCannotCallAnUndeclaredCapability(t *testing.T) {
	inner := &inputCaller{}
	runner := guardedRunner("reject", "sales.submit", "sales.delete", inner)
	err := runner.Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{"sales.order.quantity": "2"}))
	if err == nil {
		t.Fatal("an undeclared capability was called")
	}
	if len(inner.called) != 0 {
		t.Fatalf("undeclared capability reached the boundary: %v", inner.called)
	}
}

// TestGuardedFailsWithoutABoundaryWhenTheRuleHolds checks that a satisfied rule
// with no boundary is a failure, not a silent success.
func TestGuardedFailsWithoutABoundaryWhenTheRuleHolds(t *testing.T) {
	runner := guardedRunner("reject", "sales.submit", "sales.submit", nil)
	err := runner.Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{"sales.order.quantity": "2"}))
	if err == nil {
		t.Fatal("a capability was reported as succeeded without a boundary")
	}
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_REJECTED" {
		t.Fatalf("err=%v", err)
	}
}

// TestGuardedNeedsACapability checks that a misconfigured guarded rule is
// reported instead of silently passing.
func TestGuardedNeedsACapability(t *testing.T) {
	inner := &inputCaller{}
	runner := guardedRunner("reject", "", "", inner)
	err := runner.Run(context.Background(), ruleInvocation(map[core.CanonicalID]string{"sales.order.quantity": "2"}))
	if err == nil {
		t.Fatal("a guarded hook without a capability passed")
	}
	if len(inner.called) != 0 {
		t.Fatalf("calls=%v", inner.called)
	}
}

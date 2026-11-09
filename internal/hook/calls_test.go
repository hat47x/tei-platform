package hook_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// callWorld declares a record pattern whose hook reads the quantity and declares
// the capability it calls, and a command pattern that reaches that capability.
// Without the command pattern, internal/validation refuses the declaration; the
// call path resolves it through CommandPatternFor.
func callWorld(uses ...core.CanonicalID) ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID:         "sales.order",
				Attributes: []semantic.Attribute{{ID: "sales.order.order_id", TypeRef: "std.identifier"}},
			}},
			Capabilities: []semantic.Capability{{
				ID:       "sales.submit_order",
				Symbol:   "submit_order",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
				Outcomes: []semantic.Outcome{{Name: "submitted", Payload: "sales.order"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{
				{
					ID:   "order_entry.order_record",
					Kind: application.PatternRecord,
					Data: "sales.order",
					Interaction: &application.Interaction{
						// The interaction command states the correspondence the
						// hook's reads feed: capability parameter "order" is fed
						// by attribute sales.order.order_id.
						Commands: []application.InteractionCommand{{
							Capability: "sales.submit_order",
							Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
						}},
						Hooks: []application.Hook{{
							Phase: "before-update",
							Reads: []core.CanonicalID{"sales.order.order_id"},
							Uses:  uses,
						}},
					},
				},
				{
					// CommandPatternFor resolves a command pattern by the
					// Pattern's Capability, which is how a command pattern
					// declares the capability it reaches. A command pattern
					// carries no interaction, so the input binding lives on the
					// record pattern above.
					ID:         "order_entry.submit_order",
					Kind:       application.PatternCommand,
					Capability: "sales.submit_order",
					Data:       "sales.order",
				},
			},
		}},
	}
}

// callBoundary is a caller that records what reached it.
type callBoundary struct {
	calls []core.CanonicalID
	err   error
}

func (b *callBoundary) Call(_ context.Context, capability core.CanonicalID, _ map[string]string) (any, error) {
	b.calls = append(b.calls, capability)
	if b.err != nil {
		return nil, b.err
	}
	return nil, nil
}

// TestRunDeclaredCallsCallsTheDeclaredCapability checks that a hook's `uses`
// reaches the boundary. Before this path existed, nothing called it.
func TestRunDeclaredCallsCallsTheDeclaredCapability(t *testing.T) {
	boundary := &callBoundary{}
	runner := hook.Runner{World: callWorld("sales.submit_order"), Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update",
		map[core.CanonicalID]map[string]string{"sales.submit_order": {"order": "ORD-1"}}, boundary)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(boundary.calls) != 1 || boundary.calls[0] != "sales.submit_order" {
		t.Fatalf("the declaration did not reach the boundary: %v", boundary.calls)
	}
	if len(outcomes) != 1 || !outcomes[0].Called || outcomes[0].Capability != "sales.submit_order" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	if outcomes[0].Error != "" {
		t.Fatalf("outcome records an error: %+v", outcomes[0])
	}
}

// TestRunDeclaredCallsRefusesADeclaredButUnboundCapabilityNotBound checks that a
// declaration the caller gave no inputs for is reported as not called, with a
// reason, instead of being called with made-up inputs.
func TestRunDeclaredCallsRefusesADeclaredButUnboundCapabilityNotBound(t *testing.T) {
	boundary := &callBoundary{}
	runner := hook.Runner{World: callWorld("sales.submit_order"), Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update", nil, boundary)
	if err != nil {
		t.Fatalf("a missing binding is not a call failure: %v", err)
	}
	if len(boundary.calls) != 0 {
		t.Fatalf("a capability with no inputs was called: %v", boundary.calls)
	}
	if len(outcomes) != 1 || outcomes[0].Called {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	if outcomes[0].Reason != hook.ReasonNoInputs {
		t.Fatalf("reason=%q", outcomes[0].Reason)
	}
}

// TestRunDeclaredCallsRefusesWithoutACaller checks that the restricted caller
// refuses when there is no boundary behind it, rather than reporting a call.
func TestRunDeclaredCallsRefusesWithoutACaller(t *testing.T) {
	runner := hook.Runner{World: callWorld("sales.submit_order"), Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update",
		map[core.CanonicalID]map[string]string{"sales.submit_order": {"order": "ORD-1"}}, nil)
	if err == nil {
		t.Fatal("a call with no caller was accepted")
	}
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_IMPLEMENTATION_UNAVAILABLE" {
		t.Fatalf("err=%v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Error == "" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
}

// TestRunDeclaredCallsReportsTheBoundaryRefusal checks that a refusal is
// returned and recorded, so a refused call is not turned into a silent success.
func TestRunDeclaredCallsReportsTheBoundaryRefusal(t *testing.T) {
	boundary := &callBoundary{err: &hook.Error{Code: "HOOK_CALL_REFUSED", Message: "the command refused"}}
	runner := hook.Runner{World: callWorld("sales.submit_order"), Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update",
		map[core.CanonicalID]map[string]string{"sales.submit_order": {"order": "ORD-1"}}, boundary)
	if err == nil {
		t.Fatal("a refusal was not returned")
	}
	if len(outcomes) != 1 || outcomes[0].Error == "" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	if !strings.Contains(outcomes[0].Error, "the command refused") {
		t.Fatalf("the record does not name the refusal: %+v", outcomes[0])
	}
}

// TestRunDeclaredCallsWithoutUsesCallsNothing checks that a hook which declares
// no call reaches no boundary.
func TestRunDeclaredCallsWithoutUsesCallsNothing(t *testing.T) {
	boundary := &callBoundary{}
	runner := hook.Runner{World: callWorld(), Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update", nil, boundary)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(outcomes) != 0 || len(boundary.calls) != 0 {
		t.Fatalf("outcomes=%+v calls=%v", outcomes, boundary.calls)
	}
}

// TestDeclaredCallsResolvesTheCommandPattern checks that the binding helper uses
// CommandPatternFor, so an undeclared capability is refused rather than guessed.
func TestDeclaredCallsResolvesTheCommandPattern(t *testing.T) {
	world := callWorld("sales.submit_order")
	bound, err := hook.DeclaredCalls(world, "order_entry", "order_entry.order_record", "before-update",
		map[core.CanonicalID]string{"sales.order.order_id": "ORD-1"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(bound["sales.submit_order"]) != 1 {
		t.Fatalf("bound=%+v", bound)
	}

	// A capability with no command pattern is refused.
	if _, err := hook.DeclaredCalls(callWorld("sales.other"), "order_entry", "order_entry.order_record", "before-update", nil); err == nil {
		t.Fatal("an undeclared capability was accepted")
	}
}

// recordPatternOf returns the record pattern a world declares, so the binding
// helper can be exercised on the pattern that carries the interaction command.
func recordPatternOf(world ir.World, applicationID, patternID core.CanonicalID) *application.Pattern {
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for j := range world.Applications[i].Patterns {
			if world.Applications[i].Patterns[j].ID == patternID {
				return &world.Applications[i].Patterns[j]
			}
		}
	}
	return nil
}

// TestBindInputsFromReadsUsesTheDeclaredBinding checks the correspondence: the
// interaction command says parameter "order" is fed by attribute
// sales.order.order_id, so a hook read of that attribute supplies that parameter.
func TestBindInputsFromReadsUsesTheDeclaredBinding(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern := recordPatternOf(world, "order_entry", "order_entry.order_record")
	bound := hook.BindInputsFromReads(pattern, "sales.submit_order",
		map[core.CanonicalID]string{"sales.order.order_id": "ORD-7"})
	if bound["order"] != "ORD-7" {
		t.Fatalf("bound=%+v", bound)
	}
}

// TestBindInputsFromReadsOmitsAnUnreadParameter checks the negative direction: a
// parameter whose attribute the hook did not read is left out, so the call is
// reported as not made rather than called with a guessed value.
func TestBindInputsFromReadsOmitsAnUnreadParameter(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern := recordPatternOf(world, "order_entry", "order_entry.order_record")
	bound := hook.BindInputsFromReads(pattern, "sales.submit_order",
		map[core.CanonicalID]string{"sales.order.other": "x"})
	if len(bound) != 0 {
		t.Fatalf("bound=%+v", bound)
	}
}

// TestBindInputsFromReadsFansOneAttributeToTwoParameters checks that one read
// can feed more than one parameter: the declaration binds both "original" and
// "copy" to sales.order.order_id, and one read supplies both.
//
// The binding is keyed by parameter name, not by attribute, so a second
// parameter naming the same attribute is a second binding rather than a
// duplicate to drop.
func TestBindInputsFromReadsFansOneAttributeToTwoParameters(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern := recordPatternOf(world, "order_entry", "order_entry.order_record")
	pattern.Interaction.Commands[0].Inputs = []application.CommandInputBinding{
		{Name: "original", Attribute: "sales.order.order_id"},
		{Name: "copy", Attribute: "sales.order.order_id"},
	}
	bound := hook.BindInputsFromReads(pattern, "sales.submit_order",
		map[core.CanonicalID]string{"sales.order.order_id": "ORD-9"})
	if bound["original"] != "ORD-9" {
		t.Fatalf("original=%q bound=%+v", bound["original"], bound)
	}
	if bound["copy"] != "ORD-9" {
		t.Fatalf("copy=%q bound=%+v", bound["copy"], bound)
	}
	if len(bound) != 2 {
		t.Fatalf("bound=%+v", bound)
	}
}

// TestBindInputsFromReadsKeepsTheFirstBindingForOneParameter checks the other
// direction: the same parameter bound twice keeps the first declaration, so a
// later binding does not silently replace an earlier one.
func TestBindInputsFromReadsKeepsTheFirstBindingForOneParameter(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern := recordPatternOf(world, "order_entry", "order_entry.order_record")
	pattern.Interaction.Commands[0].Inputs = []application.CommandInputBinding{
		{Name: "order", Attribute: "sales.order.order_id"},
		{Name: "order", Attribute: "sales.order.other"},
	}
	bound := hook.BindInputsFromReads(pattern, "sales.submit_order",
		map[core.CanonicalID]string{
			"sales.order.order_id": "ORD-1",
			"sales.order.other":    "SHOULD-NOT-WIN",
		})
	if bound["order"] != "ORD-1" {
		t.Fatalf("order=%q bound=%+v", bound["order"], bound)
	}
}

// TestBindInputsFromReadsIgnoresAnotherCapability checks that bindings are read
// only from the interaction command that names the capability being called.
func TestBindInputsFromReadsIgnoresAnotherCapability(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern := recordPatternOf(world, "order_entry", "order_entry.order_record")
	bound := hook.BindInputsFromReads(pattern, "sales.other",
		map[core.CanonicalID]string{"sales.order.order_id": "ORD-7"})
	if len(bound) != 0 {
		t.Fatalf("bound=%+v", bound)
	}
}

// TestCommandCallerRunsTheDeclaredCallInOnePath checks the whole path: resolve the
// command pattern, call through the Command boundary, and see the effect reported
// by the invoker.
func TestCommandCallerRunsTheDeclaredCallInOnePath(t *testing.T) {
	world := callWorld("sales.submit_order")
	pattern, err := hook.CommandPatternFor(world, "order_entry", "sales.submit_order")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	invoker := command.Invoker{
		World:      world,
		Executor:   command.ReferenceExecutor{World: world},
		UnitOfWork: &command.MemoryUnitOfWork{},
		Authorizer: command.ReferenceAuthorizer{},
	}
	caller := hook.CommandCaller{
		Invoker:     invoker,
		Application: "order_entry",
		Pattern:     func(core.CanonicalID) (core.CanonicalID, bool) { return pattern, true },
	}
	runner := hook.Runner{World: world, Implementation: hook.ReferenceRule{}}
	outcomes, err := runner.RunDeclaredCalls(context.Background(), "order_entry", "order_entry.order_record", "before-update",
		map[core.CanonicalID]map[string]string{"sales.submit_order": {"order": "ORD-1"}}, caller)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(outcomes) != 1 || !outcomes[0].Called || outcomes[0].Error != "" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
}

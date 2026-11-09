package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// producesWorld declares one hook that reads one attribute and proposes two.
func producesWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase:    "before-update",
						Reads:    []core.CanonicalID{"sales.order.quantity"},
						Produces: []core.CanonicalID{"sales.order.amount", "sales.order.tax"},
					}},
				},
			}},
		}},
	}
}

// producingRule proposes a fixed amount, so the test checks the boundary rather
// than a business rule.
type producingRule struct {
	proposed Produced
	err      error
}

func (p producingRule) Run(context.Context, Invocation) error { return p.err }

func (p producingRule) RunProducing(_ context.Context, _ Invocation) (Produced, error) {
	return p.proposed, p.err
}

func producesInvocation() Invocation {
	return Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
		Reads:       map[core.CanonicalID]string{"sales.order.quantity": "2"},
	}
}

// TestProducesForAcceptsDeclaredValues checks that a proposal naming only
// declared attributes passes.
func TestProducesForAcceptsDeclaredValues(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Produces: []core.CanonicalID{"sales.order.amount"}}
	produced, err := ProducesFor(hook, Produced{"sales.order.amount": "600"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if produced["sales.order.amount"] != "600" {
		t.Fatalf("produced=%v", produced)
	}
}

// TestProducesForRefusesAnUndeclaredValue checks that a proposal outside the
// declaration is refused: a hook must not write an item it did not declare.
func TestProducesForRefusesAnUndeclaredValue(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Produces: []core.CanonicalID{"sales.order.amount"}}
	_, err := ProducesFor(hook, Produced{"sales.order.customer": "C-1"})
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PRODUCE_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestProducesForAcceptsAnEmptyProposal checks that producing nothing is allowed:
// an empty proposal is not an undeclared write.
func TestProducesForAcceptsAnEmptyProposal(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Produces: []core.CanonicalID{"sales.order.amount"}}
	if _, err := ProducesFor(hook, nil); err != nil {
		t.Fatalf("err=%v", err)
	}
}

// TestProducesForCopiesTheValues checks that the returned proposal is a copy, so
// a caller cannot widen it after the check.
func TestProducesForCopiesTheValues(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Produces: []core.CanonicalID{"sales.order.amount"}}
	input := Produced{"sales.order.amount": "600"}
	produced, err := ProducesFor(hook, input)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	produced["sales.order.other"] = "x"
	if _, exists := input["sales.order.other"]; exists {
		t.Fatal("the resolved proposal shares storage with the input")
	}
}

// TestRunWithDeclaredValuesReturnsTheProposal checks that the Runner passes the
// declared reads and returns the checked proposal.
func TestRunWithDeclaredValuesReturnsTheProposal(t *testing.T) {
	runner := Runner{
		World:          producesWorld(),
		Implementation: producingRule{proposed: Produced{"sales.order.amount": "600"}},
	}
	produced, err := runner.RunWithDeclaredValues(context.Background(), producesInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if produced["sales.order.amount"] != "600" {
		t.Fatalf("produced=%v", produced)
	}
}

// TestRunWithDeclaredValuesRefusesAnUndeclaredProposal checks that the Runner
// does not return a proposal the declaration does not name.
func TestRunWithDeclaredValuesRefusesAnUndeclaredProposal(t *testing.T) {
	runner := Runner{
		World:          producesWorld(),
		Implementation: producingRule{proposed: Produced{"sales.order.customer": "C-1"}},
	}
	_, err := runner.RunWithDeclaredValues(context.Background(), producesInvocation())
	if err == nil {
		t.Fatal("an undeclared proposal was returned")
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PRODUCE_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestRunWithDeclaredValuesRefusesAnUndeclaredRead checks that the read check
// still runs before the producing implementation is called.
func TestRunWithDeclaredValuesRefusesAnUndeclaredRead(t *testing.T) {
	runner := Runner{
		World:          producesWorld(),
		Implementation: producingRule{},
	}
	invocation := producesInvocation()
	invocation.Reads["sales.order.secret"] = "hidden"
	if _, err := runner.RunWithDeclaredValues(context.Background(), invocation); err == nil {
		t.Fatal("an undeclared read reached the implementation")
	}
}

// TestRunWithDeclaredValuesRunsAPlainImplementation checks that a hook which
// does not propose values still runs and returns no proposal.
func TestRunWithDeclaredValuesRunsAPlainImplementation(t *testing.T) {
	runner := Runner{World: producesWorld(), Implementation: fakeImplementation{}}
	produced, err := runner.RunWithDeclaredValues(context.Background(), producesInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(produced) != 0 {
		t.Fatalf("produced=%v", produced)
	}
}

// TestRunWithDeclaredValuesKeepsTheWarnRule checks that a failure from the
// producing implementation keeps the declared warn behaviour.
func TestRunWithDeclaredValuesKeepsTheWarnRule(t *testing.T) {
	world := producesWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Failure = "warn"
	runner := Runner{World: world, Implementation: producingRule{err: errors.New("provider unavailable")}}
	if _, err := runner.RunWithDeclaredValues(context.Background(), producesInvocation()); err == nil {
		t.Fatal("a warn failure was swallowed")
	}
}

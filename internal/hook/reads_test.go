package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// readsWorld declares a record pattern whose single hook names the reads the
// boundary is expected to enforce.
func readsWorld(reads ...core.CanonicalID) ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{{
						Phase: "before-update",
						Reads: reads,
					}},
				},
			}},
		}},
	}
}

// capturedReads records the reads the implementation received.
type capturedReads struct {
	reads map[core.CanonicalID]string
}

func (c *capturedReads) Run(_ context.Context, invocation Invocation) error {
	c.reads = invocation.Reads
	return nil
}

func readsInvocation(reads map[core.CanonicalID]string) Invocation {
	return Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-update",
		Reads:       reads,
	}
}

// TestReadsForPassesTheDeclaredReads checks that the exact declared values reach
// the implementation.
func TestReadsForPassesTheDeclaredReads(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}}
	reads, err := ReadsFor(hook, map[core.CanonicalID]string{"sales.order.status": "open"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if reads["sales.order.status"] != "open" {
		t.Fatalf("reads=%v", reads)
	}
}

// TestReadsForRefusesAMissingRead checks that an unobserved declared read is
// refused instead of being passed as an empty value.
func TestReadsForRefusesAMissingRead(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}}
	_, err := ReadsFor(hook, nil)
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_READ_NOT_OBSERVED" {
		t.Fatalf("err=%v", err)
	}
}

// TestReadsForRefusesAnUndeclaredRead checks that a value the hook does not
// declare is refused: a hook must not see data outside its declaration.
func TestReadsForRefusesAnUndeclaredRead(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}}
	_, err := ReadsFor(hook, map[core.CanonicalID]string{
		"sales.order.status":   "open",
		"sales.order.customer": "C-1",
	})
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_READ_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestReadsForRefusesWithoutADeclaration checks that reads resolved without a
// declaration are refused.
func TestReadsForRefusesWithoutADeclaration(t *testing.T) {
	if _, err := ReadsFor(nil, map[core.CanonicalID]string{"sales.order.status": "open"}); err == nil {
		t.Fatal("reads without a declaration were accepted")
	}
}

// TestReadsForCopiesTheValues checks that the returned map is a copy, so a caller
// cannot add a value after the check.
func TestReadsForCopiesTheValues(t *testing.T) {
	hook := &application.Hook{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.status"}}
	input := map[core.CanonicalID]string{"sales.order.status": "open"}
	reads, err := ReadsFor(hook, input)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	reads["sales.order.other"] = "x"
	if _, exists := input["sales.order.other"]; exists {
		t.Fatal("the resolved map shares storage with the input")
	}
}

// TestRunWithDeclaredReadsHandsTheDeclaredValues checks that the Runner resolves
// reads through the declaration before the implementation sees them.
func TestRunWithDeclaredReadsHandsTheDeclaredValues(t *testing.T) {
	implementation := &capturedReads{}
	runner := Runner{World: readsWorld("sales.order.quantity"), Implementation: implementation}
	if err := runner.RunWithDeclaredReads(context.Background(), readsInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "2",
	})); err != nil {
		t.Fatalf("err=%v", err)
	}
	if implementation.reads["sales.order.quantity"] != "2" {
		t.Fatalf("reads=%v", implementation.reads)
	}
}

// TestRunWithDeclaredReadsRefusesBeforeTheImplementation checks that an
// undeclared read never reaches the implementation.
func TestRunWithDeclaredReadsRefusesBeforeTheImplementation(t *testing.T) {
	implementation := &capturedReads{}
	runner := Runner{World: readsWorld("sales.order.quantity"), Implementation: implementation}
	err := runner.RunWithDeclaredReads(context.Background(), readsInvocation(map[core.CanonicalID]string{
		"sales.order.quantity": "2",
		"sales.order.secret":   "hidden",
	}))
	if err == nil {
		t.Fatal("an undeclared read reached the implementation")
	}
	if implementation.reads != nil {
		t.Fatalf("reads=%v", implementation.reads)
	}
}

// TestRunWithDeclaredReadsKeepsTheWarnRule checks that the read check does not
// change how a failure is reported: warn still continues as a warning.
func TestRunWithDeclaredReadsKeepsTheWarnRule(t *testing.T) {
	world := readsWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Failure = "warn"
	runner := Runner{World: world, Implementation: &failingImplementation{err: errors.New("provider unavailable")}}
	err := runner.RunWithDeclaredReads(context.Background(), readsInvocation(nil))
	var warning *Warning
	if !errors.As(err, &warning) {
		t.Fatalf("err=%v", err)
	}
}

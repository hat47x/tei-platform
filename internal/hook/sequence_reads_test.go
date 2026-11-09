package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// perPhaseWorld declares two phases whose reads differ, so the test can check
// that the wider declaration does not leak into the narrower one.
func perPhaseWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Hooks: []application.Hook{
						{
							Phase: "before-read",
							Reads: []core.CanonicalID{"sales.order.status", "sales.order.quantity"},
						},
						{
							Phase: "before-update",
							Reads: []core.CanonicalID{"sales.order.status"},
						},
					},
				},
			}},
		}},
	}
}

// readRecorder records the reads each phase received.
type readRecorder struct {
	reads map[string]map[core.CanonicalID]string
}

func (r *readRecorder) Run(_ context.Context, invocation Invocation) error {
	if r.reads == nil {
		r.reads = map[string]map[core.CanonicalID]string{}
	}
	copied := map[core.CanonicalID]string{}
	for attribute, value := range invocation.Reads {
		copied[attribute] = value
	}
	r.reads[invocation.Phase] = copied
	return nil
}

func perPhaseInvocation() Invocation {
	return Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Reads: map[core.CanonicalID]string{
			"sales.order.status":   "open",
			"sales.order.quantity": "2",
		},
	}
}

// TestRunSequenceGivesEachPhaseItsOwnDeclaredReads checks that a phase with a
// narrower declaration does not inherit the wider view of another phase.
func TestRunSequenceGivesEachPhaseItsOwnDeclaredReads(t *testing.T) {
	recorder := &readRecorder{}
	runner := Runner{World: perPhaseWorld(), Implementation: recorder}
	result, err := runner.RunSequence(context.Background(), perPhaseInvocation())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Refused != "" {
		t.Fatalf("refused=%s", result.Refused)
	}
	if len(recorder.reads["before-read"]) != 2 {
		t.Fatalf("before-read reads=%v", recorder.reads["before-read"])
	}
	if len(recorder.reads["before-update"]) != 1 {
		t.Fatalf("before-update reads=%v", recorder.reads["before-update"])
	}
	if _, leaked := recorder.reads["before-update"]["sales.order.quantity"]; leaked {
		t.Fatalf("the narrower phase saw the wider read: %v", recorder.reads["before-update"])
	}
}

// TestRunSequenceDoesNotPassAnUndeclaredRead checks that a value no phase
// declares in that phase is not handed to it, even though another phase reads it.
func TestRunSequenceDoesNotPassAnUndeclaredRead(t *testing.T) {
	recorder := &readRecorder{}
	runner := Runner{World: perPhaseWorld(), Implementation: recorder}
	if _, err := runner.RunSequence(context.Background(), perPhaseInvocation()); err != nil {
		t.Fatalf("err=%v", err)
	}
	if _, leaked := recorder.reads["before-update"]["sales.order.quantity"]; leaked {
		t.Fatalf("an undeclared read reached the phase: %v", recorder.reads["before-update"])
	}
}

// TestRunSequenceStopsOnAMissingRead checks that a declared read the caller did
// not supply stops the sequence before the implementation runs.
func TestRunSequenceStopsOnAMissingRead(t *testing.T) {
	world := perPhaseWorld()
	world.Applications[0].Patterns[0].Interaction.Hooks[0].Reads = []core.CanonicalID{"sales.order.customer"}
	recorder := &readRecorder{}
	runner := Runner{World: world, Implementation: recorder}
	result, err := runner.RunSequence(context.Background(), perPhaseInvocation())
	if err == nil {
		t.Fatal("a missing read was passed through")
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_READ_NOT_OBSERVED" {
		t.Fatalf("err=%v", err)
	}
	if result.Refused != "before-read" {
		t.Fatalf("refused=%s", result.Refused)
	}
}

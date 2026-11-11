package command

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/semantic"
)

// correctionExemptWorld is a closed-rejected update period that names a correction
// capability whose own effect is an update. Without an exemption the two
// declarations would contradict each other.
func correctionExemptWorld() (Invoker, Request) {
	world := correctionWorld("sales.correct", []string{"update"})
	world.Applications[0].Patterns[0].Interaction.Commands = append(
		world.Applications[0].Patterns[0].Interaction.Commands,
		application.InteractionCommand{
			Capability: "sales.correct",
			Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
		},
	)
	world.Semantics[0].Capabilities = append(world.Semantics[0].Capabilities, semantic.Capability{
		ID:       "sales.correct",
		Symbol:   "correct",
		Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
		Outcomes: []semantic.Outcome{{Name: "corrected", Payload: "sales.order"}},
		Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
	})
	invoker := closedInvoker(world)
	// The executor reports the outcome the correction capability declares, so the
	// request reaches the period check rather than failing the effect contract.
	invoker.Executor = fakeExecutor{result: Result{
		Outcome: &semantic.Outcome{Name: "corrected", Payload: "sales.order"},
		Effect:  &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
	}}
	request := Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.correct",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
	return invoker, request
}

// TestInvokeLetsTheDeclaredCorrectionThroughAClosure checks that the capability a
// period declares for corrections is not also refused by closed-rejects.
//
// A period that says "a change must go through X" and "this kind of change is
// refused" would contradict itself if X's change were refused: the correction could
// never run. The exemption is narrow — only the capability the period names.
func TestInvokeLetsTheDeclaredCorrectionThroughAClosure(t *testing.T) {
	invoker, request := correctionExemptWorld()
	if _, err := invoker.Invoke(context.Background(), request); err != nil {
		t.Fatalf("the declared correction was refused by the closure: %v", err)
	}
}

// TestInvokeStillRefusesAnotherCapabilityInTheSameClosure checks that the exemption is
// narrow: another capability with the same effect is still refused by closed-rejects.
func TestInvokeStillRefusesAnotherCapabilityInTheSameClosure(t *testing.T) {
	invoker, _ := correctionExemptWorld()
	// The period's own submit capability has the same update effect and is not the
	// declared correction, so the correction rule refuses it before the closure is
	// consulted.
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_PERIOD_CORRECTION_REQUIRED" {
		t.Fatalf("code=%s", code)
	}
}

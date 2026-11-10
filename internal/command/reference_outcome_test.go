package command

import (
	"context"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// TestReferenceExecutorDoesNotChooseAmongSeveralOutcomes checks that the
// reference executor refuses to pick one of several outcomes. Before this, it
// returned Outcomes[0], and because the loader sorts outcomes by name, "first"
// meant whichever name sorts earliest: renaming an outcome changed the answer.
func TestReferenceExecutorDoesNotChooseAmongSeveralOutcomes(t *testing.T) {
	world := commandWorld()
	setCapabilityOutcomes(&world, []semantic.Outcome{
		{Name: "denied", Payload: "sales.order"},
		{Name: "granted", Payload: "sales.order"},
	})
	executor := ReferenceExecutor{World: world}
	result, err := executor.Execute(context.Background(), request())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Outcome != nil {
		t.Fatalf("an outcome was chosen from several: %+v", result.Outcome)
	}
	if result.OutcomeUndetermined == "" {
		t.Fatal("the absence of an outcome has no stated reason")
	}
	for _, want := range []string{"sales.submit", "2 outcomes"} {
		if !strings.Contains(result.OutcomeUndetermined, want) {
			t.Fatalf("the reason does not mention %q: %s", want, result.OutcomeUndetermined)
		}
	}
}

// TestReferenceExecutorReturnsTheSingleOutcome checks that the ordinary case
// still returns the declared outcome.
func TestReferenceExecutorReturnsTheSingleOutcome(t *testing.T) {
	world := commandWorld()
	setCapabilityOutcomes(&world, []semantic.Outcome{{Name: "submitted", Payload: "sales.order"}})
	executor := ReferenceExecutor{World: world}
	result, err := executor.Execute(context.Background(), request())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("outcome=%+v", result.Outcome)
	}
	if result.OutcomeUndetermined != "" {
		t.Fatalf("a determined outcome carries a reason: %s", result.OutcomeUndetermined)
	}
}

// TestReferenceExecutorWithNoOutcomeHasNoReason checks that a capability which
// declares no outcome is not reported as undetermined.
func TestReferenceExecutorWithNoOutcomeHasNoReason(t *testing.T) {
	world := commandWorld()
	setCapabilityOutcomes(&world, nil)
	executor := ReferenceExecutor{World: world}
	result, err := executor.Execute(context.Background(), request())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Outcome != nil || result.OutcomeUndetermined != "" {
		t.Fatalf("result=%+v", result)
	}
}

// TestInvokerRecordsWhyNoOutcomeWasDetermined checks that the reason reaches the
// operation record, so a reader of the log does not have to guess.
func TestInvokerRecordsWhyNoOutcomeWasDetermined(t *testing.T) {
	world := commandWorld()
	setCapabilityOutcomes(&world, []semantic.Outcome{
		{Name: "denied", Payload: "sales.order"},
		{Name: "granted", Payload: "sales.order"},
	})
	recorder := &MemoryRecorder{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      world,
		Executor:   ReferenceExecutor{World: world},
		UnitOfWork: &MemoryUnitOfWork{},
		Recorder:   recorder,
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if result.Outcome != nil {
		t.Fatalf("an outcome was chosen: %+v", result.Outcome)
	}
	if len(recorder.Records) == 0 {
		t.Fatal("no operation record")
	}
	if recorder.Records[len(recorder.Records)-1].OutcomeUndetermined == "" {
		t.Fatal("the record does not state why no outcome was determined")
	}
}

// setCapabilityOutcomes replaces the outcomes of the fixture capability.
func setCapabilityOutcomes(world *ir.World, outcomes []semantic.Outcome) {
	for i := range world.Semantics {
		for j := range world.Semantics[i].Capabilities {
			if world.Semantics[i].Capabilities[j].ID == "sales.submit" {
				world.Semantics[i].Capabilities[j].Output = ""
				world.Semantics[i].Capabilities[j].Outcomes = outcomes
				return
			}
		}
	}
}

package main

import (
	"context"
	"testing"

	"tei/internal/driver"
	"tei/internal/execution"
	"tei/internal/ir"
	"tei/internal/operations"
)

type closedSource struct{}

func (closedSource) PeriodStates(context.Context, string) (map[string]operations.PeriodState, error) {
	return map[string]operations.PeriodState{"sales.month": operations.PeriodClosed}, nil
}

func periodSurface() operations.OperationSurface {
	return operations.OperationSurface{Operations: []operations.Operation{{
		ID:            "app.create." + string(capabilityID),
		Kind:          "create",
		Target:        "sales.order",
		Binding:       string(capabilityID),
		Period:        "sales.month",
		PeriodRejects: []string{"create"},
	}}}
}

func TestGuardedSubmitRefusesAClosedPeriod(t *testing.T) {
	stages := []stageResult{{Stage: "physical-schema", Status: stageResolved}}
	resolver := driver.NewResolver(ir.World{})
	gate := submitIfReadyGuarded(context.Background(), resolver, stages, nil, submission{}, execution.PeriodGuard{
		Surface: periodSurface(),
		Source:  closedSource{},
	})
	if !gate.Refused {
		t.Fatalf("a closed period must refuse the submission: %+v", gate)
	}
	if len(gate.Reasons) != 1 || gate.Reasons[0].Stage != "period" {
		t.Fatalf("the refusal must name the period stage: %+v", gate.Reasons)
	}
	if gate.Invocation != nil {
		t.Fatal("a refused submission must not carry an invocation")
	}
}

func TestSubmitWithoutAGuardKeepsThePreflightOnly(t *testing.T) {
	// The resolver is not exercised: the refusal must happen before it is used.
	resolver := driver.NewResolver(ir.World{})
	gate := submitIfReadyGuarded(context.Background(), resolver, []stageResult{{Stage: "physical-schema", Status: stageUnresolved}}, nil, submission{}, execution.PeriodGuard{})
	if !gate.Refused {
		t.Fatalf("an unresolved preflight must still refuse: %+v", gate)
	}
}

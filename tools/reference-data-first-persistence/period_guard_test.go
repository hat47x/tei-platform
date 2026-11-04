package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/driver"
	"tei/internal/execution"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/periodstate"
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

// TestGuardedSubmitRechecksFileStateOnEachAttempt connects the same real
// periodstate.File source to the existing reference executor gate. A period
// decision from an earlier Action offer cannot be reused at dispatch; refusal
// is before Driver invocation. No real Provider/Transaction is exercised.
func TestGuardedSubmitRechecksFileStateOnEachAttempt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "period-state.json")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	guard := execution.PeriodGuard{
		Surface: periodSurface(),
		Source:  periodstate.File{Path: path},
	}
	stages := []stageResult{{Stage: "physical-schema", Status: stageResolved}}
	resolver := driver.NewResolver(ir.World{})

	// The File is read again for each guarded reference invocation.
	for _, state := range []string{"closing", "closed"} {
		write(`{"sales.month":"` + state + `"}`)
		gate := submitIfReadyGuarded(context.Background(), resolver, stages, nil, submission{}, guard)
		if !gate.Refused || gate.Invocation != nil || gate.Err != nil {
			t.Fatalf("state %s must refuse before Driver/Adapter: %+v", state, gate)
		}
		if len(gate.Reasons) != 1 || gate.Reasons[0].Stage != "period" {
			t.Fatalf("state %s must cite period refusal: %+v", state, gate.Reasons)
		}
	}

	// A malformed or unavailable Source is not the same as a successful
	// business refusal, and never proves that the period is currently open.
	write(`{"sales.month":"sideways"}`)
	invalid := submitIfReadyGuarded(context.Background(), resolver, stages, nil, submission{}, guard)
	if invalid.Err == nil || invalid.Invocation != nil {
		t.Fatalf("invalid runtime period Source must report technical error: %+v", invalid)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	unavailable := submitIfReadyGuarded(context.Background(), resolver, stages, nil, submission{}, guard)
	if unavailable.Err == nil || unavailable.Invocation != nil {
		t.Fatalf("unavailable runtime period Source must report technical error: %+v", unavailable)
	}
}

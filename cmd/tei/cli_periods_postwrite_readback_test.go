package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/periodstate"
)

// This is a characterization of the existing handler order, not a fix:
// Writer.Write persists the next state before CheckPeriods can return a
// rejection. A non-nil handler error therefore does not mean no effect.
// The real xtea-lowcode workspace is used rather than a second DSL fixture.
func TestPeriodsAdvanceRejectionAfterStateWriteMustReadBack(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "xtea-lowcode")
	const appID = "order_entry"
	const periodID = "sales.month"

	surface, err := operations.SurfaceWorkspace(root, appID)
	if err != nil {
		t.Fatalf("read actual workspace operation surface: %v", err)
	}
	for _, state := range []operations.PeriodState{
		operations.PeriodClosing, operations.PeriodClosed,
	} {
		refusals := operations.CheckPeriods(surface, map[string]operations.PeriodState{
			periodID: state,
		})
		if len(refusals) == 0 {
			t.Fatalf("reference workspace does not reject any operation at state %q", state)
		}
	}

	statePath := filepath.Join(t.TempDir(), "period-state.json")
	if err := os.WriteFile(statePath, []byte("{\"sales.month\":\"open\"}\n"), 0o600); err != nil {
		t.Fatalf("seed isolated state file: %v", err)
	}

	readState := func() operations.PeriodState {
		t.Helper()
		states, err := (periodstate.File{Path: statePath}).PeriodStates(context.Background(), appID)
		if err != nil {
			t.Fatalf("read back committed period file: %v", err)
		}
		return states[periodID]
	}

	for _, expected := range []operations.PeriodState{
		operations.PeriodClosing, operations.PeriodClosed,
	} {
		err := runPeriodsCommand(root, appID, "", statePath, periodID, "")
		if err == nil || !strings.Contains(err.Error(), "operation(s) are refused") {
			t.Fatalf("state transition to %s: expected post-write refusal, got %v", expected, err)
		}
		if got := readState(); got != expected {
			t.Fatalf("post-error state file: got %s; want committed %s", got, expected)
		}
	}

	// A blind third retry cannot be regarded as a clean retry of the first:
	// the persisted state is already closed, so NextState rejects it.
	if err := runPeriodsCommand(root, appID, "", statePath, periodID, ""); err == nil ||
		!strings.Contains(err.Error(), "cannot be closed further") {
		t.Fatalf("expected closed-period repeat refusal, got %v", err)
	}
	if got := readState(); got != operations.PeriodClosed {
		t.Fatalf("rejected retry must not rewrite final state: got %s", got)
	}
}

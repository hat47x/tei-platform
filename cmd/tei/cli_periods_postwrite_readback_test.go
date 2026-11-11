package main

import (
	"context"
	"fmt"
	"io/fs"
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
	// The existing xtea-lowcode example declares the period, but has no
	// persisted Order mapping from which the standard CRUD update operation
	// could be derived. Add only that mapping to a private copy for this test.
	root := copyPeriodReadbackFixture(t, filepath.Join("..", "..", "examples", "xtea-lowcode"))
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

func copyPeriodReadbackFixture(t *testing.T, source string) string {
	t.Helper()
	target := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected nonregular fixture input %s", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	})
	if err != nil {
		t.Fatalf("copy original example workspace: %v", err)
	}
	mapping := `persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders

    column sales.order.order_id -> order_id storage int64 required
    column sales.order.customer_id -> customer_id storage int64 required
    column sales.order.status -> status storage varchar required

    primary-key sales.order.order_id
}
`
	dir := filepath.Join(target, "databases", "sales-core", "persistence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("make isolated persistence source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "order.tei"), []byte(mapping), 0o644); err != nil {
		t.Fatalf("add minimal test-only mapping: %v", err)
	}
	return target
}

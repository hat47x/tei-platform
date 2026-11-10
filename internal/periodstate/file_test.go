package periodstate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
	"tei/internal/periodstate"
)

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "period-state.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestFileReadsHalfMonthScopes checks that a scope key such as
// "sales.month#1" is read like any other period key, so a period split by a
// closing day can be closed by half from the same file.
func TestFileReadsHalfMonthScopes(t *testing.T) {
	states, err := periodstate.File{Path: write(t, `{"sales.month#1": "closed", "sales.month#2": "open"}`)}.PeriodStates(context.Background(), "app")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if states["sales.month#1"] != operations.PeriodClosed || states["sales.month#2"] != operations.PeriodOpen {
		t.Fatalf("unexpected states: %+v", states)
	}
}

func TestFileReadsStates(t *testing.T) {
	states, err := periodstate.File{Path: write(t, `{"sales.month": "closed", "sales.week": "open"}`)}.PeriodStates(context.Background(), "app")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if states["sales.month"] != operations.PeriodClosed || states["sales.week"] != operations.PeriodOpen {
		t.Fatalf("unexpected states: %+v", states)
	}
}

func TestFileRejectsAnUnknownState(t *testing.T) {
	if _, err := (periodstate.File{Path: write(t, `{"sales.month": "sideways"}`)}).PeriodStates(context.Background(), "app"); err == nil {
		t.Fatal("an unknown state must be rejected")
	}
}

func TestFileRejectsAMissingFile(t *testing.T) {
	if _, err := (periodstate.File{Path: filepath.Join(t.TempDir(), "missing.json")}).PeriodStates(context.Background(), "app"); err == nil {
		t.Fatal("a missing file must be reported")
	}
}

var _ operations.PeriodStateSource = periodstate.File{}

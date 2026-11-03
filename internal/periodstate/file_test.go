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

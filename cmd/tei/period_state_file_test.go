package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
)

func writeStateFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "period-state.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPeriodStateFileReadsStates(t *testing.T) {
	path := writeStateFile(t, `{"sales.month": "closed", "sales.week": "open"}`)
	states, err := periodStateFile{path: path}.PeriodStates(context.Background(), "app")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if states["sales.month"] != operations.PeriodClosed || states["sales.week"] != operations.PeriodOpen {
		t.Fatalf("unexpected states: %+v", states)
	}
}

func TestPeriodStateFileRejectsAnUnknownState(t *testing.T) {
	path := writeStateFile(t, `{"sales.month": "sideways"}`)
	if _, err := (periodStateFile{path: path}).PeriodStates(context.Background(), "app"); err == nil {
		t.Fatal("an unknown state must be rejected")
	}
}

func TestPeriodStateFileRejectsAMissingFile(t *testing.T) {
	if _, err := (periodStateFile{path: filepath.Join(t.TempDir(), "missing.json")}).PeriodStates(context.Background(), "app"); err == nil {
		t.Fatal("a missing file must be reported")
	}
}

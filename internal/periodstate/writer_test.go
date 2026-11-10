package periodstate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/operations"
	"tei/internal/periodstate"
)

// TestWriterRecordsAndKeepsStates checks that a write replaces the periods it names
// and keeps the others, so a closing run does not drop a state it did not touch.
func TestWriterRecordsAndKeepsStates(t *testing.T) {
	path := write(t, `{"sales.month": "open", "sales.week": "closed"}`)
	writer := periodstate.Writer{Path: path}
	if err := writer.Write(context.Background(), map[string]operations.PeriodState{
		"sales.month": operations.PeriodClosing,
	}); err != nil {
		t.Fatal(err)
	}
	states, err := periodstate.File{Path: path}.PeriodStates(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if states["sales.month"] != operations.PeriodClosing {
		t.Fatalf("sales.month=%s", states["sales.month"])
	}
	if states["sales.week"] != operations.PeriodClosed {
		t.Fatalf("sales.week=%s", states["sales.week"])
	}
}

// TestWriterRecordsHalfMonthScopes checks that a half-month scope is written like
// any other key, so a period split by a closing day can be closed by half.
func TestWriterRecordsHalfMonthScopes(t *testing.T) {
	path := write(t, `{}`)
	if err := (periodstate.Writer{Path: path}).Write(context.Background(), map[string]operations.PeriodState{
		"sales.month#2": operations.PeriodClosing,
	}); err != nil {
		t.Fatal(err)
	}
	states, err := periodstate.File{Path: path}.PeriodStates(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if states["sales.month#2"] != operations.PeriodClosing {
		t.Fatalf("states=%+v", states)
	}
}

// TestWriterCreatesTheFile checks that a first write creates the file, so a caller
// does not have to seed an empty one.
func TestWriterCreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "period-state.json")
	if err := (periodstate.Writer{Path: path}).Write(context.Background(), map[string]operations.PeriodState{
		"sales.month": operations.PeriodClosed,
	}); err != nil {
		t.Fatal(err)
	}
	states, err := periodstate.File{Path: path}.PeriodStates(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if states["sales.month"] != operations.PeriodClosed {
		t.Fatalf("states=%+v", states)
	}
	// No temporary file is left behind, so a later reader does not see a stray.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != filepath.Base(path) {
			t.Fatalf("a stray file was left behind: %s", entry.Name())
		}
	}
}

// TestWriterRefusesAnUnknownState checks that a misspelt state is refused before
// anything is written, so the file keeps the states it had.
func TestWriterRefusesAnUnknownState(t *testing.T) {
	path := write(t, `{"sales.month": "open"}`)
	if err := (periodstate.Writer{Path: path}).Write(context.Background(), map[string]operations.PeriodState{
		"sales.month": "archived",
	}); err == nil {
		t.Fatal("an undeclared state was written")
	}
	states, err := periodstate.File{Path: path}.PeriodStates(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if states["sales.month"] != operations.PeriodOpen {
		t.Fatalf("the file changed: %+v", states)
	}
}

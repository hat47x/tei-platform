package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestLoadCarriesDeclaredPeriods(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: w\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	write("semantic/s/x.tei", "namespace s\n\ndata Order {\n    attribute order_date type std.date\n}\n\nperiod s.month {\n    data s.order\n    business-date s.order.order_date\n    states open closing closed\n    closed-rejects create update\n}\n")

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var periods []semantic.Period
	for _, namespace := range loaded.World.Semantics {
		if namespace.ID == "s" {
			periods = namespace.Periods
		}
	}
	if len(periods) != 1 || periods[0].ID != "s.month" {
		t.Fatalf("the loader must carry the declared period: %+v", periods)
	}
	if len(periods[0].ClosedRejects) != 2 {
		t.Fatalf("the period loses the operations it rejects: %+v", periods[0])
	}
}

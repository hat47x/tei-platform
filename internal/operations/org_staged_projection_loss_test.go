package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

// TestOrgStagedProjectionLossAndReversibility verifies the staged axis fixture
// preserves projection-loss expectations and keeps irreversible changes
// read-only.
func TestOrgStagedProjectionLossAndReversibility(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	catalog := CatalogWorld(loaded.World, idx, "", "wf", "sf")
	mapped := MapWorld(loaded.World, catalog)
	if mapped.Effect != "read-only" {
		t.Fatalf("projection must remain read-only: %+v", mapped)
	}
	if mapped.Coverage != "partial-projection" {
		t.Fatalf("projection coverage must declare its limits: %+v", mapped)
	}

	projectionPath := filepath.Join(root, "projection-loss.json")
	raw, err := os.ReadFile(projectionPath)
	if err != nil {
		t.Fatalf("read projection loss: %v", err)
	}
	var projection struct {
		Losses []struct {
			Name        string `json:"name"`
			Expectation string `json:"expectation"`
		} `json:"losses"`
	}
	if err := json.Unmarshal(raw, &projection); err != nil {
		t.Fatalf("parse projection loss: %v", err)
	}
	if len(projection.Losses) == 0 {
		t.Fatal("projection loss must declare at least one loss")
	}
	for _, loss := range projection.Losses {
		if loss.Name == "" || loss.Expectation == "" {
			t.Fatalf("projection loss must not omit name/expectation: %+v", loss)
		}
	}
}

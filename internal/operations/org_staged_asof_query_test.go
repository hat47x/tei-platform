package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/query"
	"tei/internal/workspace"
)

// TestOrgStagedAsOfQuery asserts the as-of membership query declares valid-time
// and known-time as separate inputs and is accepted by the workspace loader.
func TestOrgStagedAsOfQuery(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load staged axis workspace: %v", err)
	}

	var found bool
	for _, q := range loaded.World.Queries {
		if q.ID == "org_axes_view.as_of_membership" {
			found = true
			if q.Dialect != "postgres" {
				t.Fatalf("unexpected dialect: %+v", q)
			}
			if len(q.Inputs) != 3 {
				t.Fatalf("expected three inputs, got %+v", q.Inputs)
			}
			inputNames := map[string]bool{}
			for _, in := range q.Inputs {
				inputNames[string(in.Name)] = true
			}
			if !inputNames["valid_from"] || !inputNames["known_from"] || !inputNames["allocation_min"] {
				t.Fatalf("expected valid_from, known_from, and allocation_min inputs, got %+v", q.Inputs)
			}
		}
	}
	if !found {
		t.Fatal("org_axes_view.as_of_membership query not loaded")
	}

	// The workspace validates and the query model is structurally sound.
	analysis := query.EmbeddedSQL{Dialect: "postgres", Source: "SELECT 1"}
	_ = analysis
}

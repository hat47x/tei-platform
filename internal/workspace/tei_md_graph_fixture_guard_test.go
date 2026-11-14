package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These are the authored Graph/Rule fixtures, not synthetic TEI syntax samples.
// Until an explicitly selected Markdown reader is implemented, Load must not
// turn their managed presence into an empty/partially parsed successful World.
func TestWorkspaceGraphTEIMarkdownFixturesFailClosed(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "tei-md-graph")
	expectedBytes, err := os.ReadFile(filepath.Join(fixtureDir, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var candidate struct {
		Adoption string `json:"adoption"`
		Cases    []struct {
			File string `json:"file"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(expectedBytes, &candidate); err != nil {
		t.Fatal(err)
	}
	// The proposal-only oracle is an inventory of fixtures, not an
	// authorization to interpret them as accepted TEI Meaning.
	if candidate.Adoption != "proposal-only" || len(candidate.Cases) != 5 {
		t.Fatalf("unexpected fixture inventory/acceptance status: %q, %d cases", candidate.Adoption, len(candidate.Cases))
	}

	for _, tc := range candidate.Cases {
		t.Run(tc.File, func(t *testing.T) {
			if filepath.Base(tc.File) != tc.File || !strings.HasSuffix(tc.File, ".tei.md") {
				t.Fatalf("fixture must be a local TEI Markdown file: %q", tc.File)
			}
			raw, err := os.ReadFile(filepath.Join(fixtureDir, tc.File))
			if err != nil {
				t.Fatal(err)
			}
			root := testTEIMDWorkspace(t, "semantic")
			const managedPath = "semantic/graph.tei.md"
			writeTEIMDTestFile(t, root, managedPath, string(raw))

			if _, err := Load(root); err == nil ||
				!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") ||
				!strings.Contains(err.Error(), managedPath) {
				t.Fatalf("managed Graph fixture was not rejected by the unbound parser guard: %v", err)
			}
		})
	}
}

// A draft using the same real authored Graph Markdown does not become a
// Workspace input merely because its suffix is .tei.md.
func TestWorkspaceGraphTEIMarkdownFixtureOutsideSourceRoot(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	root := testTEIMDWorkspace(t, "semantic")
	writeTEIMDTestFile(t, root, "semantic/model.tei", legacyOrgSource)
	writeTEIMDTestFile(t, root, "drafts/graph.tei.md", string(raw))
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("unmanaged Markdown draft unexpectedly blocked legacy TEI loading: %v", err)
	}
	for _, ns := range loaded.World.Semantics {
		if ns.ID == "organization" && len(ns.Data) == 1 && ns.Data[0].ID == "organization.org_unit" {
			return
		}
	}
	t.Fatal("legacy TEI meaning disappeared when an unmanaged Graph draft was present")
}

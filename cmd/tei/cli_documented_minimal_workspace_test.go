package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
)

// The minimal Workspace in docs/getting-started.md ("外部の呼出し元が書く最小の
// 作業領域") is written out literally here. If the documented files stop
// validating, or the documented Canonical IDs stop appearing in the catalog,
// this test fails instead of letting the document drift (#3555).
const documentedMinimalManifest = `# tei.workspace.yaml
workspace: scn001
formatVersion: 1
sources:
  - semantic
`

const documentedMinimalSource = `namespace scn

data RuleCandidate {
    attribute sourceVersion type std.text
}
`

func writeDocumentedMinimalWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(documentedMinimalManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(documentedMinimalSource), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestDocumentedMinimalWorkspaceValidatesAndNamesCanonicalIDs(t *testing.T) {
	root := writeDocumentedMinimalWorkspace(t)
	var out strings.Builder
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 0 {
		t.Fatalf("documented minimal workspace did not validate: code=%d err=%v body=%s", code, err, out.String())
	}
	var report cliValidationReport
	if err := json.Unmarshal([]byte(out.String()), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Error != nil {
		t.Fatalf("documented minimal workspace is not ok: %+v", report)
	}
	if report.DeclarationCounts == nil ||
		report.DeclarationCounts.Namespaces != 1 ||
		report.DeclarationCounts.Data != 1 ||
		report.DeclarationCounts.Attributes != 1 ||
		report.DeclarationCounts.Total != 3 {
		t.Fatalf("documented counts changed: %+v", report.DeclarationCounts)
	}
	if report.Workspace == nil || report.Workspace.Name != "scn001" {
		t.Fatalf("documented workspace name is missing: %+v", report.Workspace)
	}

	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		kind, id, symbol string
	}{
		{kind: "semantic", id: "scn"},
		{kind: "semantic-data", id: "scn.rule_candidate", symbol: "RuleCandidate"},
		{kind: "semantic-attribute", id: "scn.rule_candidate.source_version", symbol: "sourceVersion"},
	} {
		found := false
		for _, entry := range catalog.Entries {
			if entry.Subject.Kind == want.kind && entry.Subject.ID == want.id {
				found = true
				if want.symbol != "" && entry.Symbol != want.symbol {
					t.Fatalf("%s:%s symbol=%q want %q", want.kind, want.id, entry.Symbol, want.symbol)
				}
			}
		}
		if !found {
			t.Fatalf("documented Canonical ID %s:%s is missing from the catalog", want.kind, want.id)
		}
	}
}

// The conversion examples in the document's table are checked against the
// implementation too, so a rule change cannot leave the table wrong.
func TestDocumentedCanonicalIDConversionsHold(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"),
		[]byte("workspace: ids\nformatVersion: 1\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const source = `namespace sales

type OrderStatus from std.code

data Order {
    attribute orderId type std.identifier
    relation OrderLines to sales.order_line one
    candidate-key BusinessKey {
        field sales.order.order_id
    }
}

data OrderLine {
    attribute orderLineId type std.identifier
}

capability SubmitOrder {
    input order sales.order
    outcome submitted sales.order
    effect update sales.order
}
`
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"sales.order_status",       // type
		"sales.submit_order",       // capability
		"sales.order.order_lines",  // relation
		"sales.order.business_key", // candidate-key
		"sales.order.order_id",     // attribute
	} {
		found := false
		for _, entry := range catalog.Entries {
			if entry.Subject.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("documented conversion %q is missing from the catalog", id)
		}
	}
}

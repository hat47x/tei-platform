package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ercrossview "tei/tools/reference-er-table-crossview"
)

func TestCLIExportsSourceWorkspaceOnlyToExplicitNewPrivateFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: er_crossview_test\nsources:\n  - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "semantic", "sales")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	src := `namespace sales
data OrderLine {
    attribute orderNumber type std.code
    attribute lineNumber type std.integer
    candidate-key businessLine {
        field sales.order_line.order_number
        field sales.order_line.line_number
    }
}
`
	if err := os.WriteFile(filepath.Join(dir, "order.tei"), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "er.json")
	if err := run(root, target); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	var view ercrossview.Projection
	if err := json.Unmarshal(content, &view); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range view.Data {
		if d.ID != "sales.order_line" {
			continue
		}
		if len(d.CandidateKeys) != 1 || d.CandidateKeys[0].ID != "sales.order_line.business_line" ||
			len(d.PhysicalMappingIDs) != 0 {
			t.Fatalf("source Data key or unmapped physical view lost: %+v", d)
		}
		found = true
	}
	if !found {
		t.Fatalf("source Data missing: %+v", view.Data)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("projection output is not private: %v %v", info, err)
	}
	if err := run(root, target); err == nil {
		t.Fatal("CLI silently overwrote existing evidence output")
	}
	preserved, err := os.ReadFile(target)
	if err != nil || string(preserved) != string(content) {
		t.Fatal("failed overwrite altered output")
	}
}

func TestCLIRequiresExplicitWorkspaceAndOutput(t *testing.T) {
	if err := run("", filepath.Join(t.TempDir(), "out.json")); err == nil {
		t.Fatal("missing Workspace was accepted")
	}
	if err := run(t.TempDir(), ""); err == nil {
		t.Fatal("missing output was accepted")
	}
}

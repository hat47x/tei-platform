package main

import (
	"path/filepath"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestXTEAExampleWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "xtea-lowcode")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}

	view, err := applicationview.Build(loaded.World, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Items) != 2 || len(view.Unused) != 1 || len(view.Hooks) != 1 || len(view.Commands) != 1 {
		t.Fatalf("unexpected record view: %+v", view)
	}

	plan, err := applicationview.BuildCommandPlan(loaded.World, "order_entry", "order_entry.order_record", "sales.submit_order", map[string]string{"order": "ORD-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Ready || len(plan.Inputs) != 1 {
		t.Fatalf("unexpected command plan: %+v", plan)
	}

	pkg, err := applicationview.BuildPackage(loaded.World, "order_entry", "order_entry.list_to_record")
	if err != nil {
		t.Fatal(err)
	}
	if !pkg.Bound || len(pkg.Members) != 4 {
		t.Fatalf("unexpected package view: %+v", pkg)
	}
}

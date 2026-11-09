package studiohost

import (
	"path/filepath"
	"testing"

	"tei/internal/applicationview"
)

func TestReadOnlyViewsXteaExampleWorkspace(t *testing.T) {
	host, err := NewReadOnly([]string{filepath.Join("..", "..", "examples", "xtea-lowcode")})
	if err != nil {
		t.Fatal(err)
	}

	pattern := host.ViewPattern("workspace-1", "order_entry", "order_entry.order_record")
	if !pattern.OK {
		t.Fatalf("pattern view failed: %+v", pattern.Failure)
	}
	view, ok := pattern.Value.(applicationview.View)
	if !ok || len(view.Items) != 2 || len(view.Hooks) != 1 {
		t.Fatalf("unexpected pattern view: %#v", pattern.Value)
	}

	plan := host.PlanCommand("workspace-1", "order_entry", "order_entry.order_record", "sales.submit_order", map[string]string{"order": "ORD-1"})
	if !plan.OK || plan.Value == nil || !plan.Value.Ready || plan.Value.Period == nil {
		t.Fatalf("unexpected command plan: %+v", plan)
	}

	pkg := host.ViewPackage("workspace-1", "order_entry", "order_entry.list_to_record")
	if !pkg.OK || pkg.Value == nil || !pkg.Value.Bound || len(pkg.Value.Members) != 4 {
		t.Fatalf("unexpected package view: %+v", pkg)
	}
}

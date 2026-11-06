package studiohost

import (
	"path/filepath"
	"testing"

	"tei/internal/applicationview"
)

func referenceWorkspace(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestViewPatternReturnsApplicationView(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	response := host.ViewPattern("workspace-1", "order_entry", "order_entry.submit_order")
	if !response.OK || response.Failure != nil {
		t.Fatalf("response=%+v failure=%+v", response, response.Failure)
	}
	view, ok := response.Value.(applicationview.CommandPatternView)
	if !ok {
		t.Fatalf("value is %T", response.Value)
	}
	if view.Pattern != "order_entry.submit_order" || view.Capability == "" {
		t.Fatalf("view=%+v", view)
	}
}

func TestViewApplicationReturnsSummary(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	response := host.ViewApplication("workspace-1", "order_entry")
	if !response.OK || response.Failure != nil {
		t.Fatalf("response=%+v failure=%+v", response, response.Failure)
	}
	view, ok := response.Value.(applicationview.ApplicationView)
	if !ok {
		t.Fatalf("value is %T", response.Value)
	}
	if len(view.Patterns) == 0 || view.SemanticFingerprint == "" {
		t.Fatalf("view=%+v", view)
	}
}

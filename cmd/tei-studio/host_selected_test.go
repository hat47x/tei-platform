//go:build server

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func TestNormalStudioBridgeUsesHostSelectedLocalInspector(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	bridge := &StudioBridge{host: host}
	// The exported surface is a tripwire: it is what the Wails service publishes.
	// It was 5 while assets/client.mjs already addressed 12 names, so every
	// Workbench view and command action called a method that was never
	// registered (#4295). The seven projections the client names are now
	// registered, and TestStudioBridgeRegistersEveryClientMethod keeps the two
	// sides in step instead of leaving it to this count alone.
	//
	// The two SHO relay methods are counted here too: they are Go-facing, so
	// assets/client.mjs does not name them, and this count is what keeps the
	// surface from growing without anyone saying so. 13 client-named + 2
	// Go-facing = 15, plus NavigateThreeView, which assets/three-view/client.mjs
	// names: 16 exported methods on the bridge and nothing else.
	if reflect.TypeOf(bridge).NumMethod() != 16 {
		t.Fatal("Studio Bridge exported surface changed")
	}
	got := bridge.InspectSubject("workspace-1", "semantic-data", "sales.order")
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	trustedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := operations.InspectSubject(trustedRoot, sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Value == nil || got.Failure != nil ||
		got.Value.Subject.Kind != "semantic-data" ||
		got.Value.Subject.ID != "sales.order" ||
		!reflect.DeepEqual(*got.Value, want) {
		t.Fatalf("Studio selected Inspector read failed: %+v", got)
	}
}

func TestStudioSelectedInspectorStillEnforcesWorkspaceAllowlist(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	bridge := &StudioBridge{host: host}
	for _, id := range []string{"workspace-2", "", "../workspace", "workspace-1/../"} {
		r := bridge.InspectSubject(id, "semantic-data", "sales.order")
		if r.OK || r.Failure == nil || r.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("Studio selected Inspector accepted workspace %q: %+v", id, r)
		}
	}
}

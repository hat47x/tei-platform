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
	if reflect.TypeOf(bridge).NumMethod() != 5 {
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

//go:build server

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// Two separately linked entries with byte-identical answers, and one that
// reads a Workspace other than the root the Host handed it.
var studioProductCalls struct{ a, b, foreign int }
var studioProductForeignRoot string

func studioProductEntryA(root string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	studioProductCalls.a++
	return operations.InspectSubject(root, s)
}

func studioProductEntryB(root string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	studioProductCalls.b++
	return operations.InspectSubject(root, s)
}

func studioProductEntryForeign(_ string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	studioProductCalls.foreign++
	return operations.InspectSubject(studioProductForeignRoot, s)
}

// Independent expectations, written by hand.
const (
	studioWantEntryA       = "tei/cmd/tei-studio.studioProductEntryA"
	studioWantEntryB       = "tei/cmd/tei-studio.studioProductEntryB"
	studioWantEntryForeign = "tei/cmd/tei-studio.studioProductEntryForeign"
)

func studioProductWorkspace(t *testing.T, name, extra string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - semantic\n"
	source := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n" + extra + "}\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "order.tei"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func studioProductBridge(t *testing.T, roots []string, selected string, inventory []inspectoradapter.LocalProvider) (*StudioBridge, *[]inspectoradapter.Invocation) {
	t.Helper()
	inspector, err := inspectoradapter.NewLocalInspectorHostSelecting(selected, inventory)
	if err != nil {
		t.Fatal(err)
	}
	seen := &[]inspectoradapter.Invocation{}
	inspector.Observe(func(i inspectoradapter.Invocation) { *seen = append(*seen, i) })
	host, err := newStudioHostWithInspector(roots, inspector)
	if err != nil {
		t.Fatal(err)
	}
	return &StudioBridge{host: host}, seen
}

// The Wails Bridge's Go-facing method entered through the Host's selection:
// B is selected and B's linked function is entered; A would answer the same.
func TestStudioSelectedBEntersBWithAnAnswerIdenticalToA(t *testing.T) {
	studioProductCalls = struct{ a, b, foreign int }{}
	root := studioProductWorkspace(t, "alpha", "")
	inventory := []inspectoradapter.LocalProvider{
		{ID: "inspector-A", Entry: studioWantEntryA, Inspect: studioProductEntryA},
		{ID: "inspector-B", Entry: studioWantEntryB, Inspect: studioProductEntryB},
	}
	bridge, seen := studioProductBridge(t, []string{root}, "inspector-B", inventory)
	fromB := bridge.InspectSubject("workspace-1", "semantic-data", "sales.order")
	if !fromB.OK || fromB.Value == nil || fromB.Failure != nil {
		t.Fatalf("selected B failed through the Studio Bridge: %+v", fromB)
	}
	if studioProductCalls.a != 0 || studioProductCalls.b != 1 || len(*seen) != 1 ||
		(*seen)[0].SelectedProviderID != "inspector-B" || (*seen)[0].ObservedEntry != studioWantEntryB ||
		!(*seen)[0].OriginVerified {
		t.Fatalf("Studio did not enter B on the allowlisted root: calls=%+v seen=%+v", studioProductCalls, *seen)
	}
	bridgeA, seenA := studioProductBridge(t, []string{root}, "inspector-A", inventory)
	fromA := bridgeA.InspectSubject("workspace-1", "semantic-data", "sales.order")
	a, _ := json.Marshal(fromA)
	b, _ := json.Marshal(fromB)
	if !fromA.OK || !bytes.Equal(a, b) {
		t.Fatalf("A and B were expected to answer identically:\n%s\n%s", a, b)
	}
	if len(*seenA) != 1 || (*seenA)[0].ObservedEntry != studioWantEntryA {
		t.Fatalf("selecting A did not enter A: %+v", *seenA)
	}
}

func TestStudioHostIsNotComposedForBWiredToTheCodeOfA(t *testing.T) {
	studioProductCalls = struct{ a, b, foreign int }{}
	inspector, err := inspectoradapter.NewLocalInspectorHostSelecting("inspector-B", []inspectoradapter.LocalProvider{
		{ID: "inspector-A", Entry: studioWantEntryA, Inspect: studioProductEntryA},
		{ID: "inspector-B", Entry: studioWantEntryB, Inspect: studioProductEntryA},
	})
	if err == nil || inspector != nil {
		t.Fatalf("B wired to A's code was composed for Studio: %v", err)
	}
	if studioProductCalls.a != 0 || studioProductCalls.b != 0 {
		t.Fatalf("a refused composition entered a Provider: %+v", studioProductCalls)
	}
}

// Both roots are allowlisted and declare the same Subject ID. The renderer
// asks for workspace-1; the Provider answers from workspace-2's directory.
func TestStudioRefusesAValidAnswerFromAnotherAllowlistedWorkspace(t *testing.T) {
	studioProductCalls = struct{ a, b, foreign int }{}
	alpha := studioProductWorkspace(t, "alpha", "")
	studioProductForeignRoot = studioProductWorkspace(t, "beta", "    attribute customerId type std.identifier\n")
	bridge, seen := studioProductBridge(t, []string{alpha, studioProductForeignRoot}, "inspector-B",
		[]inspectoradapter.LocalProvider{
			{ID: "inspector-B", Entry: studioWantEntryForeign, Inspect: studioProductEntryForeign},
		})
	got := bridge.InspectSubject("workspace-1", "semantic-data", "sales.order")
	raw, _ := json.Marshal(got)
	if got.OK || got.Value != nil || got.Failure == nil {
		t.Fatalf("Studio returned another Workspace's answer for workspace-1: %s", raw)
	}
	if bytes.Contains(raw, []byte("beta")) || bytes.Contains(raw, []byte(filepath.Base(studioProductForeignRoot))) {
		t.Fatalf("Studio failure disclosed the foreign Workspace: %s", raw)
	}
	if studioProductCalls.foreign != 1 || len(*seen) != 1 || (*seen)[0].OriginVerified ||
		(*seen)[0].ReturnedWorkspace == nil || (*seen)[0].ReturnedWorkspace.Name != "beta" {
		t.Fatalf("Host observation is missing the mismatch: %+v", *seen)
	}
	// Asking for workspace-2 makes the same answer the right one.
	if again := bridge.InspectSubject("workspace-2", "semantic-data", "sales.order"); !again.OK {
		t.Fatalf("the Provider's own root was refused: %+v", again)
	}
}

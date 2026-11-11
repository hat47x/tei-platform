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
var cliProductCalls struct{ a, b, foreign int }
var cliProductForeignRoot string

func cliProductEntryA(root string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	cliProductCalls.a++
	return operations.InspectSubject(root, s)
}

func cliProductEntryB(root string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	cliProductCalls.b++
	return operations.InspectSubject(root, s)
}

func cliProductEntryForeign(_ string, s sourceindex.Subject) (operations.SubjectInspection, error) {
	cliProductCalls.foreign++
	return operations.InspectSubject(cliProductForeignRoot, s)
}

// Independent expectations, written by hand.
const (
	cliWantEntryA       = "tei/cmd/tei.cliProductEntryA"
	cliWantEntryB       = "tei/cmd/tei.cliProductEntryB"
	cliWantEntryForeign = "tei/cmd/tei.cliProductEntryForeign"
)

func cliProductWorkspace(t *testing.T, name, extra string) string {
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

func cliProductRun(t *testing.T, root, selected string, inventory []inspectoradapter.LocalProvider) (int, cliInspectionReport, []byte, []inspectoradapter.Invocation) {
	t.Helper()
	var seen []inspectoradapter.Invocation
	compose := func() (*inspectoradapter.LocalInspectorHost, error) {
		host, err := inspectoradapter.NewLocalInspectorHostSelecting(selected, inventory)
		if err == nil {
			host.Observe(func(i inspectoradapter.Invocation) { seen = append(seen, i) })
		}
		return host, err
	}
	var out bytes.Buffer
	exit, err := inspectWorkspaceJSONFromHost(&out, compose, root, "", "semantic-data", "sales.order", 0)
	if err != nil {
		t.Fatal(err)
	}
	var report cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("CLI did not emit one JSON document: %s", out.String())
	}
	return exit, report, out.Bytes(), seen
}

// The CLI transport entered through the Host's selection: B is selected, B's
// linked function is the one entered, and A is never called although A would
// print the same bytes.
func TestCLISelectedBEntersBWithAnAnswerIdenticalToA(t *testing.T) {
	cliProductCalls = struct{ a, b, foreign int }{}
	root := cliProductWorkspace(t, "alpha", "")
	inventory := []inspectoradapter.LocalProvider{
		{ID: "inspector-A", Entry: cliWantEntryA, Inspect: cliProductEntryA},
		{ID: "inspector-B", Entry: cliWantEntryB, Inspect: cliProductEntryB},
	}
	exit, report, fromB, seen := cliProductRun(t, root, "inspector-B", inventory)
	if exit != 0 || !report.OK || report.Result == nil {
		t.Fatalf("selected B failed through the CLI: exit=%d %+v", exit, report)
	}
	if cliProductCalls.a != 0 || cliProductCalls.b != 1 || len(seen) != 1 ||
		seen[0].SelectedProviderID != "inspector-B" || seen[0].ObservedEntry != cliWantEntryB ||
		!seen[0].OriginVerified {
		t.Fatalf("CLI did not enter B on the selected root: calls=%+v seen=%+v", cliProductCalls, seen)
	}
	exitA, _, fromA, seenA := cliProductRun(t, root, "inspector-A", inventory)
	if exitA != 0 || !bytes.Equal(fromA, fromB) {
		t.Fatalf("A and B were expected to print identical bytes:\n%s\n%s", fromA, fromB)
	}
	if len(seenA) != 1 || seenA[0].ObservedEntry != cliWantEntryA || cliProductCalls.a != 1 || cliProductCalls.b != 1 {
		t.Fatalf("selecting A did not enter A: calls=%+v seen=%+v", cliProductCalls, seenA)
	}
}

func TestCLIRefusesSelectedBWiredToTheCodeOfA(t *testing.T) {
	cliProductCalls = struct{ a, b, foreign int }{}
	root := cliProductWorkspace(t, "alpha", "")
	exit, report, _, seen := cliProductRun(t, root, "inspector-B", []inspectoradapter.LocalProvider{
		{ID: "inspector-A", Entry: cliWantEntryA, Inspect: cliProductEntryA},
		{ID: "inspector-B", Entry: cliWantEntryB, Inspect: cliProductEntryA},
	})
	if exit != 2 || report.OK || report.Result != nil || report.Error == nil ||
		report.Error.Code != "INSPECTION_PROVIDER_UNAVAILABLE" {
		t.Fatalf("CLI served B through A's code: exit=%d %+v", exit, report)
	}
	if cliProductCalls.a != 0 || cliProductCalls.b != 0 || len(seen) != 0 {
		t.Fatalf("a refused selection still entered a Provider: %+v %+v", cliProductCalls, seen)
	}
}

func TestCLIRefusesAValidAnswerFromAnotherWorkspace(t *testing.T) {
	cliProductCalls = struct{ a, b, foreign int }{}
	root := cliProductWorkspace(t, "alpha", "")
	cliProductForeignRoot = cliProductWorkspace(t, "beta", "    attribute customerId type std.identifier\n")
	exit, report, raw, seen := cliProductRun(t, root, "inspector-B", []inspectoradapter.LocalProvider{
		{ID: "inspector-B", Entry: cliWantEntryForeign, Inspect: cliProductEntryForeign},
	})
	if exit != 2 || report.OK || report.Result != nil || report.Error == nil {
		t.Fatalf("CLI printed another Workspace's answer: exit=%d %s", exit, raw)
	}
	// Neither root path nor the foreign Workspace name may reach the output.
	if bytes.Contains(raw, []byte("beta")) || bytes.Contains(raw, []byte(filepath.Base(cliProductForeignRoot))) {
		t.Fatalf("CLI failure disclosed the foreign Workspace: %s", raw)
	}
	if cliProductCalls.foreign != 1 || len(seen) != 1 || seen[0].OriginVerified ||
		seen[0].ReturnedWorkspace == nil || seen[0].ReturnedWorkspace.Name != "beta" {
		t.Fatalf("Host observation is missing the mismatch: %+v", seen)
	}
}

// The normal dispatch still selects the reference Provider, and the entry it
// enters is the canonical Operation itself.
func TestNormalCLIReferenceSelectionEntersTheCanonicalOperation(t *testing.T) {
	root := cliProductWorkspace(t, "alpha", "")
	exit, report, _, seen := cliProductRun(t, root,
		inspectoradapter.LocalInspectorProviderID, inspectoradapter.ReferenceInventory())
	if exit != 0 || !report.OK || len(seen) != 1 ||
		seen[0].ObservedEntry != "tei/internal/operations.InspectSubject" || !seen[0].OriginVerified {
		t.Fatalf("reference selection did not enter the canonical Operation: exit=%d %+v", exit, seen)
	}
}

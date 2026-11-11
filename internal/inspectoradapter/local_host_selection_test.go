package inspectoradapter

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// Two separately linked entries that give byte-identical answers. Only the
// function the Host enters tells them apart, which is the point of the tests.
var selectionCalls struct{ a, b, foreign int }

func selectionEntryA(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	selectionCalls.a++
	return operations.InspectSubject(root, subject)
}

func selectionEntryB(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	selectionCalls.b++
	return operations.InspectSubject(root, subject)
}

// selectionForeignRoot is the Workspace a dishonest entry reads instead of the
// root the Host handed it.
var selectionForeignRoot string

func selectionEntryForeign(_ string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	selectionCalls.foreign++
	return operations.InspectSubject(selectionForeignRoot, subject)
}

// These are the independent expectations: the symbols are written here by
// hand, not derived from the inventory under test.
const (
	wantEntryA       = "tei/internal/inspectoradapter.selectionEntryA"
	wantEntryB       = "tei/internal/inspectoradapter.selectionEntryB"
	wantEntryForeign = "tei/internal/inspectoradapter.selectionEntryForeign"
)

func twoProviderInventory() []LocalProvider {
	return []LocalProvider{
		{ID: "inspector-A", Entry: wantEntryA, Inspect: selectionEntryA},
		{ID: "inspector-B", Entry: wantEntryB, Inspect: selectionEntryB},
	}
}

func selectionWorkspace(t *testing.T, name, extraAttribute string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - semantic\n"
	source := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n" + extraAttribute + "}\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "order.tei"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

var selectionSubject = sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}

func TestHostSelectingBEntersBAndNotA(t *testing.T) {
	selectionCalls = struct{ a, b, foreign int }{}
	root := selectionWorkspace(t, "alpha", "")
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", twoProviderInventory())
	if err != nil {
		t.Fatal(err)
	}
	var seen []Invocation
	host.Observe(func(i Invocation) { seen = append(seen, i) })
	selected, err := host.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := selected.Inspect(selectionSubject)
	if err != nil {
		t.Fatal(err)
	}
	if selectionCalls.a != 0 || selectionCalls.b != 1 {
		t.Fatalf("selected B but calls were A=%d B=%d", selectionCalls.a, selectionCalls.b)
	}
	if len(seen) != 1 || seen[0].SelectedProviderID != "inspector-B" ||
		seen[0].RegisteredProviderID != "inspector-B" ||
		seen[0].ObservedEntry != wantEntryB || !seen[0].OriginVerified {
		t.Fatalf("Host observation does not show B entered on the selected root: %+v", seen)
	}
	// The same request through A is byte-identical. The answer therefore
	// cannot be what distinguishes the two Providers.
	hostA, err := newLocalInspectorHost(inspectorArtifact, "inspector-A", twoProviderInventory())
	if err != nil {
		t.Fatal(err)
	}
	selectedA, err := hostA.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	fromA, err := selectedA.Inspect(selectionSubject)
	if err != nil || !bytes.Equal(fromA.Canonical, got.Canonical) ||
		fromA.WorldFingerprint != got.WorldFingerprint ||
		fromA.SourceIndexFingerprint != got.SourceIndexFingerprint {
		t.Fatalf("A and B were expected to answer identically: %+v %+v %v", fromA, got, err)
	}
}

func TestHostRejectsSelectedBWiredToTheCodeOfA(t *testing.T) {
	selectionCalls = struct{ a, b, foreign int }{}
	// The label says B and the declared entry says B, but the function value
	// is A's. Its answer would be identical. The Host must not compose it.
	miswired := []LocalProvider{
		{ID: "inspector-A", Entry: wantEntryA, Inspect: selectionEntryA},
		{ID: "inspector-B", Entry: wantEntryB, Inspect: selectionEntryA},
	}
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", miswired)
	if err == nil || host != nil || !strings.Contains(err.Error(), wantEntryA) {
		t.Fatalf("B wired to A's code was composed: host=%v err=%v", host, err)
	}
	if selectionCalls.a != 0 || selectionCalls.b != 0 {
		t.Fatalf("a refused composition still entered a Provider: %+v", selectionCalls)
	}
}

// When an inventory consistently declares A's code under the ID B, the Host
// has no second fact to compare and composes it. The Provider ID alone is then
// a self-consistent label. The observation still names the entered function,
// so an independent expectation (#1180/#2514) can fail it. This test fixes
// that boundary: the Host does not claim to detect this case by itself.
func TestObservationExposesACodeUnderTheLabelBThatTheHostCannotJudge(t *testing.T) {
	selectionCalls = struct{ a, b, foreign int }{}
	relabelled := []LocalProvider{{ID: "inspector-B", Entry: wantEntryA, Inspect: selectionEntryA}}
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", relabelled)
	if err != nil {
		t.Fatal(err)
	}
	var seen Invocation
	host.Observe(func(i Invocation) { seen = i })
	selected, err := host.Bind(selectionWorkspace(t, "alpha", ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(selectionSubject); err != nil {
		t.Fatal(err)
	}
	if seen.RegisteredProviderID != "inspector-B" || seen.ObservedEntry == wantEntryB ||
		seen.ObservedEntry != wantEntryA || selectionCalls.a != 1 {
		t.Fatalf("observation did not separate the label from the entered code: %+v", seen)
	}
}

func TestHostSelectionFailsClosedForUnknownOrIncompleteInventory(t *testing.T) {
	if _, err := newLocalInspectorHost(inspectorArtifact, "inspector-C", twoProviderInventory()); !errors.Is(err, composition.ErrSelectionMismatch) {
		t.Fatalf("a Provider outside the inventory was selected: %v", err)
	}
	for name, inventory := range map[string][]LocalProvider{
		"no-entry":  {{ID: "inspector-B", Inspect: selectionEntryB}},
		"no-code":   {{ID: "inspector-B", Entry: wantEntryB}},
		"no-id":     {{Entry: wantEntryB, Inspect: selectionEntryB}},
		"duplicate": {{ID: "inspector-B", Entry: wantEntryB, Inspect: selectionEntryB}, {ID: "inspector-B", Entry: wantEntryA, Inspect: selectionEntryA}},
		"closure": {{ID: "inspector-B", Entry: wantEntryB, Inspect: func(root string, s sourceindex.Subject) (operations.SubjectInspection, error) {
			return selectionEntryB(root, s)
		}}},
	} {
		if _, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", inventory); err == nil {
			t.Fatalf("%s inventory was composed", name)
		}
	}
}

func TestHostRejectsAValidAnswerReadFromAnotherWorkspace(t *testing.T) {
	selectionCalls = struct{ a, b, foreign int }{}
	hostRoot := selectionWorkspace(t, "alpha", "")
	selectionForeignRoot = selectionWorkspace(t, "beta", "    attribute customerId type std.identifier\n")
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", []LocalProvider{
		{ID: "inspector-B", Entry: wantEntryForeign, Inspect: selectionEntryForeign},
	})
	if err != nil {
		t.Fatal(err)
	}
	var seen Invocation
	host.Observe(func(i Invocation) { seen = i })
	selected, err := host.Bind(hostRoot)
	if err != nil {
		t.Fatal(err)
	}
	// The foreign answer is well formed for the same Subject ID: it passes
	// the projection contract on its own.
	foreign, err := operations.InspectSubject(selectionForeignRoot, selectionSubject)
	if err != nil || operations.ValidateInspectionProjection(selectionSubject, foreign) != nil {
		t.Fatalf("the foreign answer must itself be valid for this test: %v", err)
	}
	got, err := selected.Inspect(selectionSubject)
	if !errors.Is(err, ErrWorkspaceOriginUnverified) || got.Canonical != nil || selectionCalls.foreign != 1 {
		t.Fatalf("an answer from another Workspace was accepted: %+v err=%v", got, err)
	}
	if seen.OriginVerified || seen.ReturnedWorkspace == nil ||
		seen.ReturnedWorkspace.Name != "beta" || seen.HostManifest.SHA256 == "" ||
		seen.HostManifest == seen.ReturnedWorkspace.Manifest || seen.HostRoot != hostRoot {
		t.Fatalf("observation does not separate the Host's root from the Provider's claim: %+v", seen)
	}
}

func TestHostRejectsAnAnswerThatNamesNoWorkspace(t *testing.T) {
	root := selectionWorkspace(t, "alpha", "")
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", []LocalProvider{
		{ID: "inspector-B", Entry: "tei/internal/inspectoradapter.selectionEntryAnonymous", Inspect: selectionEntryAnonymous},
	})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := host.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(selectionSubject); !errors.Is(err, ErrWorkspaceOriginUnverified) {
		t.Fatalf("an answer without a Workspace identity was accepted: %v", err)
	}
}

func selectionEntryAnonymous(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	value, err := operations.InspectSubject(root, subject)
	value.Workspace = nil
	return value, err
}

// The Host compares manifest bytes only. A Provider that reads another root
// whose manifest is byte-identical, but whose semantic Source differs, is NOT
// caught here. This test fixes that known limit so that the manifest check is
// not later read as a Source origin proof (#3524/#3479 own the Source bytes).
func TestOriginCheckDoesNotCoverSourceBytesBehindAnIdenticalManifest(t *testing.T) {
	selectionCalls = struct{ a, b, foreign int }{}
	hostRoot := selectionWorkspace(t, "alpha", "")
	selectionForeignRoot = selectionWorkspace(t, "alpha", "    attribute customerId type std.identifier\n")
	host, err := newLocalInspectorHost(inspectorArtifact, "inspector-B", []LocalProvider{
		{ID: "inspector-B", Entry: wantEntryForeign, Inspect: selectionEntryForeign},
	})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := host.Bind(hostRoot)
	if err != nil {
		t.Fatal(err)
	}
	honest, err := operations.InspectSubject(hostRoot, selectionSubject)
	if err != nil {
		t.Fatal(err)
	}
	got, err := selected.Inspect(selectionSubject)
	if err != nil {
		t.Fatalf("the limit moved: an identical manifest is now rejected (%v); update the claim", err)
	}
	if bytes.Equal(got.Canonical, honest.Canonical) {
		t.Fatal("the two roots were expected to differ in semantic Source")
	}
}

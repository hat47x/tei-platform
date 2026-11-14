package threeviewadapter

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/workspace"
)

const testDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000001"
const testSource = "design/model.src"

// answer is a smallest valid three-view answer about root: one data subject
// read from nothing but the root's own manifest. The adapter is tested with
// it so that this package needs no real Provider.
func answer(root string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	manifest, err := workspace.ObserveManifestIdentity(root)
	if err != nil {
		return operations.ThreeViewNavigation{}, err
	}
	limits := []string{"test-answer-is-not-a-reading-of-any-source"}
	return operations.ThreeViewNavigation{
		SchemaVersion: operations.ThreeViewNavigationSchemaVersion,
		Effect:        "read-only",
		Workspace:     &workspace.WorkspaceIdentity{Manifest: manifest},
		Source: operations.ThreeViewSource{
			Path: r.SourcePath, Format: "test", ArtifactDigest: testDigest,
			MappingFingerprint: "map", DerivationRevision: "rev", ProjectionFingerprint: "fp",
			Coverage: operations.ThreeViewCoverage{
				Scope: operations.ThreeViewScopeSingleSource, Status: operations.ThreeViewCoverageComplete,
			},
		},
		Subjects: []operations.ThreeViewSubject{{
			Occurrence: "d", View: operations.ThreeViewData,
			SourceFact: operations.ThreeViewSourceFact{
				Kind: "Entity", Locator: "/Entity[1]", SourceIDStatus: operations.ThreeViewSourceIDMissing,
			},
			Candidate: operations.ThreeViewCandidate{
				Nature: operations.ThreeViewNatureStructuralCandidate, AuthorityLimits: limits,
			},
			Meaning: operations.ThreeViewMeaning{Status: operations.ThreeViewMeaningNotBound},
		}},
		Relations:       []operations.ThreeViewRelation{},
		CRUD:            []operations.ThreeViewCRUDRow{},
		AuthorityLimits: limits,
	}, nil
}

// Two separately linked entries with identical answers, one that answers
// about a Workspace other than the root the Host handed it, and one that
// presents its candidate as an accepted meaning.
var calls struct{ a, b, foreign int }
var foreignRoot string

func entryA(root string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	calls.a++
	return answer(root, r)
}

func entryB(root string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	calls.b++
	return answer(root, r)
}

func entryForeign(_ string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	calls.foreign++
	return answer(foreignRoot, r)
}

func entryForged(root string, r operations.ThreeViewRequest) (operations.ThreeViewNavigation, error) {
	value, err := answer(root, r)
	if err == nil {
		value.Subjects[0].Meaning.Status = "accepted"
	}
	return value, err
}

// Independent expectations, written by hand.
const (
	wantEntryA       = "tei/internal/threeviewadapter.entryA"
	wantEntryB       = "tei/internal/threeviewadapter.entryB"
	wantEntryForeign = "tei/internal/threeviewadapter.entryForeign"
	wantEntryForged  = "tei/internal/threeviewadapter.entryForged"
)

func scratchWorkspace(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	manifest := "workspace: " + name + "\nformatVersion: 1\nsources:\n  - design\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func bound(t *testing.T, selected string, inventory []Provider, root string) (*Selected, *[]Invocation) {
	t.Helper()
	host, err := newHost(testDigest, selected, inventory)
	if err != nil {
		t.Fatal(err)
	}
	seen := &[]Invocation{}
	host.Observe(func(i Invocation) { *seen = append(*seen, i) })
	service, err := host.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	return service, seen
}

// B is selected and B's linked function is entered. A would answer the same,
// so only the observed entry tells them apart.
func TestSelectedBEntersBAlthoughAAnswersIdentically(t *testing.T) {
	calls = struct{ a, b, foreign int }{}
	root := scratchWorkspace(t, "alpha")
	inventory := []Provider{
		{ID: "three-view-A", Entry: wantEntryA, Navigate: entryA},
		{ID: "three-view-B", Entry: wantEntryB, Navigate: entryB},
	}
	request := operations.ThreeViewRequest{SourcePath: testSource}
	serviceB, seenB := bound(t, "three-view-B", inventory, root)
	fromB, err := serviceB.Navigate(request)
	if err != nil {
		t.Fatal(err)
	}
	if calls.a != 0 || calls.b != 1 || len(*seenB) != 1 ||
		(*seenB)[0].SelectedProviderID != "three-view-B" || (*seenB)[0].RegisteredProviderID != "three-view-B" ||
		(*seenB)[0].ObservedEntry != wantEntryB || !(*seenB)[0].OriginVerified ||
		(*seenB)[0].Request.SourcePath != testSource || (*seenB)[0].HostRoot != root {
		t.Fatalf("B was selected but B was not entered: calls=%+v seen=%+v", calls, *seenB)
	}
	serviceA, seenA := bound(t, "three-view-A", inventory, root)
	fromA, err := serviceA.Navigate(request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromA, fromB) {
		t.Fatal("A and B were expected to answer identically")
	}
	if len(*seenA) != 1 || (*seenA)[0].ObservedEntry != wantEntryA {
		t.Fatalf("selecting A did not enter A: %+v", *seenA)
	}
}

func TestHostIsNotComposedForBWiredToTheCodeOfA(t *testing.T) {
	calls = struct{ a, b, foreign int }{}
	host, err := newHost(testDigest, "three-view-B", []Provider{
		{ID: "three-view-A", Entry: wantEntryA, Navigate: entryA},
		{ID: "three-view-B", Entry: wantEntryB, Navigate: entryA},
	})
	if err == nil || host != nil {
		t.Fatalf("B wired to A's code was composed: %v", err)
	}
	if calls.a != 0 || calls.b != 0 {
		t.Fatalf("a refused composition entered a Provider: %+v", calls)
	}
}

func TestUnknownOrIncompleteSelectionFailsClosed(t *testing.T) {
	inventory := []Provider{{ID: "three-view-A", Entry: wantEntryA, Navigate: entryA}}
	if host, err := newHost(testDigest, "three-view-missing", inventory); err == nil || host != nil {
		t.Fatalf("a Provider outside the inventory was selected: %v", err)
	}
	if host, err := newHost(testDigest, "", inventory); err == nil || host != nil {
		t.Fatalf("an empty selection picked a default Provider: %v", err)
	}
	if host, err := newHost(testDigest, "three-view-A", append(inventory, inventory[0])); err == nil || host != nil {
		t.Fatalf("a repeated Provider ID was accepted: %v", err)
	}
	if host, err := newHost("not-a-digest", "three-view-A", inventory); err == nil || host != nil {
		t.Fatalf("an invalid host digest was accepted: %v", err)
	}
	if host, err := newHost(testDigest, "three-view-A", []Provider{{ID: "three-view-A", Entry: wantEntryA}}); err == nil || host != nil {
		t.Fatalf("an entry without a function was accepted: %v", err)
	}
}

// The Provider answers correctly, but about another Workspace's directory.
func TestAnswerFromAnotherWorkspaceIsRefused(t *testing.T) {
	calls = struct{ a, b, foreign int }{}
	alpha := scratchWorkspace(t, "alpha")
	foreignRoot = scratchWorkspace(t, "beta")
	service, seen := bound(t, "three-view-B",
		[]Provider{{ID: "three-view-B", Entry: wantEntryForeign, Navigate: entryForeign}}, alpha)
	value, err := service.Navigate(operations.ThreeViewRequest{SourcePath: testSource})
	if !errors.Is(err, ErrWorkspaceOriginUnverified) || value.SchemaVersion != "" {
		t.Fatalf("an answer about another Workspace was returned: %v %+v", err, value.Source)
	}
	if strings.Contains(err.Error(), foreignRoot) {
		t.Fatalf("the refusal discloses the other root: %v", err)
	}
	if calls.foreign != 1 || len(*seen) != 1 || (*seen)[0].OriginVerified {
		t.Fatalf("the Host observation is missing the mismatch: %+v", *seen)
	}
}

// The Host checks the answer against the Operation contract, so a Provider
// cannot present a candidate as an accepted meaning.
func TestAnswerClaimingAnAcceptedMeaningIsRefused(t *testing.T) {
	service, seen := bound(t, "three-view-forged",
		[]Provider{{ID: "three-view-forged", Entry: wantEntryForged, Navigate: entryForged}},
		scratchWorkspace(t, "alpha"))
	value, err := service.Navigate(operations.ThreeViewRequest{SourcePath: testSource})
	if err == nil || len(value.Subjects) != 0 {
		t.Fatalf("a forged accepted meaning reached the caller: %+v", value.Subjects)
	}
	if len(*seen) != 1 || (*seen)[0].ObservedEntry != wantEntryForged {
		t.Fatalf("the forged Provider was not the one entered: %+v", *seen)
	}
}

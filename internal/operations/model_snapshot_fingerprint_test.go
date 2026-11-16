package operations

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
)

// referenceLocateSubject is one subject the reference Workspace's model catalog
// lists, so a location read and an inspect read of it describe one Workspace.
func referenceLocateSubject() sourceindex.Subject {
	return sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
}

// assertStaleSnapshot reports the one code a refused precondition must carry.
// The code is compared literally, because a caller matches on the string it
// receives over the wire rather than on a Go constant.
func assertStaleSnapshot(t *testing.T, err error) {
	t.Helper()
	var inspection *InspectionError
	if !errors.As(err, &inspection) {
		t.Fatalf("a stale precondition was not reported as an inspection error: %v", err)
	}
	if inspection.Code != "STALE_SNAPSHOT" {
		t.Fatalf("stale precondition code = %q, want STALE_SNAPSHOT", inspection.Code)
	}
}

// model locate answers a navigation question over the same load inspect reads,
// so a caller comparing two of its reads needs the same precondition inspect
// already takes. A matching value passes and a stale one fails closed.
func TestLocateWorkspaceWithOptionsAcceptsMatchingSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	subject := referenceLocateSubject()
	first, err := LocateWorkspaceWithOptions(root, subject, 0, LocateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotFingerprint == "" {
		t.Fatalf("location carries no snapshot fingerprint: %+v", first)
	}
	second, err := LocateWorkspaceWithOptions(root, subject, 0,
		LocateOptions{ExpectedFingerprint: first.SnapshotFingerprint})
	if err != nil {
		t.Fatalf("unchanged Workspace rejected its own snapshot fingerprint: %v", err)
	}
	if second.SnapshotFingerprint != first.SnapshotFingerprint {
		t.Fatalf("snapshot fingerprint is not stable across reads: %s vs %s",
			first.SnapshotFingerprint, second.SnapshotFingerprint)
	}
	if want := InspectionSnapshotFingerprint(
		first.WorldFingerprint, first.SourceIndexFingerprint,
	); first.SnapshotFingerprint != want {
		t.Fatalf("snapshot fingerprint is not the composition of the two halves: %s want %s",
			first.SnapshotFingerprint, want)
	}
}

func TestLocateWorkspaceWithOptionsRejectsStaleSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	_, err := LocateWorkspaceWithOptions(root, referenceLocateSubject(), 0,
		LocateOptions{ExpectedFingerprint: "sha256:not-this-workspace"})
	assertStaleSnapshot(t, err)
}

func TestLocateWorkspaceWithOptionsWithoutExpectedFingerprintPerformsNoCheck(t *testing.T) {
	root := referenceWorkspaceRoot()
	subject := referenceLocateSubject()
	if _, err := LocateWorkspaceWithOptions(root, subject, 0, LocateOptions{}); err != nil {
		t.Fatalf("an omitted fingerprint was treated as a check: %v", err)
	}
	// The unqualified entry point keeps its behaviour for existing callers.
	if _, err := LocateWorkspace(root, subject, 0); err != nil {
		t.Fatalf("LocateWorkspace changed behaviour: %v", err)
	}
}

// model overview is the other half of the same gap: it reports the two
// fingerprints and accepts neither. The precondition is judged before the
// section list is cut, so a stale value is refused however large -limit is.
func TestOverviewWorkspaceWithOptionsAcceptsMatchingSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	first, err := OverviewWorkspaceWithOptions(root, 0, OverviewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotFingerprint == "" {
		t.Fatalf("overview carries no snapshot fingerprint: %+v", first)
	}
	second, err := OverviewWorkspaceWithOptions(root, 0,
		OverviewOptions{ExpectedFingerprint: first.SnapshotFingerprint})
	if err != nil {
		t.Fatalf("unchanged Workspace rejected its own snapshot fingerprint: %v", err)
	}
	if second.SnapshotFingerprint != first.SnapshotFingerprint {
		t.Fatalf("snapshot fingerprint is not stable across reads: %s vs %s",
			first.SnapshotFingerprint, second.SnapshotFingerprint)
	}
	if want := InspectionSnapshotFingerprint(
		first.WorldFingerprint, first.SourceIndexFingerprint,
	); first.SnapshotFingerprint != want {
		t.Fatalf("snapshot fingerprint is not the composition of the two halves: %s want %s",
			first.SnapshotFingerprint, want)
	}
}

func TestOverviewWorkspaceWithOptionsRejectsStaleSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	_, err := OverviewWorkspaceWithOptions(root, 0,
		OverviewOptions{ExpectedFingerprint: "sha256:not-this-workspace"})
	assertStaleSnapshot(t, err)
}

func TestOverviewWorkspaceWithOptionsWithoutExpectedFingerprintPerformsNoCheck(t *testing.T) {
	root := referenceWorkspaceRoot()
	if _, err := OverviewWorkspaceWithOptions(root, 0, OverviewOptions{}); err != nil {
		t.Fatalf("an omitted fingerprint was treated as a check: %v", err)
	}
	// The unqualified entry point keeps its behaviour for existing callers.
	if _, err := OverviewWorkspace(root, 0); err != nil {
		t.Fatalf("OverviewWorkspace changed behaviour: %v", err)
	}
}

// A token a caller can pass between two commands has to be the same value in
// both, otherwise the caller would have to know which read produced it. inspect,
// model locate and model overview report one snapshotFingerprint for one
// Workspace because they share InspectionSnapshotFingerprint.
//
// trace is deliberately outside that set: its token also binds the typed graph
// fingerprint, because a trace continuation must additionally prove that the
// projected relations did not change. Its value therefore differs from the other
// three by design and is not interchangeable with them.
func TestSnapshotFingerprintIsOneValueAcrossTheReadCommands(t *testing.T) {
	root := referenceWorkspaceRoot()
	inspection, err := InspectSubject(root, referenceLocateSubject())
	if err != nil {
		t.Fatal(err)
	}
	location, err := LocateWorkspace(root, referenceLocateSubject(), 0)
	if err != nil {
		t.Fatal(err)
	}
	overview, err := OverviewWorkspace(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := TraceWorkspace(root, referenceTraceKey(), modelref.TraceOptions{Mode: "summary"})
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]string{
		"model locate":   location.SnapshotFingerprint,
		"model overview": overview.SnapshotFingerprint,
	} {
		if got != inspection.SnapshotFingerprint {
			t.Fatalf("%s snapshot fingerprint = %s, inspect reports %s",
				name, got, inspection.SnapshotFingerprint)
		}
	}
	// Each command binds the same two halves, so the tokens can only agree if
	// the halves themselves agree; a shared composer over different halves would
	// still produce different values.
	if location.WorldFingerprint != inspection.WorldFingerprint ||
		location.SourceIndexFingerprint != inspection.SourceIndexFingerprint ||
		overview.WorldFingerprint != inspection.WorldFingerprint ||
		overview.SourceIndexFingerprint != inspection.SourceIndexFingerprint {
		t.Fatal("the three reads disagree on the World or the Source Index fingerprint")
	}
	if trace.SnapshotFingerprint == inspection.SnapshotFingerprint {
		t.Fatal("trace reports the inspect snapshot fingerprint; its continuation token is expected to bind the graph as well")
	}
	if trace.WorldFingerprint != inspection.WorldFingerprint ||
		trace.SourceIndexFingerprint != inspection.SourceIndexFingerprint {
		t.Fatal("trace and inspect disagree on the World or the Source Index fingerprint")
	}
}

// The precondition is only useful if the token moves when the Workspace moves.
// The Workspace is copied out of the reference tree, because reference/workspace
// itself is never modified.
func TestModelSnapshotFingerprintFollowsSourceEdits(t *testing.T) {
	root := copyReferenceWorkspace(t)
	subject := referenceLocateSubject()
	before, err := LocateWorkspace(root, subject, 0)
	if err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(root,
		filepath.FromSlash(referenceSourceArtifact(t, root, subject)))
	content, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	// A comment changes the Source bytes without changing the Canonical meaning,
	// so the two halves must be judged separately and the snapshot must move.
	if err := os.WriteFile(edited, append(content, []byte("\n// model snapshot precondition\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := LocateWorkspace(root, subject, 0)
	if err != nil {
		t.Fatal(err)
	}
	if after.WorldFingerprint != before.WorldFingerprint {
		t.Fatalf("a comment changed the World fingerprint: %s vs %s",
			before.WorldFingerprint, after.WorldFingerprint)
	}
	if after.SourceIndexFingerprint == before.SourceIndexFingerprint {
		t.Fatal("a Source edit left the Source Index fingerprint unchanged")
	}
	if after.SnapshotFingerprint == before.SnapshotFingerprint {
		t.Fatal("a Source edit left the snapshot fingerprint unchanged")
	}
	// The earlier token is now stale for both commands, and the current one is
	// not, so the precondition is what makes the change observable to a caller.
	if _, err := LocateWorkspaceWithOptions(root, subject, 0,
		LocateOptions{ExpectedFingerprint: before.SnapshotFingerprint}); err == nil {
		t.Fatal("an edited Workspace still matched the earlier location snapshot")
	} else {
		assertStaleSnapshot(t, err)
	}
	if _, err := OverviewWorkspaceWithOptions(root, 0,
		OverviewOptions{ExpectedFingerprint: before.SnapshotFingerprint}); err == nil {
		t.Fatal("an edited Workspace still matched the earlier overview snapshot")
	} else {
		assertStaleSnapshot(t, err)
	}
	if _, err := LocateWorkspaceWithOptions(root, subject, 0,
		LocateOptions{ExpectedFingerprint: after.SnapshotFingerprint}); err != nil {
		t.Fatalf("the current location snapshot fingerprint was refused: %v", err)
	}
	overviewAfter, err := OverviewWorkspace(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OverviewWorkspaceWithOptions(root, 0,
		OverviewOptions{ExpectedFingerprint: overviewAfter.SnapshotFingerprint}); err != nil {
		t.Fatalf("the current overview snapshot fingerprint was refused: %v", err)
	}
}

// referenceSourceArtifact returns the Source path an edit to this subject
// changes, read through inspect so the test edits the file the Source Index
// actually points at rather than one this test guesses.
func referenceSourceArtifact(t *testing.T, root string, subject sourceindex.Subject) string {
	t.Helper()
	inspection, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Sources.References) == 0 {
		t.Fatalf("%s has no Source reference to edit", subject)
	}
	return inspection.Sources.References[0].ArtifactPath
}

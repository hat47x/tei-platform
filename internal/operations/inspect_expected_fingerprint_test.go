package operations

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestInspectSubjectWithOptionsAcceptsMatchingSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	first, err := InspectSubjectWithOptions(root, subject, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.SnapshotFingerprint == "" {
		t.Fatalf("inspection carries no snapshot fingerprint: %+v", first)
	}
	second, err := InspectSubjectWithOptions(root, subject,
		InspectOptions{ExpectedFingerprint: first.SnapshotFingerprint})
	if err != nil {
		t.Fatalf("unchanged Workspace rejected its own snapshot fingerprint: %v", err)
	}
	if second.SnapshotFingerprint != first.SnapshotFingerprint {
		t.Fatalf("snapshot fingerprint is not stable across reads: %s vs %s",
			first.SnapshotFingerprint, second.SnapshotFingerprint)
	}
	// The token is the composition of the two halves it binds, not a third
	// observation, so a caller can derive it from either read.
	if want := InspectionSnapshotFingerprint(
		first.WorldFingerprint, first.SourceIndexFingerprint,
	); first.SnapshotFingerprint != want {
		t.Fatalf("snapshot fingerprint is not the composition of the two halves: %s want %s",
			first.SnapshotFingerprint, want)
	}
}

func TestInspectSubjectWithOptionsRejectsStaleSnapshotFingerprint(t *testing.T) {
	root := referenceWorkspaceRoot()
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	first, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	_, err = InspectSubjectWithOptions(root, subject,
		InspectOptions{ExpectedFingerprint: "sha256:not-this-workspace"})
	var inspection *InspectionError
	if !errors.As(err, &inspection) || inspection.Code != InspectionStaleSnapshot {
		t.Fatalf("stale snapshot fingerprint was not refused: %v", err)
	}
	if inspection.Code != "STALE_SNAPSHOT" {
		t.Fatalf("stale snapshot code = %q", inspection.Code)
	}
	// The answer must name what changed, so a caller does not read a mismatch as
	// "the subject moved" or "the answer was rejected as invalid".
	if !strings.Contains(inspection.Message, "World") ||
		!strings.Contains(inspection.Message, "Source Index") {
		t.Fatalf("stale snapshot message does not name what changed: %q", inspection.Message)
	}
	// A token from another subject of the same Workspace is not this read's
	// snapshot either: the precondition covers the Workspace, not the subject.
	other, err := InspectSubject(root, sourceindex.Subject{
		Kind: "semantic-data", ID: "sales.order_line",
	})
	if err != nil {
		t.Fatal(err)
	}
	if other.SnapshotFingerprint != first.SnapshotFingerprint {
		_, err = InspectSubjectWithOptions(root, subject,
			InspectOptions{ExpectedFingerprint: other.SnapshotFingerprint})
		if !errors.As(err, &inspection) || inspection.Code != InspectionStaleSnapshot {
			t.Fatalf("another subject's snapshot fingerprint was accepted: %v", err)
		}
	}
}

func TestInspectSubjectWithOptionsWithoutExpectedFingerprintPerformsNoCheck(t *testing.T) {
	root := referenceWorkspaceRoot()
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	// Only an absent expectation is "no check". A value that is not the empty
	// string is a claim this read must reproduce, exactly as trace judges a
	// continuation token, so a whitespace-only value is refused rather than
	// silently ignored.
	if _, err := InspectSubjectWithOptions(root, subject, InspectOptions{}); err != nil {
		t.Fatalf("an omitted fingerprint was treated as a check: %v", err)
	}
	// The unqualified entry point keeps its behaviour for existing callers.
	if _, err := InspectSubject(root, subject); err != nil {
		t.Fatalf("InspectSubject changed behaviour: %v", err)
	}
}

// A fingerprint that does not move when a Source file changes would make the
// precondition useless: the caller could not tell an edited Workspace from the
// one it read. The Workspace is copied out of the reference tree, because
// reference/workspace itself is never modified.
func TestInspectSnapshotFingerprintFollowsSourceEdits(t *testing.T) {
	root := copyReferenceWorkspace(t)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	before, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	if before.SnapshotFingerprint == "" || before.WorldFingerprint == "" ||
		before.SourceIndexFingerprint == "" {
		t.Fatalf("inspection is missing fingerprints: %+v", before)
	}
	edited := filepath.Join(root, filepath.FromSlash(before.Sources.References[0].ArtifactPath))
	content, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	// A comment changes the Source bytes without changing the Canonical meaning,
	// so the two halves must be judged separately and the snapshot must move.
	if err := os.WriteFile(edited, append(content, []byte("\n// inspect snapshot precondition\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := InspectSubject(root, subject)
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
	// The precondition is what makes that observable to a caller: the earlier
	// token is now stale and the current one is not.
	_, err = InspectSubjectWithOptions(root, subject,
		InspectOptions{ExpectedFingerprint: before.SnapshotFingerprint})
	var inspection *InspectionError
	if !errors.As(err, &inspection) || inspection.Code != InspectionStaleSnapshot {
		t.Fatalf("an edited Workspace still matched the earlier snapshot: %v", err)
	}
	if _, err := InspectSubjectWithOptions(root, subject,
		InspectOptions{ExpectedFingerprint: after.SnapshotFingerprint}); err != nil {
		t.Fatalf("the current snapshot fingerprint was refused: %v", err)
	}
}

func copyReferenceWorkspace(t *testing.T) string {
	t.Helper()
	source := referenceWorkspaceRoot()
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

package studiohost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// The expected snapshot compares World and Source Index as one observation.
// It is a read-only freshness check, never an acceptance or write grant.
func TestInspectSubjectAtSnapshotRejectsMissingAndStaleEvidence(t *testing.T) {
	h, err := NewReadOnly([]string{filepath.Join("..", "..", "reference", "workspace")})
	if err != nil {
		t.Fatal(err)
	}
	catalog := h.CatalogWorkspace("workspace-1", "")
	if !catalog.OK || catalog.Value == nil || len(catalog.Value.Entries) == 0 {
		t.Fatalf("reference fixture lacks an inspectable subject: %+v", catalog)
	}
	subject := catalog.Value.Entries[0].Subject
	before := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if !before.OK || before.Value == nil || before.Value.SnapshotFingerprint == "" {
		t.Fatalf("missing current inspection fingerprint: %+v", before)
	}
	unchanged := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind,
		subject.ID, before.Value.SnapshotFingerprint)
	if !unchanged.OK || unchanged.Value == nil ||
		unchanged.Value.SnapshotFingerprint != before.Value.SnapshotFingerprint ||
		unchanged.Value.Effect != "read-only" {
		t.Fatalf("current snapshot was not read-only and accepted: %+v", unchanged)
	}
	missing := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind, subject.ID, "")
	if missing.OK || missing.Failure == nil ||
		missing.Failure.Code != "SNAPSHOT_REQUIRED" ||
		missing.Failure.EffectUnknown {
		t.Fatalf("freshness precondition silently optional: %+v", missing)
	}
	staleToken := "sha256:" + strings.Repeat("0", 64)
	if staleToken == before.Value.SnapshotFingerprint {
		staleToken = "sha256:" + strings.Repeat("f", 64)
	}
	stale := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind, subject.ID, staleToken)
	if stale.OK || stale.Failure == nil ||
		stale.Failure.Code != "STALE_SNAPSHOT" ||
		stale.Failure.Kind != "business_rejection" ||
		stale.Failure.Retryable != "not_retryable" || stale.Failure.EffectUnknown {
		t.Fatalf("stale Source/World snapshot not rejected as a zero-effect read: %+v", stale)
	}
	unlisted := h.InspectSubjectAtSnapshot("workspace-2", subject.Kind, subject.ID, "")
	if unlisted.OK || unlisted.Failure == nil ||
		unlisted.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("allowlist was bypassed by snapshot check: %+v", unlisted)
	}
}

type snapshotInspectionService struct {
	result operations.SubjectInspection
	calls  *int
}

func (s snapshotInspectionService) Inspect(_ sourceindex.Subject) (operations.SubjectInspection, error) {
	(*s.calls)++
	return s.result, nil
}

func TestSelectedInspectorAlsoEnforcesExpectedSnapshot(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	baseline, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	catalog := baseline.CatalogWorkspace("workspace-1", "")
	if !catalog.OK || catalog.Value == nil || len(catalog.Value.Entries) == 0 {
		t.Fatalf("missing fixture subjects: %+v", catalog)
	}
	subject := catalog.Value.Entries[0].Subject
	observed := baseline.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if !observed.OK || observed.Value == nil || observed.Value.SnapshotFingerprint == "" {
		t.Fatalf("missing fixture inspection: %+v", observed)
	}
	calls := 0
	selected, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
		return RootServices{Inspection: snapshotInspectionService{
			result: *observed.Value, calls: &calls,
		}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	match := selected.InspectSubjectAtSnapshot(
		"workspace-1", subject.Kind, subject.ID, observed.Value.SnapshotFingerprint)
	if !match.OK || match.Value == nil || calls != 1 {
		t.Fatalf("selected Inspector rejected matching snapshot: %+v, calls=%d", match, calls)
	}
	stale := selected.InspectSubjectAtSnapshot(
		"workspace-1", subject.Kind, subject.ID, "sha256:"+strings.Repeat("f", 64))
	if stale.Failure == nil || stale.Failure.Code != "STALE_SNAPSHOT" ||
		stale.Failure.Kind != "business_rejection" || stale.Failure.EffectUnknown {
		t.Fatalf("selected Provider did not reject stale snapshot safely: %+v", stale)
	}
	if calls != 2 {
		t.Fatalf("selected Inspector was bypassed: %d calls", calls)
	}
}

// A real file relocation changes the Source Index even if its bytes, semantic
// subject and World fingerprint do not change. Conditional reinspection must
// reject the old observation and accept the newly read one without a write.
func TestInspectSubjectAtSnapshotRejectsRealSameByteMove(t *testing.T) {
	root := t.TempDir()
	semanticDir := filepath.Join(root, "semantic")
	if err := os.Mkdir(semanticDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: source_move_fixture\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceBytes := []byte("namespace studio\n\ntype Marker from std.text\n")
	oldPath := filepath.Join(semanticDir, "marker.tei")
	newPath := filepath.Join(semanticDir, "marker-relocated.tei")
	if err := os.WriteFile(oldPath, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	catalog := h.CatalogWorkspace("workspace-1", "")
	if !catalog.OK || catalog.Value == nil {
		t.Fatalf("initial Catalog: %+v", catalog)
	}
	var subject sourceindex.Subject
	for _, entry := range catalog.Value.Entries {
		if entry.Subject.Kind == "semantic-type" && !entry.Prelude {
			if subject.ID != "" {
				t.Fatal("fixture has more than one non-prelude semantic type")
			}
			subject = entry.Subject
		}
	}
	if subject.ID == "" {
		t.Fatal("fixture semantic type not found")
	}
	before := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if !before.OK || before.Value == nil || before.Value.SnapshotFingerprint == "" ||
		before.Value.SourceStatus != "indexed" || before.Value.Effect != "read-only" {
		t.Fatalf("initial Inspect: %+v", before)
	}
	matching := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind,
		subject.ID, before.Value.SnapshotFingerprint)
	if !matching.OK || matching.Value == nil ||
		matching.Value.SnapshotFingerprint != before.Value.SnapshotFingerprint {
		t.Fatalf("unchanged Source failed conditional Inspect: %+v", matching)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	reloaded, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(reloaded) != string(sourceBytes) {
		t.Fatal("fixture changed Source bytes rather than its location")
	}
	stale := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind,
		subject.ID, before.Value.SnapshotFingerprint)
	if stale.OK || stale.Failure == nil ||
		stale.Failure.Code != "STALE_SNAPSHOT" ||
		stale.Failure.Kind != "business_rejection" ||
		stale.Failure.Retryable != "not_retryable" || stale.Failure.EffectUnknown {
		t.Fatalf("old snapshot was not rejected after real Source move: %+v", stale)
	}
	current := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if !current.OK || current.Value == nil ||
		current.Value.SourceStatus != "indexed" || current.Value.Effect != "read-only" ||
		len(current.Value.Sources.References) != 1 {
		t.Fatalf("current Inspect failed after Source move: %+v", current)
	}
	if string(before.Value.Canonical) != string(current.Value.Canonical) ||
		before.Value.WorldFingerprint != current.Value.WorldFingerprint {
		t.Fatal("physical relocation changed semantic value or World fingerprint")
	}
	if before.Value.SourceIndexFingerprint == current.Value.SourceIndexFingerprint ||
		before.Value.SnapshotFingerprint == current.Value.SnapshotFingerprint ||
		before.Value.Sources.References[0].ArtifactPath == current.Value.Sources.References[0].ArtifactPath {
		t.Fatal("Source relocation did not invalidate the old path and snapshot")
	}
	matchingCurrent := h.InspectSubjectAtSnapshot("workspace-1", subject.Kind,
		subject.ID, current.Value.SnapshotFingerprint)
	if !matchingCurrent.OK || matchingCurrent.Value == nil ||
		matchingCurrent.Value.SnapshotFingerprint != current.Value.SnapshotFingerprint ||
		matchingCurrent.Value.Effect != "read-only" {
		t.Fatalf("current Source snapshot was not accepted for read-only Inspect: %+v", matchingCurrent)
	}
}

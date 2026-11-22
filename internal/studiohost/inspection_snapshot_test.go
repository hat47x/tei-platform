package studiohost

import (
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

package studiohost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func rootReplacementPair(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	root := filepath.Join(tmp, "workspace-a")
	replacement := filepath.Join(tmp, "workspace-b")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(replacement, 0o700); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	replacementInfo, err := os.Stat(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(originalInfo, replacementInfo) {
		t.Fatal("replacement fixture must have a distinct directory identity")
	}
	return root, replacement
}

func replaceWorkspaceRoot(t *testing.T, root, replacement string) {
	t.Helper()
	moved := root + "-original"
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, root); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(moved)
	if err != nil {
		t.Fatal(err)
	}
	replacementInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(originalInfo, replacementInfo) {
		t.Fatal("replacement path still identifies the original directory")
	}
}

func TestReadOnlyRejectsRootReplacement(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	h, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)

	if got := h.ValidateWorkspace("workspace-1"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("validation did not reject replacement root: %+v", got)
	}
	if got := h.CatalogWorkspace("workspace-1", ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("catalog did not reject replacement root: %+v", got)
	}
}

func TestFactoryIsNotCalledForReplacedWorkspaceRoot(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	factoryCalls := 0
	h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
		factoryCalls++
		return RootServices{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)

	if got := h.ValidateWorkspace("workspace-1"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("factory validation did not reject replacement root: %+v", got)
	}
	if got := h.CatalogWorkspace("workspace-1", ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("factory catalog did not reject replacement root: %+v", got)
	}
	if factoryCalls != 0 {
		t.Fatalf("factory was called for replaced root %d times", factoryCalls)
	}
}

func TestReadOnlyRejectsUnauthorizedIDs(t *testing.T) {
	h, err := NewReadOnly([]string{filepath.Join("..", "..", "reference", "workspace")})
	if err != nil {
		t.Fatal(err)
	}
	list := h.ListWorkspaces()
	if !list.OK || len(list.Value) != 1 || list.Value[0].ID != "workspace-1" {
		t.Fatalf("unexpected allowlist: %+v", list)
	}
	list.Value[0].ID = "changed"
	if h.ListWorkspaces().Value[0].ID != "workspace-1" {
		t.Fatal("caller mutated the host's workspace list")
	}
	for _, id := range []string{"", "../reference/workspace", "workspace-2", " workspace-1"} {
		if got := h.ValidateWorkspace(id); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("validate %q: %+v", id, got)
		}
		if got := h.CatalogWorkspace(id, ""); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("catalog %q: %+v", id, got)
		}
		if got := h.InspectSubject(id, "semantic", "x"); got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("inspect %q: %+v", id, got)
		}
	}
}

func TestRealOperationReadOnlyReference(t *testing.T) {
	h, err := NewReadOnly([]string{filepath.Join("..", "..", "reference", "workspace")})
	if err != nil {
		t.Fatal(err)
	}
	catalog := h.CatalogWorkspace("workspace-1", "")
	if !catalog.OK || catalog.Value == nil || catalog.Value.Effect != "read-only" {
		t.Fatalf("catalog: %+v", catalog)
	}
	validation := h.ValidateWorkspace("workspace-1")
	if !validation.OK || validation.Value == nil {
		t.Fatalf("validation: %+v", validation)
	}
	if len(catalog.Value.Entries) > 0 {
		subject := catalog.Value.Entries[0].Subject
		inspection := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
		if !inspection.OK || inspection.Value == nil || inspection.Value.Effect != "read-only" {
			t.Fatalf("inspect %v: %+v", subject, inspection)
		}
	}
}

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
		missing.Failure.Code != "INSPECTION_SNAPSHOT_REQUIRED" ||
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

// Moving a Source without changing its bytes must invalidate a prior
// inspection snapshot; the World may remain semantically identical.
func TestInspectSubjectAtSnapshotRejectsSameByteSourceMove(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "semantic")
	if err := os.Mkdir(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"),
		[]byte("workspace: source_move_snapshot\nformatVersion: 1\nsources:\n  - semantic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(sourceDir, "marker.tei")
	moved := filepath.Join(sourceDir, "relocated.tei")
	if err := os.WriteFile(original, []byte("namespace alpha\n\ntype Marker from std.text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	before := h.InspectSubject("workspace-1", "semantic", "alpha")
	if !before.OK || before.Value == nil || before.Value.SnapshotFingerprint == "" {
		t.Fatalf("failed to observe original Source: %+v", before)
	}
	if err := os.Rename(original, moved); err != nil {
		t.Fatal(err)
	}
	stale := h.InspectSubjectAtSnapshot("workspace-1", "semantic",
		"alpha", before.Value.SnapshotFingerprint)
	if stale.OK || stale.Failure == nil || stale.Failure.Code != "STALE_SNAPSHOT" ||
		stale.Failure.EffectUnknown {
		t.Fatalf("Source relocation reused a stale snapshot: %+v", stale)
	}
	current := h.InspectSubject("workspace-1", "semantic", "alpha")
	if !current.OK || current.Value == nil ||
		current.Value.SourceIndexFingerprint == before.Value.SourceIndexFingerprint {
		t.Fatalf("Source relocation was not reflected in the Index: %+v", current)
	}
	matched := h.InspectSubjectAtSnapshot("workspace-1", "semantic",
		"alpha", current.Value.SnapshotFingerprint)
	if !matched.OK || matched.Value == nil || matched.Value.Effect != "read-only" {
		t.Fatalf("current relocated Source was not inspectable: %+v", matched)
	}
}

func TestRootConfigurationRefusesDuplicatesAndMissing(t *testing.T) {
	path := filepath.Join("..", "..", "reference", "workspace")
	if _, err := NewReadOnly(nil); err == nil {
		t.Fatal("empty root list accepted")
	}
	if _, err := NewReadOnly([]string{path, path}); err == nil {
		t.Fatal("duplicate root accepted")
	}
	if _, err := NewReadOnly([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("missing root accepted")
	}
}

func TestLoadFailureIsDistinguishedFromValidationDiagnostics(t *testing.T) {
	h, err := NewReadOnly([]string{t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	got := h.ValidateWorkspace("workspace-1")
	if got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_LOAD_FAILED" || got.Failure.Stage != "load" {
		t.Fatalf("unexpected failure: %+v", got)
	}
}

// TestReadOnlyPinsTheResolvedRootNotTheConfiguredAlias checks that a Host
// configured through one path cannot be retargeted by replacing that path.
// The existing root-replacement test renames the resolved root itself; this
// one changes only the configured alias, as the Studio fixed-Host negative
// attempts (#3061).
func TestReadOnlyPinsTheResolvedRootNotTheConfiguredAlias(t *testing.T) {
	parent := t.TempDir()
	allowed := filepath.Join(parent, "allowed")
	outside := filepath.Join(parent, "outside")
	alias := filepath.Join(parent, "configured-alias")
	for _, fixture := range []struct{ dir, namespace string }{{allowed, "allowed"}, {outside, "secret"}} {
		if err := os.Mkdir(fixture.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(fixture.dir, "semantic"), 0o700); err != nil {
			t.Fatal(err)
		}
		manifest := "workspace: root_identity_fixture\nformatVersion: 1\nsources:\n  - semantic\n"
		if err := os.WriteFile(filepath.Join(fixture.dir, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		source := "namespace " + fixture.namespace + "\n\ntype Marker from std.text\n"
		if err := os.WriteFile(filepath.Join(fixture.dir, "semantic", "marker.tei"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(allowed, alias); err != nil {
		t.Skipf("creating symlinks is unavailable on this host: %v", err)
	}
	h, err := NewReadOnly([]string{alias})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	// The allowlist pinned the resolved root at startup. Replacing the alias
	// must not change which directory workspace-1 names.
	got := h.CatalogWorkspace("workspace-1", "")
	if !got.OK || got.Value == nil {
		t.Fatalf("pinned root became unavailable after alias replacement: %+v", got.Failure)
	}
	for _, entry := range got.Value.Entries {
		if entry.Subject.Kind == "semantic" && entry.Subject.ID == "secret" {
			t.Fatal("allowlisted Workspace ID followed the replaced alias into the outside Workspace")
		}
	}
}

func TestFailureClassifyCompletesMissingRetryWithoutDiscardingExplicitContract(t *testing.T) {
	tests := []struct {
		name        string
		input       Failure
		wantKind    string
		wantRetry   string
		wantUnknown bool
	}{
		{
			name:     "code only technical",
			input:    Failure{Code: "COMMAND_FAILED", Message: "failed"},
			wantKind: "technical_failure", wantRetry: "lookup_first", wantUnknown: true,
		},
		{
			name:     "explicit technical kind with missing retry",
			input:    Failure{Code: "COMMAND_FAILED", Message: "failed", Kind: "technical_failure"},
			wantKind: "technical_failure", wantRetry: "lookup_first", wantUnknown: true,
		},
		{
			name:     "explicit retry policy with missing kind",
			input:    Failure{Code: "COMMAND_FAILED", Message: "failed", Retryable: "lookup_first"},
			wantKind: "technical_failure", wantRetry: "lookup_first", wantUnknown: true,
		},
		{
			name:     "explicit business kind with missing retry",
			input:    Failure{Code: "WORKSPACE_NOT_ALLOWED", Message: "denied", Kind: "business_rejection"},
			wantKind: "business_rejection", wantRetry: "not_retryable", wantUnknown: false,
		},
		{
			name:     "disagreeing explicit kind fails closed",
			input:    Failure{Code: "WORKSPACE_NOT_ALLOWED", Message: "failed", Kind: "technical_failure"},
			wantKind: "technical_failure", wantRetry: "lookup_first", wantUnknown: true,
		},
		{
			name:     "unknown code and explicit business kind fail closed",
			input:    Failure{Code: "FUTURE_CODE", Message: "failed", Kind: "business_rejection"},
			wantKind: "business_rejection", wantRetry: "lookup_first", wantUnknown: true,
		},
		{
			name: "all explicit axes win",
			input: Failure{Code: "COMMAND_FAILED", Message: "refused by provider",
				Kind: "business_rejection", Retryable: "not_retryable", EffectUnknown: false},
			wantKind: "business_rejection", wantRetry: "not_retryable", wantUnknown: false,
		},
		{
			name:     "explicit effect uncertainty is never cleared",
			input:    Failure{Code: "WORKSPACE_NOT_ALLOWED", Message: "unknown", EffectUnknown: true},
			wantKind: "business_rejection", wantRetry: "not_retryable", wantUnknown: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inProcess := tt.input
			got := inProcess.classify()
			if got.Kind != tt.wantKind || got.Retryable != tt.wantRetry || got.EffectUnknown != tt.wantUnknown {
				t.Fatalf("in-process failure: got %+v; want kind=%q retry=%q effectUnknown=%t", got, tt.wantKind, tt.wantRetry, tt.wantUnknown)
			}
			wireInput := tt.input
			b, err := json.Marshal(&wireInput)
			if err != nil {
				t.Fatal(err)
			}
			var wire Failure
			if err := json.Unmarshal(b, &wire); err != nil {
				t.Fatal(err)
			}
			if wire.Kind != tt.wantKind || wire.Retryable != tt.wantRetry || wire.EffectUnknown != tt.wantUnknown {
				t.Fatalf("wire failure: %s; want kind=%q retry=%q effectUnknown=%t", b, tt.wantKind, tt.wantRetry, tt.wantUnknown)
			}
		})
	}
}

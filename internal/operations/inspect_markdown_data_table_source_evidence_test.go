package operations

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a product Inspector check for a marked Markdown Data item table.
// A selected Reader yields the existing Data/Attribute meaning; an unmarked
// example is not another declaration, and Source evidence grants no edit or
// execution authority. No new Reader, Inspector API, or synthetic Provider is
// involved.
func TestSelectedMarkdownDataTableInspectorPreservesSourceAndRole(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	const sourcePath = "semantic/organization/transfer-request.tei.md"

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("product Load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 {
		t.Fatalf("selected authoring sources=%+v", loaded.AuthoringSources)
	}
	reader := loaded.AuthoringSources[0]
	if reader.Path != sourcePath ||
		reader.Language.LanguageID != "tei.authoring" ||
		reader.Language.ContractVersion != "v1" ||
		reader.Language.ImplementationID != "tei.builtin.authoring-markdown" ||
		reader.Regions != 2 || reader.Definitions != 2 {
		t.Fatalf("wrong Reader or coverage: %+v", reader)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("same-load Source Index: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	// The table's Data, its cell-defined Attribute and the adjacent fenced
	// Rule have different kinds but share this one pinned Source byte stream.
	cases := []struct {
		kind  string
		id    string
		token string
	}{
		{"semantic-data", "organization.transfer_request", "TransferRequest"},
		{"semantic-attribute", "organization.transfer_request.effective_to", "effectiveTo"},
		{"semantic-validation", "organization.transfer_request.allocation_positive", "organization.transfer_request.allocation_positive"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			subject := sourceindex.Subject{Kind: tc.kind, ID: tc.id}
			got, err := InspectionFromLoad(loaded, index, subject)
			if err != nil {
				t.Fatalf("product Inspector: %v", err)
			}
			if err := ValidateInspectionProjection(subject, got); err != nil {
				t.Fatalf("invalid public Inspector projection: %v", err)
			}
			if got.Effect != "read-only" || got.SourceStatus != "indexed" ||
				!reflect.DeepEqual(got.AuthorityLimits, InspectionAuthorityLimits()) {
				t.Fatalf("inspection lost its limited authority: %+v", got)
			}
			var canonical struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(got.Canonical, &canonical); err != nil || canonical.ID != tc.id {
				t.Fatalf("wrong typed Subject: %q, err=%v", got.Canonical, err)
			}
			if len(got.Sources.References) != 1 || len(got.Sources.Artifacts) != 1 {
				t.Fatalf("unexpected source multiplicity: %+v", got.Sources)
			}
			ref := got.Sources.References[0]
			artifact := got.Sources.Artifacts[0]
			if ref.Subject != subject || ref.Role != sourceindex.RoleDefinition ||
				ref.ArtifactPath != sourcePath || artifact.Path != sourcePath ||
				artifact.Fingerprint != sourceindex.Fingerprint(raw) {
				t.Fatalf("wrong authoring provenance: ref=%+v artifact=%+v", ref, artifact)
			}
			if err := ref.Range.Validate(raw); err != nil {
				t.Fatalf("stale byte range: %v", err)
			}
			if ref.Range.Start.Offset != bytes.Index(raw, []byte(tc.token)) {
				t.Fatalf("wrong definition anchor for %s: %+v", tc.id, ref.Range)
			}
			traced, err := TraceSubjectSourcesFromIndex(index, subject)
			if err != nil || !reflect.DeepEqual(traced, got.Sources) {
				t.Fatalf("Trace and Inspector disagree: trace=%+v, inspect=%+v, err=%v", traced, got.Sources, err)
			}
		})
	}

	// The document's unmarked example row is neither a Data definition nor
	// a callable Capability; the Rule likewise creates no public Entry.
	for _, missing := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "organization.example_record"},
		{Kind: "semantic-capability", ID: "organization.transfer_request.allocation_positive"},
	} {
		_, err := InspectionFromLoad(loaded, index, missing)
		inspectionCode(t, err, InspectionNotFound)
	}
}

// A prose-only revision of a marked Markdown definition must invalidate the
// Source-bound Inspector/Model revision without silently changing its typed
// meaning. Restoring identical bytes only restores descriptive fingerprints,
// not an edit grant, execution authorization, or an earlier Consumer result.
func TestSelectedMarkdownDataTableProseRevisionPreservesMeaningNotSource(t *testing.T) {
	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	root := t.TempDir()
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	manifest, err := os.ReadFile(filepath.Join(fixture, "tei.workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	write := func(path string, value []byte) {
		t.Helper()
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, value, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", manifest)
	write(sourcePath, original)
	subject := sourceindex.Subject{Kind: "semantic-data", ID: "organization.transfer_request"}
	before, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("initial product read: %v", err)
	}
	write(sourcePath, append(append([]byte(nil), original...), []byte("\nこの段落は定義ではなく、説明の追記です。\n")...))
	after, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("read after prose revision: %v", err)
	}
	if before.ModelRevision == after.ModelRevision ||
		before.Inspection.SourceIndexFingerprint == after.Inspection.SourceIndexFingerprint {
		t.Fatal("prose-only revision retained stale Source qualification")
	}
	if before.Inspection.WorldFingerprint != after.Inspection.WorldFingerprint ||
		!bytes.Equal(before.Inspection.Canonical, after.Inspection.Canonical) {
		t.Fatal("prose-only revision unexpectedly changed typed Data meaning")
	}
	if !reflect.DeepEqual(before.Inspection.Sources.References, after.Inspection.Sources.References) {
		t.Fatal("a trailing prose edit unexpectedly moved Data definition anchors")
	}

	write(sourcePath, original)
	restored, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ModelRevision != before.ModelRevision ||
		restored.Inspection.SourceIndexFingerprint != before.Inspection.SourceIndexFingerprint ||
		!bytes.Equal(restored.Inspection.Canonical, before.Inspection.Canonical) {
		t.Fatal("A-B-A did not restore the descriptive Source and Meaning state")
	}
}

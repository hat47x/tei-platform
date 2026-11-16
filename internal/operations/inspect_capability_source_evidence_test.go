package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
)

// A real semantic Capability can be inspected and traced to the original
// identifier bytes. Neither the read-only inspection nor the trace proves
// that its public Entry was invoked or that any Provider was executed.
func TestInspectCapabilitySourceEvidenceFromWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	subject := sourceindex.Subject{Kind: "semantic-capability", ID: "sales.order.submit"}

	inspection, err := InspectSubject(root, subject)
	if err != nil {
		t.Fatalf("inspect capability: %v", err)
	}
	if inspection.Subject != subject || inspection.Effect != "read-only" {
		t.Fatalf("inspection subject/effect mismatch: %+v", inspection)
	}
	if inspection.SourceStatus != "indexed" || inspection.SourceIndexFingerprint == "" {
		t.Fatalf("source index is not qualified: %+v", inspection)
	}
	if !reflect.DeepEqual(inspection.AuthorityLimits, InspectionAuthorityLimits()) {
		t.Fatalf("source inspection lost read-only authority limits: %v", inspection.AuthorityLimits)
	}

	var canonical struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(inspection.Canonical, &canonical); err != nil {
		t.Fatalf("canonical capability: %v", err)
	}
	if canonical.ID != subject.ID {
		t.Fatalf("inspected a different canonical capability: %q", canonical.ID)
	}

	refs := inspection.Sources.References
	if len(refs) == 0 || len(inspection.Sources.Artifacts) == 0 {
		t.Fatalf("canonical capability has no source evidence: %+v", inspection.Sources)
	}
	artifactByPath := make(map[string]sourceindex.Artifact, len(inspection.Sources.Artifacts))
	for _, artifact := range inspection.Sources.Artifacts {
		artifactByPath[artifact.Path] = artifact
	}
	const sourcePath = "semantic/sales/order.tei"
	found := false
	for _, ref := range refs {
		if ref.Subject != subject || ref.Role != sourceindex.RoleDefinition {
			t.Fatalf("reference changed subject or role: %+v", ref)
		}
		artifact, ok := artifactByPath[ref.ArtifactPath]
		if !ok {
			t.Fatalf("reference has no matching source artifact: %+v", ref)
		}
		// The path is a workspace-relative source index value, not an editor
		// URL or an authorization to open a Source in a product session.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			t.Fatalf("read fixture source %s: %v", artifact.Path, err)
		}
		if got := sourceindex.Fingerprint(raw); artifact.Fingerprint != got {
			t.Fatalf("source bytes and indexed artifact differ: %s != %s", artifact.Fingerprint, got)
		}
		if err := ref.Range.Validate(raw); err != nil {
			t.Fatalf("stale source byte range for %s: %v", artifact.Path, err)
		}
		if ref.ArtifactPath == sourcePath {
			found = true
			anchor := string(raw[ref.Range.Start.Offset:ref.Range.End.Offset])
			if anchor != subject.ID {
				t.Fatalf("capability source anchor=%q, want %q", anchor, subject.ID)
			}
		}
	}
	if !found {
		t.Fatalf("expected source %q not found in references: %+v", sourcePath, refs)
	}

	traced, err := TraceSubjectSources(root, subject)
	if err != nil {
		t.Fatalf("trace capability source: %v", err)
	}
	if !reflect.DeepEqual(traced.References, refs) ||
		!reflect.DeepEqual(traced.Artifacts, inspection.Sources.Artifacts) {
		t.Fatalf("trace and inspection disagree on the same Source: trace=%+v inspect=%+v",
			traced, inspection.Sources)
	}

	// A source lookup alone may return an empty set. Inspect must instead
	// reject a missing canonical Capability; it must not promote the trace
	// result into a model declaration.
	missing := sourceindex.Subject{Kind: "semantic-capability", ID: "sales.not_declared"}
	empty, err := TraceSubjectSources(root, missing)
	if err != nil || len(empty.References) != 0 {
		t.Fatalf("missing source trace=%+v, error=%v", empty, err)
	}
	_, err = InspectSubject(root, missing)
	inspectionCode(t, err, InspectionNotFound)
}

// A stable Canonical Capability is not a Source revision token. This fixture
// uses the same product Workspace reader as document projections; a trailing
// newline must invalidate Source-bound views without inventing new business
// meaning or claiming that a runtime call took place.
func TestCapabilitySourceOnlyEditInvalidatesPinnedRevision(t *testing.T) {
	root := documentDraftFixture(t)
	file := filepath.Join(root, "semantic", "sales", "order.tei")
	base, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	// The base Workspace declares sales.order; this adds a real Capability
	// referring to that Data, with the same syntax as reference/workspace.
	capability := []byte("\ncapability sales.order.submit {\n    input order sales.order\n    output sales.order\n}\n")
	original := append(append([]byte(nil), base...), capability...)
	if err := os.WriteFile(file, original, 0o644); err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "semantic-capability", ID: "sales.order.submit"}
	before, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("read Capability before Source-only edit: %v", err)
	}
	if before.Inspection.Subject != subject || before.Inspection.SourceStatus != "indexed" ||
		before.ModelRevision == "" || before.Inspection.SourceIndexFingerprint == "" ||
		len(before.Inspection.Sources.References) == 0 {
		t.Fatalf("missing pinned Capability evidence: %+v", before)
	}

	// This change is semantically neutral, but it produces different managed
	// Source bytes. An existing Source-bound view cannot silently stay current.
	edited := append(append([]byte(nil), original...), '\n')
	if err := os.WriteFile(file, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("read Capability after Source-only edit: %v", err)
	}
	if before.ModelRevision == after.ModelRevision ||
		before.Inspection.SourceIndexFingerprint == after.Inspection.SourceIndexFingerprint {
		t.Fatal("Source-only change did not invalidate Source revision and Index fingerprint")
	}
	if before.Inspection.WorldFingerprint != after.Inspection.WorldFingerprint ||
		string(before.Inspection.Canonical) != string(after.Inspection.Canonical) ||
		before.Inspection.Subject != after.Inspection.Subject {
		t.Fatal("Source-only change unexpectedly changed the Canonical Capability")
	}
	oldRefs, newRefs := before.Inspection.Sources.References, after.Inspection.Sources.References
	if !reflect.DeepEqual(oldRefs, newRefs) {
		t.Fatalf("trailing newline moved a Capability identifier: before=%+v after=%+v", oldRefs, newRefs)
	}
	if len(before.Inspection.Sources.Artifacts) != 1 || len(after.Inspection.Sources.Artifacts) != 1 ||
		before.Inspection.Sources.Artifacts[0].Fingerprint == after.Inspection.Sources.Artifacts[0].Fingerprint {
		t.Fatal("edited Source bytes did not change the indexed artifact fingerprint")
	}

	// Returning to the old bytes can restore the descriptive fingerprint;
	// it does not prove that there was no intermediate revision (A->B->A) or
	// restore any edit, disclosure or execution grant.
	if err := os.WriteFile(file, original, 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ModelRevision != before.ModelRevision ||
		restored.Inspection.SourceIndexFingerprint != before.Inspection.SourceIndexFingerprint ||
		string(restored.Inspection.Canonical) != string(before.Inspection.Canonical) {
		t.Fatal("A-B-A bytes failed to recover the descriptive Source/Meaning state")
	}
}

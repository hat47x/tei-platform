package exampleslide

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func modelHypothesisAuditFixture(t *testing.T) (string, Input, ModelHypotheses) {
	t.Helper()
	root, input, _ := sourceAuditWorkspaceFixture(t)
	attribute := input.Catalog.Tables["sales.child"].Fields[0].ID
	proposals := ModelHypotheses{
		Schema: modelHypothesesSchema,
		CandidateKeys: []CandidateKeyHypothesis{{
			ID: "business-parent", Table: "sales.child", Fields: []string{attribute},
		}},
		FunctionalDependencies: []FDHypothesis{{
			ID: "parent-depends-on-itself", Table: "sales.child",
			Determinants: []string{attribute}, Dependents: []string{attribute},
		}},
	}
	return root, input, proposals
}

func TestModelHypothesisSourceAuditChecksExactDataAndAttributeAnchors(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	review, err := ReviewModelHypotheses(input, proposals)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals)
	if err != nil {
		t.Fatal(err)
	}
	if audit.CaseID != review.CaseID ||
		audit.ModelReviewFingerprint != review.Fingerprint ||
		audit.HypothesesFingerprint != review.HypothesesFingerprint ||
		!strings.HasPrefix(audit.Fingerprint, "sha256:") ||
		len(audit.Findings) != 4 {
		t.Fatalf("wrong provenance binding: %+v", audit)
	}
	for _, finding := range audit.Findings {
		if finding.Status != "source-range-checked" ||
			finding.ArtifactPath != "semantic/model.tei" ||
			!strings.HasPrefix(finding.ArtifactFingerprint, "sha256:") ||
			finding.SourceRange == nil || finding.SourceRange.Start.Offset >= finding.SourceRange.End.Offset ||
			finding.IssueID == "" {
			t.Fatalf("missing exact source range or identity: %+v", finding)
		}
		if finding.SubjectKind != "semantic-data" && finding.SubjectKind != "semantic-attribute" {
			t.Fatalf("unexpected source subject: %+v", finding)
		}
	}
	again, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals)
	if err != nil || !reflect.DeepEqual(audit, again) {
		t.Fatalf("unstable provenance audit: %+v / %v", again, err)
	}
}

func TestModelHypothesisSourceAuditNeverCertifiesBusinessHypothesis(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	// The semantic identifier is deliberately opaque. Even if two JSON
	// strings look identical, no key collision is established without
	// comparable domain semantics. Source anchors can still be exact.
	record := input.Example.Base[0]
	record.ID = "second-child"
	input.Example.Base = append(input.Example.Base, record)
	review, err := ReviewModelHypotheses(input, proposals)
	if err != nil {
		t.Fatal(err)
	}
	if review.Issues[0].Result != "unresolved" || review.Issues[0].IndeterminatePairs != 1 ||
		len(review.Issues[0].RecordPairs) != 0 {
		t.Fatalf("opaque business identifiers were compared as raw strings: %+v", review.Issues)
	}
	audit, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Findings) != 4 {
		t.Fatalf("lost location evidence: %+v", audit.Findings)
	}
	for _, finding := range audit.Findings {
		if finding.Status != "source-range-checked" {
			t.Fatalf("definition disappeared: %+v", finding)
		}
	}
	encoded, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "accepted") || strings.Contains(string(encoded), "add-relation") ||
		strings.Contains(string(encoded), "confirmed-business-key") {
		t.Fatalf("source locator audit falsely accepted the hypothesis: %s", encoded)
	}
}

func TestModelHypothesisSourceAuditRejectsForgedCatalogAndStaleWorkspace(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	input.Catalog.Relations = nil
	if _, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals); err == nil ||
		!strings.Contains(err.Error(), "differs") {
		t.Fatalf("forged Catalog with copied revision was trusted: %v", err)
	}
	root, input, proposals = modelHypothesisAuditFixture(t)
	path := filepath.Join(root, "semantic", "model.tei")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Changing a semantic definition invalidates the existing revision
	// while leaving the old illustration untouched.
	changed := strings.Replace(string(data), "attribute parentId", "attribute changedId", 1)
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals); err == nil {
		t.Fatal("stale model and stale hypothesis were treated as current")
	}
}

func TestModelHypothesisSourceAuditHandlesUnindexedAndAmbiguousSubjects(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	attribute := sourceindex.Subject{Kind: "semantic-attribute", ID: proposals.CandidateKeys[0].Fields[0]}
	remove := make([]sourceindex.Reference, 0, len(idx.References))
	var original sourceindex.Reference
	for _, ref := range idx.References {
		if ref.Subject == attribute && ref.Role == sourceindex.RoleDefinition {
			original = ref
			continue
		}
		remove = append(remove, ref)
	}
	if original.Subject.Kind == "" {
		t.Fatal("missing fixture source reference")
	}
	unindexed := idx
	unindexed.References = remove
	got, err := AuditModelHypothesisSourceEvidence(input, proposals, unindexed, read)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Findings {
		if f.SubjectKind == "semantic-attribute" && f.Status != "definition-not-indexed" {
			t.Fatalf("unindexed attribute was assumed to have a source: %+v", f)
		}
	}
	ambiguous := idx
	ambiguous.References = append(append([]sourceindex.Reference(nil), idx.References...), original)
	got, err = AuditModelHypothesisSourceEvidence(input, proposals, ambiguous, read)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Findings {
		if f.SubjectKind == "semantic-attribute" && f.Status != "ambiguous-definition" {
			t.Fatalf("ambiguous attribute was auto-selected: %+v", f)
		}
	}
}

func TestModelHypothesisSourceAuditRefusesIncorrectSemanticAnchor(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	// A range pointing to the unrelated 'data Parent' token is still
	// syntactically valid. The parser identity check must reject it.
	content, err := read("semantic/model.tei")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(content), "Parent")
	if start < 0 {
		t.Fatal("fixture changed")
	}
	wrong, err := sourceindex.RangeFromOffsets(content, start, start+len("Parent"))
	if err != nil {
		t.Fatal(err)
	}
	refs := append([]sourceindex.Reference(nil), idx.References...)
	for i := range refs {
		if refs[i].Subject.Kind == "semantic-attribute" &&
			refs[i].Subject.ID == proposals.CandidateKeys[0].Fields[0] {
			refs[i].Range = wrong
		}
	}
	idx.References = refs
	got, err := AuditModelHypothesisSourceEvidence(input, proposals, idx, read)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range got.Findings {
		if f.SubjectKind == "semantic-attribute" &&
			(f.Status != "exact-subject-anchor-not-confirmed" || f.SourceRange != nil) {
			t.Fatalf("unrelated token passed source verification: %+v", f)
		}
	}
}

func TestModelHypothesisSourceAuditFailsClosedOnTamperAndUnboundReader(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuditModelHypothesisSourceEvidence(input, proposals, idx, nil); err == nil {
		t.Fatal("nil source reader accepted")
	}
	readChanged := func(path string) ([]byte, error) {
		data, err := read(path)
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	}
	if _, err := AuditModelHypothesisSourceEvidence(input, proposals, idx, readChanged); err == nil {
		t.Fatal("tampered source bytes passed fingerprint verification")
	}
	readFailure := func(string) ([]byte, error) { return nil, fmt.Errorf("reader failed") }
	if _, err := AuditModelHypothesisSourceEvidence(input, proposals, idx, readFailure); err == nil {
		t.Fatal("reader error did not fail closed")
	}
}

func TestModelHypothesisSourceAuditRejectsInvalidQuestionWithoutPartialEvidence(t *testing.T) {
	root, input, proposals := modelHypothesisAuditFixture(t)
	proposals.CandidateKeys[0].Fields = []string{"unknown.attribute"}
	if _, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposals); err == nil {
		t.Fatal("invalid key declaration was evaluated against source artifacts")
	}
}

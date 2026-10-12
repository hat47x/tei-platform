package exampleslide

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func sourceEvidenceFixture(t *testing.T) (Input, AnswerInput, sourceindex.Index, func(string) ([]byte, error)) {
	t.Helper()
	in, proposed := reviewAnswerFixture(t)
	// Actual TEI syntax stores the short token "parent" for a qualified
	// semantic-relation subject "sales.child.parent".
	in.Catalog.Relations[0].ID = "sales.child.parent"
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	proposed.ReviewFingerprint = review.Fingerprint
	proposed.Answers[0].Question.ID = review.Issues[0].ID
	content := []byte("namespace sales\n\ndata Child {\n    attribute parentId type std.identifier\n    relation parent to sales.parent one\n}\n")
	id := "sales.child.parent"
	offset := strings.Index(string(content), "relation parent") + len("relation ")
	if offset < len("relation ") {
		t.Fatal("missing source anchor in fixture")
	}
	artifact, err := sourceindex.NewArtifact("semantic/sales.tei", content)
	if err != nil {
		t.Fatal(err)
	}
	location, err := sourceindex.RangeFromOffsets(content, offset, offset+len("parent"))
	if err != nil {
		t.Fatal(err)
	}
	idx := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{artifact},
		References: []sourceindex.Reference{{
			Subject:      sourceindex.Subject{Kind: "semantic-relation", ID: id},
			ArtifactPath: artifact.Path,
			Range:        location,
			Role:         sourceindex.RoleDefinition,
		}},
	}
	proposed.Answers[0].Evidence = []EvidencePointer{{
		Kind: "source-index", Ref: "semantic-relation:" + id,
		Fingerprint: artifact.Fingerprint,
	}}
	read := func(path string) ([]byte, error) {
		if path != artifact.Path {
			return nil, fmt.Errorf("unknown source %s", path)
		}
		return append([]byte(nil), content...), nil
	}
	return in, proposed, idx, read
}

func TestAuditAnswerSourceEvidenceChecksExactCurrentDefinition(t *testing.T) {
	in, proposed, idx, read := sourceEvidenceFixture(t)
	audit, err := AuditAnswerSourceEvidence(in, proposed, idx, read)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Findings) != 1 || audit.Findings[0].Status != "source-range-checked" ||
		audit.Findings[0].ArtifactPath != "semantic/sales.tei" ||
		audit.Findings[0].ArtifactFingerprint != idx.Artifacts[0].Fingerprint {
		t.Fatalf("missing exact source evidence: %+v", audit)
	}
	handoff, err := BindAnswers(in, proposed)
	if err != nil {
		t.Fatal(err)
	}
	if audit.AnswerFingerprint != handoff.Fingerprint || audit.ReviewFingerprint != handoff.ReviewFingerprint {
		t.Fatalf("audit does not bind to the exact answer: %+v", audit)
	}
	again, err := AuditAnswerSourceEvidence(in, proposed, idx, read)
	if err != nil || !reflect.DeepEqual(audit, again) {
		t.Fatalf("nondeterministic source inspection: %+v, %v", again, err)
	}
}

func TestAuditAnswerSourceEvidenceRejectsNonMatchingSubjectAndFingerprint(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Input, *AnswerInput, *sourceindex.Index)
		status string
	}{
		{"unrelated relation", func(_ *Input, p *AnswerInput, _ *sourceindex.Index) {
			p.Answers[0].Evidence[0].Ref = "semantic-relation:unrelated"
		}, "different-subject"},
		{"wrong kind", func(_ *Input, p *AnswerInput, _ *sourceindex.Index) {
			p.Answers[0].Evidence[0].Ref = "semantic-data:sales.child.parent"
		}, "different-subject"},
		{"missing fingerprint", func(_ *Input, p *AnswerInput, _ *sourceindex.Index) {
			p.Answers[0].Evidence[0].Fingerprint = ""
		}, "evidence-fingerprint-missing"},
		{"stale evidence fingerprint", func(_ *Input, p *AnswerInput, _ *sourceindex.Index) {
			p.Answers[0].Evidence[0].Fingerprint = "sha256:older"
		}, "evidence-fingerprint-mismatch"},
		{"subject absent from index", func(_ *Input, _ *AnswerInput, idx *sourceindex.Index) {
			idx.References = nil
		}, "definition-not-indexed"},
		{"non-definition role", func(_ *Input, _ *AnswerInput, idx *sourceindex.Index) {
			idx.References[0].Role = sourceindex.RoleDerived
		}, "definition-not-indexed"},
		{"ambiguous definitions", func(_ *Input, _ *AnswerInput, idx *sourceindex.Index) {
			idx.References = append(idx.References, idx.References[0])
		}, "ambiguous-definition"},
		{"misplaced same-length anchor", func(_ *Input, _ *AnswerInput, idx *sourceindex.Index) {
			// An index range may be in-bounds and hash-valid while pointing
			// to a different source token. The parser range must match too.
			ref := &idx.References[0]
			ref.Range.End.Offset = ref.Range.Start.Offset + len("parent")
			ref.Range.End.Column = ref.Range.Start.Column + len("parent")
			ref.Range.Start.Offset -= len("relation ")
			ref.Range.Start.Column -= len("relation ")
			ref.Range.End.Offset -= len("relation ")
			ref.Range.End.Column -= len("relation ")
		}, "exact-subject-anchor-not-confirmed"},
		{"coarse point location", func(_ *Input, _ *AnswerInput, idx *sourceindex.Index) {
			idx.References[0].Range.End = idx.References[0].Range.Start
		}, "exact-subject-anchor-not-confirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, proposed, idx, read := sourceEvidenceFixture(t)
			tc.change(&in, &proposed, &idx)
			got, err := AuditAnswerSourceEvidence(in, proposed, idx, read)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Findings) != 1 || got.Findings[0].Status != tc.status {
				t.Fatalf("expected %q, got %+v", tc.status, got)
			}
		})
	}
}

func TestAuditAnswerSourceEvidenceFailsClosedOnStaleIndex(t *testing.T) {
	in, proposed, idx, read := sourceEvidenceFixture(t)
	idx.Artifacts[0].Fingerprint = "sha256:outdated"
	if got, err := AuditAnswerSourceEvidence(in, proposed, idx, read); err == nil ||
		len(got.Findings) != 0 {
		t.Fatalf("stale source index was accepted: %+v, %v", got, err)
	}
	if got, err := AuditAnswerSourceEvidence(in, proposed, idx, nil); err == nil ||
		len(got.Findings) != 0 {
		t.Fatalf("missing trusted reader was accepted: %+v, %v", got, err)
	}
}

func TestAuditAnswerSourceEvidenceDetectsChangedSourceBetweenReads(t *testing.T) {
	in, proposed, idx, read := sourceEvidenceFixture(t)
	var count int
	unstable := func(path string) ([]byte, error) {
		count++
		content, err := read(path)
		if count > 1 {
			content = append(content, []byte("new content")...)
		}
		return content, err
	}
	if audit, err := AuditAnswerSourceEvidence(in, proposed, idx, unstable); err == nil ||
		len(audit.Findings) != 0 {
		t.Fatalf("source was changed after validation: %+v, %v", audit, err)
	}
}

func TestAuditAnswerSourceEvidenceDoesNotInventAuthoredOrDocumentProvenance(t *testing.T) {
	in, proposed, idx, read := sourceEvidenceFixture(t)
	proposed.Answers[0].Evidence = []EvidencePointer{{Kind: "document", Ref: "doc:minutes"}}
	result, err := AuditAnswerSourceEvidence(in, proposed, idx, read)
	if err != nil || len(result.Findings) != 1 || result.Findings[0].Status != "not-checked" {
		t.Fatalf("arbitrary document was treated as indexed source: %+v, %v", result, err)
	}
	in = fixture(t)
	review, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	proposed = AnswerInput{
		Schema: answerSchema, CaseID: review.CaseID, ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
		Answers: []AnswerEntry{{
			Question: QuestionRef{Kind: "authored", ID: "residual"},
			Response: "要調査", Disposition: "investigate", Context: "発注の例",
			ClaimedBy: AnswerActor{Kind: "human", ID: "unverified"},
			Evidence: []EvidencePointer{{Kind: "source-index", Ref: "semantic-relation:sales.child.parent",
				Fingerprint: idx.Artifacts[0].Fingerprint}},
		}},
	}
	result, err = AuditAnswerSourceEvidence(in, proposed, idx, read)
	if err != nil || len(result.Findings) != 1 || result.Findings[0].Status != "no-derived-relation" {
		t.Fatalf("authored question acquired a fabricated relation: %+v, %v", result, err)
	}
	raw, err := json.Marshal(result)
	if err != nil || strings.Contains(string(raw), "source-range-checked") {
		t.Fatalf("unproven authored reference became trusted: %s, %v", raw, err)
	}
}

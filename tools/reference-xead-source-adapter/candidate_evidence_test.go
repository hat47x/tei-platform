package main

import (
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
)

func TestBuildTranscriptionCandidateEvidenceForKnownMappedNode(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table ID="private-table"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := BuildTranscriptionCandidateEvidence(
		FormatXead,
		doc,
		classificationContractFixture(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.MappingStatus != rmtevidence.MappingMapped ||
		!evidence.SourceConstructKnown ||
		evidence.SourceConstruct != "Table" ||
		evidence.Extension != "reference.xead.relational-semantics" {
		t.Fatalf("evidence=%+v", evidence)
	}
	if evidence.SourceToken != sourceNodeToken(doc.ArtifactSHA256, 2) {
		t.Fatalf("source token=%q", evidence.SourceToken)
	}
	if evidence.SourceArtifactDigest != "sha256:"+doc.ArtifactSHA256 {
		t.Fatalf("source artifact=%q", evidence.SourceArtifactDigest)
	}
}

func TestBuildTranscriptionCandidateEvidenceRedactsUnknownConstructName(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><PrivateBusinessThing Secret="private-value"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := BuildTranscriptionCandidateEvidence(
		FormatXead,
		doc,
		classificationContractFixture(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.SourceConstructKnown || evidence.SourceConstruct != "" || evidence.MappingStatus != rmtevidence.MappingUnmapped {
		t.Fatalf("unknown construct was promoted: %+v", evidence)
	}
	encoded, err := rmtevidence.CanonicalBytes(evidence)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"PrivateBusinessThing", "Secret", "private-value"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("RMT Evidence leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestBuildTranscriptionCandidateEvidenceDoesNotPromoteNamespacedLookalike(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2" xmlns:x="urn:private"><x:Table ID="private"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := BuildTranscriptionCandidateEvidence(
		FormatXead,
		doc,
		classificationContractFixture(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.SourceConstructKnown || evidence.SourceConstruct != "" ||
		evidence.MappingStatus != rmtevidence.MappingUnmapped ||
		evidence.Extension != "" {
		t.Fatalf("namespaced lookalike was promoted: %+v", evidence)
	}
}

func TestBuildTranscriptionCandidateEvidenceRejectsUnsupportedSourceProfile(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="9.9"><Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildTranscriptionCandidateEvidence(
		FormatXead,
		doc,
		classificationContractFixture(),
		2,
	); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unsupported source profile was accepted: %v", err)
	}
}

func TestBuildTranscriptionCandidateEvidenceRejectsInvalidOrdinal(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildTranscriptionCandidateEvidence(
		FormatXead,
		doc,
		classificationContractFixture(),
		0,
	); err == nil || !strings.Contains(err.Error(), "ordinal") {
		t.Fatalf("invalid source ordinal was accepted: %v", err)
	}
}

func TestVerifyTranscriptionCandidateEvidenceRejectsSourceOrContractDrift(t *testing.T) {
	contract := classificationContractFixture()
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table ID="a"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := BuildTranscriptionCandidateEvidence(FormatXead, doc, contract, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, doc, contract, recorded); err != nil {
		t.Fatalf("current Evidence did not replay: %v", err)
	}

	changedSource, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table ID="b"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, changedSource, contract, recorded); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("source drift was accepted: %v", err)
	}

	changedContract := append([]byte(nil), contract...)
	changedContract = append(changedContract, ' ')
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, doc, changedContract, recorded); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("mapping contract drift was accepted: %v", err)
	}
}

func TestVerifyTranscriptionCandidateEvidenceResolvesRecordedTokenWithoutOrdinal(t *testing.T) {
	contract := classificationContractFixture()
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table ID="a"/><Table ID="b"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := BuildTranscriptionCandidateEvidence(FormatXead, doc, contract, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, doc, contract, recorded); err != nil {
		t.Fatalf("Evidence could not be replayed from its own source token: %v", err)
	}

	recorded.SourceToken = "sha256:" + strings.Repeat("f", 64)
	recorded, err = rmtevidence.Prepare(recorded)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyTranscriptionCandidateEvidence(FormatXead, doc, contract, recorded); err == nil ||
		!strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("nonexistent source token was accepted: %v", err)
	}
}

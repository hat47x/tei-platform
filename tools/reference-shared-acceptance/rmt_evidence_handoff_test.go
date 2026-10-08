package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
)

func rmtEvidenceFixture(t *testing.T, mapped bool) rmtevidence.Evidence {
	t.Helper()
	value := rmtevidence.Evidence{
		Schema:                     rmtevidence.Schema,
		SourceFormat:               "xead",
		SourceFormatVersion:        "1.2",
		SourceArtifactDigest:       "sha256:" + strings.Repeat("1", 64),
		SourceToken:                "sha256:" + strings.Repeat("2", 64),
		MappingContractFingerprint: "sha256:" + strings.Repeat("3", 64),
		AuthorityLimits: []string{
			"source-observation-is-not-canonical-authority",
			"mapping-is-candidate-not-semantic-acceptance",
			"rmt-evidence-does-not-grant-runtime-authority",
		},
	}
	if mapped {
		value.SourceConstructKnown = true
		value.SourceConstruct = "Table"
		value.MappingStatus = rmtevidence.MappingMapped
		value.Classification = []string{"canonical-candidate", "unresolved"}
		value.Extension = "reference.xead.relational-semantics"
	} else {
		value.MappingStatus = rmtevidence.MappingUnmapped
		value.Classification = []string{"unresolved"}
		value.AuthorityLimits = append(value.AuthorityLimits, "unknown-source-construct-must-not-be-silently-dropped")
	}
	prepared, err := rmtevidence.Prepare(value)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func rmtEvidenceJSON(t *testing.T, mapped bool) []byte {
	t.Helper()
	value := rmtEvidenceFixture(t, mapped)
	encoded, err := rmtevidence.CanonicalBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestRMTTranscriptionEvidenceDoesNotBecomeCandidateMeaningOrAcceptanceAuthority(t *testing.T) {
	for _, mapped := range []bool{true, false} {
		name := "unmapped"
		if mapped {
			name = "mapped"
		}
		t.Run(name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			candidate := unitPriceCandidate(revisionOfDir(t, dir))
			_, semanticBefore, err := normalizeCandidate(candidate)
			if err != nil {
				t.Fatal(err)
			}

			grounded, err := attachRMTTranscriptionEvidence(
				mustJSON(t, candidate),
				rmtEvidenceJSON(t, mapped),
			)
			if err != nil {
				t.Fatal(err)
			}
			_, semanticAfter, err := normalizeCandidate(grounded)
			if err != nil {
				t.Fatal(err)
			}
			if semanticBefore != semanticAfter {
				t.Fatalf("RMT Evidence changed candidate meaning: before=%s after=%s", semanticBefore, semanticAfter)
			}

			item, ok := rmtTranscriptionEvidenceItem(grounded.Evidence)
			if !ok || item.Artifact == nil {
				t.Fatalf("RMT artifact was not attached: %+v", grounded.Evidence)
			}
			if item.Subject != "" || item.Source != "" {
				t.Fatalf("RMT artifact was conflated with workspace Source Index Evidence: %+v", item)
			}

			ev, err := evaluate(dir, mustJSON(t, grounded))
			if err != nil {
				t.Fatal(err)
			}
			if !ev.preview.Acceptable {
				t.Fatalf("RMT mapping status became Acceptance authority: %+v", ev.preview.Stages)
			}
			review := buildReviewProjection(ev)
			rendered := renderReviewProjection(review)
			wantStatus := "mappingStatus=unmapped"
			if mapped {
				wantStatus = "mappingStatus=mapped"
			}
			if !strings.Contains(rendered, wantStatus) ||
				!strings.Contains(rendered, rmtEvidenceFixture(t, mapped).SourceToken) {
				t.Fatalf("Review hid RMT Evidence state/binding:\\n%s", rendered)
			}
			if mapped {
				if !strings.Contains(rendered, "sourceConstruct=Table") ||
					!strings.Contains(rendered, "classification=canonical-candidate,unresolved") {
					t.Fatalf("Review hid mapped RMT summary:\\n%s", rendered)
				}
			}
		})
	}
}

func TestRMTTranscriptionAttachmentRejectsTampering(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	value := rmtEvidenceFixture(t, true)
	value.Extension = "other.extension"
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attachRMTTranscriptionEvidence(mustJSON(t, candidate), encoded); err == nil ||
		!strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("tampered RMT Evidence was accepted: %v", err)
	}
}

func TestAttachRMTTranscriptionCLIProducesReviewableCandidate(t *testing.T) {
	dir := scratchWorkspace(t)
	temp := t.TempDir()
	candidatePath := filepath.Join(temp, "candidate.json")
	evidencePath := filepath.Join(temp, "rmt-evidence.json")

	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, rmtEvidenceJSON(t, false), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := cliRun(
		t,
		"attach-rmt-transcription",
		"-candidate", candidatePath,
		"-rmt-transcription", evidencePath,
	)
	if code != 0 {
		t.Fatalf("attach-rmt-transcription code=%d stderr=%s", code, errOut)
	}
	grounded, err := decodeCandidate([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rmtTranscriptionEvidenceItem(grounded.Evidence); !ok {
		t.Fatalf("CLI output lacks RMT artifact Evidence: %+v", grounded)
	}
	if _, _, err := normalizeCandidate(grounded); err != nil {
		t.Fatalf("CLI output is not a normalized candidate: %v", err)
	}

	ev, err := evaluate(dir, []byte(out))
	if err != nil {
		t.Fatal(err)
	}
	review := buildReviewProjection(ev)
	rendered := renderReviewProjection(review)
	if !strings.Contains(rendered, "artifactKind="+evidenceArtifactKindRMTTranscription) ||
		!strings.Contains(rendered, "mappingStatus=unmapped") {
		t.Fatalf("Review did not expose RMT artifact state:\\n%s", rendered)
	}
}

func TestRMTTranscriptionAttachmentIsIdempotent(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	first, err := attachRMTTranscriptionEvidence(
		mustJSON(t, candidate),
		rmtEvidenceJSON(t, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := attachRMTTranscriptionEvidence(
		mustJSON(t, first),
		rmtEvidenceJSON(t, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Evidence) != len(candidate.Evidence)+1 {
		t.Fatalf("same RMT artifact Evidence was duplicated: %+v", second.Evidence)
	}
}

func rmtTranscriptionEvidenceItem(values []Evidence) (Evidence, bool) {
	for _, value := range values {
		if value.Artifact != nil && value.Artifact.Kind == evidenceArtifactKindRMTTranscription {
			return value, true
		}
	}
	return Evidence{}, false
}


func TestDecisionIndexDoesNotCallRMTArtifactCurrent(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidate.Evidence = nil
	grounded, err := attachRMTTranscriptionEvidence(
		mustJSON(t, candidate),
		rmtEvidenceJSON(t, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision := acceptViaCLI(t, dir, candidateFile(t, grounded))
	if decision.Decision != "accepted" {
		t.Fatalf("decision=%+v", decision)
	}

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("records=%+v", index.Records)
	}
	record := index.Records[0]
	if record.EvidenceStatus != "recorded" ||
		record.SourceEvidenceStatus != "none" ||
		record.ArtifactEvidenceStatus != "recorded" {
		t.Fatalf("external RMT artifact was overstated as current: %+v", record)
	}
}

func TestUnmappedRMTReviewDoesNotInventSourceConstructName(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	grounded, err := attachRMTTranscriptionEvidence(
		mustJSON(t, candidate),
		rmtEvidenceJSON(t, false),
	)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil {
		t.Fatal(err)
	}
	rendered := renderReviewProjection(buildReviewProjection(ev))
	if strings.Contains(rendered, "sourceConstruct=") {
		t.Fatalf("unmapped RMT Evidence invented or exposed a source construct name:\n%s", rendered)
	}
}

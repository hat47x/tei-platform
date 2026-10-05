package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func semanticEvaluationFixture(status string) semanticEvaluationArtifact {
	value := semanticEvaluationArtifact{
		Schema:                         semanticEvaluationEvidenceSchema,
		SemanticSetFingerprint:         "sha256:" + strings.Repeat("1", 64),
		ConstraintExtension:            "sui.hierarchy",
		ConstraintID:                   "sui-island-hierarchy-integrity",
		Subject:                        subject.Subject{Kind: "semantic-data", ID: "sui.island"},
		ValidatorRef:                   "reference.hierarchy.integrity.v1",
		EvaluatorInterface:             "reference.hierarchy.integrity.evaluator",
		EvaluatorContractFingerprint:   "sha256:" + strings.Repeat("2", 64),
		EvaluatorConformanceProfile:    "reference.hierarchy.integrity/v1",
		EvaluatorProviderExtension:     "reference.hierarchy-evaluator",
		EvaluatorImplementationID:      "reference.sui-hierarchy-shadow-evaluator/v1",
		EvaluatorArtifactDigest:        "sha256:" + strings.Repeat("3", 64),
		ConformanceEvidenceFingerprint: "sha256:" + strings.Repeat("4", 64),
		InputFingerprint:               "sha256:" + strings.Repeat("5", 64),
		Status:                         status,
	}
	if status != "pass" {
		value.Diagnostics = []semanticEvaluationDiagnostic{{
			Code:    "REFERENCE_EVALUATION_" + strings.ToUpper(status),
			Message: "reference evaluation returned " + status,
		}}
	}
	value.EvidenceFingerprint = semanticEvaluationArtifactEvidenceFingerprint(value)
	return value
}

func semanticEvaluationJSON(t *testing.T, status string) []byte {
	t.Helper()
	encoded, err := json.Marshal(semanticEvaluationFixture(status))
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestSemanticEvaluationEvidenceDoesNotBecomeCandidateMeaningOrAcceptanceAuthority(t *testing.T) {
	for _, status := range []string{"pass", "fail", "unknown"} {
		t.Run(status, func(t *testing.T) {
			dir := scratchWorkspace(t)
			candidate := unitPriceCandidate(revisionOfDir(t, dir))
			_, semanticBefore, err := normalizeCandidate(candidate)
			if err != nil {
				t.Fatal(err)
			}

			grounded, err := attachSemanticEvaluationEvidence(
				mustJSON(t, candidate),
				semanticEvaluationJSON(t, status),
			)
			if err != nil {
				t.Fatal(err)
			}
			_, semanticAfter, err := normalizeCandidate(grounded)
			if err != nil {
				t.Fatal(err)
			}
			if semanticBefore != semanticAfter {
				t.Fatalf("artifact Evidence changed candidate meaning: before=%s after=%s", semanticBefore, semanticAfter)
			}
			if len(grounded.Evidence) != 1 || grounded.Evidence[0].Artifact == nil {
				t.Fatalf("semantic evaluation artifact was not attached: %+v", grounded.Evidence)
			}
			if grounded.Evidence[0].Subject != "" || grounded.Evidence[0].Source != "" {
				t.Fatalf("artifact Evidence was conflated with Source Index Evidence: %+v", grounded.Evidence[0])
			}

			ev, err := evaluate(dir, mustJSON(t, grounded))
			if err != nil {
				t.Fatal(err)
			}
			if !ev.preview.Acceptable {
				t.Fatalf("evaluation status %q became Acceptance authority: %+v", status, ev.preview.Stages)
			}
			review := buildReviewProjection(ev)
			rendered := renderReviewProjection(review)
			if !strings.Contains(rendered, "status="+status) ||
				!strings.Contains(rendered, grounded.Evidence[0].Artifact.ArtifactFingerprint) {
				t.Fatalf("review hid semantic evaluation state/binding:
%s", rendered)
			}

			decision, err := accept(dir, mustJSON(t, grounded), acceptOptions{
				ExpectedReviewFingerprint: review.ReviewFingerprint,
			})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Decision != "accepted" {
				t.Fatalf("explicit Acceptance was replaced by evaluation status %q: %+v", status, decision)
			}
			if decision.EvidenceFingerprint != ev.preview.EvidenceFingerprint ||
				len(decision.Evidence) != 1 ||
				decision.Evidence[0].Artifact == nil {
				t.Fatalf("Decision did not bind exact artifact Evidence: %+v", decision)
			}
		})
	}
}

func TestSemanticEvaluationAttachmentRejectsTamperingAndMixedAuthorityShapes(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))

	t.Run("internal fingerprint mismatch", func(t *testing.T) {
		value := semanticEvaluationFixture("pass")
		value.Status = "fail"
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := attachSemanticEvaluationEvidence(mustJSON(t, candidate), encoded); err == nil ||
			!strings.Contains(err.Error(), "fingerprint") {
			t.Fatalf("tampered evaluation was accepted: %v", err)
		}
	})

	t.Run("artifact fingerprint mismatch", func(t *testing.T) {
		value := semanticEvaluationFixture("pass")
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		bad := candidate
		bad.Evidence = []Evidence{{
			Artifact: &EvidenceArtifact{
				Kind:                evidenceArtifactKindSemanticEvaluation,
				Schema:              semanticEvaluationEvidenceSchema,
				ArtifactFingerprint: "sha256:" + strings.Repeat("0", 64),
				Payload:             payload,
			},
		}}
		if _, _, err := normalizeCandidate(bad); err == nil ||
			!strings.Contains(err.Error(), "artifact Evidence fingerprint") {
			t.Fatalf("wrong artifact fingerprint was accepted: %v", err)
		}
	})

	t.Run("source and artifact mixed", func(t *testing.T) {
		value := semanticEvaluationFixture("pass")
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		bad := candidate
		bad.Evidence = []Evidence{{
			Subject: "sales.order_line.quantity_positive",
			Artifact: &EvidenceArtifact{
				Kind:                evidenceArtifactKindSemanticEvaluation,
				Schema:              semanticEvaluationEvidenceSchema,
				ArtifactFingerprint: sourceindex.Fingerprint(payload),
				Payload:             payload,
			},
		}}
		if _, _, err := normalizeCandidate(bad); err == nil ||
			!strings.Contains(err.Error(), "混在できません") {
			t.Fatalf("mixed authority Evidence was accepted: %v", err)
		}
	})
}

func TestAttachSemanticEvaluationCLIProducesReviewableCandidate(t *testing.T) {
	dir := scratchWorkspace(t)
	temp := t.TempDir()
	candidatePath := filepath.Join(temp, "candidate.json")
	evaluationPath := filepath.Join(temp, "semantic-evaluation.json")

	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evaluationPath, semanticEvaluationJSON(t, "fail"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := cliRun(
		t,
		"attach-semantic-evaluation",
		"-candidate", candidatePath,
		"-semantic-evaluation", evaluationPath,
	)
	if code != 0 {
		t.Fatalf("attach-semantic-evaluation code=%d stderr=%s", code, errOut)
	}
	grounded, err := decodeCandidate([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(grounded.Evidence) != 1 || grounded.Evidence[0].Artifact == nil {
		t.Fatalf("CLI output lacks artifact Evidence: %+v", grounded)
	}
	if _, _, err := normalizeCandidate(grounded); err != nil {
		t.Fatalf("CLI output is not a normalized candidate: %v", err)
	}

	ev, err := evaluate(dir, []byte(out))
	if err != nil {
		t.Fatal(err)
	}
	review := buildReviewProjection(ev)
	if !strings.Contains(renderReviewProjection(review), "status=fail") {
		t.Fatal("Review did not expose failed semantic evaluation")
	}
}

func TestSemanticEvaluationAttachmentIsIdempotent(t *testing.T) {
	dir := scratchWorkspace(t)
	first, err := attachSemanticEvaluationEvidence(
		mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))),
		semanticEvaluationJSON(t, "pass"),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := attachSemanticEvaluationEvidence(
		mustJSON(t, first),
		semanticEvaluationJSON(t, "pass"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Evidence) != 1 {
		t.Fatalf("same artifact Evidence was duplicated: %+v", second.Evidence)
	}
}

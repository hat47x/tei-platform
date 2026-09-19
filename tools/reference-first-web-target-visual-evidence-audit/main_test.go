package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuditVisualEvidencePassesAttributedVisualReviewButNotGateC(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	summary, err := auditVisualEvidence(semanticPath, manifestPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.AllRequiredCapturesPresent {
		t.Fatal("required captures were not retained")
	}
	if !summary.AllCriteriaResolved {
		t.Fatalf("criteria resolved=%v", summary.AllCriteriaResolved)
	}
	if !summary.VisualFidelityPassed {
		t.Fatalf("visual fidelity passed=%v", summary.VisualFidelityPassed)
	}
	if len(summary.Captures) != len(requiredCaptures) {
		t.Fatalf("capture digests=%d want=%d", len(summary.Captures), len(requiredCaptures))
	}
	if summary.GateCReady {
		t.Fatal("Visual pass must not auto-declare Gate C Ready")
	}
}

func TestAuditVisualEvidenceKeepsUnresolvedCriterionVisible(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.Criteria[0].Result = "unresolved"
	writeVisualJSON(t, manifestPath, manifest)

	summary, err := auditVisualEvidence(semanticPath, manifestPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AllCriteriaResolved {
		t.Fatal("unresolved Visual criterion was hidden")
	}
	if summary.VisualFidelityPassed {
		t.Fatal("unresolved Visual criterion was treated as pass")
	}
	if summary.GateCReady {
		t.Fatal("unresolved Visual criterion auto-declared Gate C Ready")
	}
}

func TestAuditVisualEvidenceRejectsTargetRevisionMismatch(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.TargetBuildRevision = "other-target"
	writeVisualJSON(t, manifestPath, manifest)

	if _, err := auditVisualEvidence(semanticPath, manifestPath, dir); err == nil {
		t.Fatal("Visual artifact from another Target revision was accepted")
	}
}

func TestAuditVisualEvidenceRejectsMissingRequiredCapture(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.Captures = manifest.Captures[1:]
	writeVisualJSON(t, manifestPath, manifest)

	if _, err := auditVisualEvidence(semanticPath, manifestPath, dir); err == nil {
		t.Fatal("missing required Visual capture was accepted")
	}
}

func TestAuditVisualEvidenceRejectsUnsafeCapturePath(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.Captures[0].File = "../outside.png"
	writeVisualJSON(t, manifestPath, manifest)

	if _, err := auditVisualEvidence(semanticPath, manifestPath, dir); err == nil {
		t.Fatal("Visual capture path traversal was accepted")
	}
}

func TestAuditVisualEvidenceRejectsCredentialMaterialInManifest(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.Criteria[0].Notes = "editor-secret"
	writeVisualJSON(t, manifestPath, manifest)

	if _, err := auditVisualEvidence(semanticPath, manifestPath, dir); err == nil {
		t.Fatal("credential material in Visual manifest was accepted")
	}
}

func writeVisualFixture(t *testing.T, dir, targetRevision, criterionResult string) (string, string) {
	t.Helper()

	semanticPath := filepath.Join(dir, "semantic-functional-summary.json")
	writeVisualJSON(t, semanticPath, semanticFunctionalSummary{
		EvidenceNature:             "first-web-target-semantic-functional-evidence",
		RunIdentity:                "semantic-functional-run-1",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        targetRevision,
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SourceQualification:        passState{Passed: true},
		SemanticConformance:        passState{Passed: true},
		FunctionalEvidence:         passState{Passed: true},
		VisualEvidence: pendingVisualState{
			Status:   "pending",
			Required: true,
		},
		GateCReady: false,
	})

	captureIDs := []string{
		"projection-index",
		"custom-submit-initial",
		"custom-submit-success",
		"custom-submit-authorization-denied",
		"custom-submit-validation-not-satisfied",
		"custom-submit-execution-error",
	}
	captures := make([]visualCapture, 0, len(captureIDs))
	for _, id := range captureIDs {
		expected := requiredCaptures[id]
		file := id + ".png"
		writeFakePNG(t, filepath.Join(dir, file))
		captures = append(captures, visualCapture{
			ID:             id,
			Surface:        expected.Surface,
			State:          expected.State,
			File:           file,
			MediaType:      "image/png",
			ViewportWidth:  1440,
			ViewportHeight: 1000,
		})
	}

	criteria := make([]visualCriterion, 0, len(requiredCriteria))
	for _, id := range requiredCriteria {
		refs := append([]string(nil), requiredCriterionCaptureRefs[id]...)
		criteria = append(criteria, visualCriterion{
			ID:          id,
			Result:      criterionResult,
			CaptureRefs: refs,
			Notes:       "Reference evaluator note",
		})
	}

	manifestPath := filepath.Join(dir, "visual-manifest.json")
	writeVisualJSON(t, manifestPath, visualManifest{
		EvidenceNature:                      "first-web-target-visual-evidence",
		VisualRunIdentity:                   "visual-run-1",
		SourceSemanticFunctionalRunIdentity: "semantic-functional-run-1",
		HandoverBaselineID:                  "B",
		BaselineRepositoryRevision:          "baseline-r",
		TargetBuildRevision:                 targetRevision,
		TargetAdapter:                       "reference-first-web-target",
		Host:                                "go-net-http",
		Evaluator: visualEvaluator{
			Identity: "reference-human-review",
			Version:  "1",
			Method:   "screenshot-review",
		},
		Captures: captures,
		Criteria: criteria,
	})

	return semanticPath, manifestPath
}

func writeFakePNG(t *testing.T, path string) {
	t.Helper()
	data := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0x00, 0x00, 0x00, 0x0d,
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeVisualJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readVisualJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildVisualManifestScaffoldCopiesProvenanceAndStartsUnresolved(t *testing.T) {
	dir := t.TempDir()
	semanticPath, _ := writeVisualFixture(t, dir, "target-t", "pass")

	var semantic semanticFunctionalSummary
	readVisualJSON(t, semanticPath, &semantic)
	manifest, err := buildVisualManifestScaffold(
		semantic,
		"visual-scaffold-run",
		visualEvaluator{
			Identity: "reference-human-review",
			Version:  "1",
			Method:   "screenshot-review",
		},
		1280,
		900,
	)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SourceSemanticFunctionalRunIdentity != semantic.RunIdentity ||
		manifest.HandoverBaselineID != semantic.HandoverBaselineID ||
		manifest.BaselineRepositoryRevision != semantic.BaselineRepositoryRevision ||
		manifest.TargetBuildRevision != semantic.TargetBuildRevision ||
		manifest.TargetAdapter != semantic.TargetAdapter ||
		manifest.Host != semantic.Host {
		t.Fatalf("scaffold provenance=%+v source=%+v", manifest, semantic)
	}
	if len(manifest.Captures) != len(requiredCaptures) {
		t.Fatalf("scaffold captures=%d want=%d", len(manifest.Captures), len(requiredCaptures))
	}
	for _, capture := range manifest.Captures {
		expected, ok := requiredCaptures[capture.ID]
		if !ok {
			t.Fatalf("scaffold contains unexpected capture %s", capture.ID)
		}
		expectedSource := requiredCaptureSources[capture.ID]
		expectedDigest := testVisualSourceDigest(expectedSource)
		if capture.Surface != expected.Surface ||
			capture.State != expected.State ||
			capture.File != capture.ID+".png" ||
			capture.MediaType != "image/png" ||
			capture.ViewportWidth != 1280 ||
			capture.ViewportHeight != 900 ||
			capture.SourceArtifact != expectedSource ||
			capture.SourceSHA256 != expectedDigest {
			t.Fatalf("scaffold capture=%+v", capture)
		}
		writeFakePNG(t, filepath.Join(dir, capture.File))
	}
	if len(manifest.Criteria) != len(requiredCriteria) {
		t.Fatalf("scaffold criteria=%d want=%d", len(manifest.Criteria), len(requiredCriteria))
	}
	for _, criterion := range manifest.Criteria {
		if criterion.Result != "unresolved" {
			t.Fatalf("generated criterion %s result=%q, want unresolved", criterion.ID, criterion.Result)
		}
		if len(criterion.CaptureRefs) == 0 {
			t.Fatalf("generated criterion %s has no capture refs", criterion.ID)
		}
	}

	manifestPath := filepath.Join(dir, "visual-scaffold.json")
	writeVisualJSON(t, manifestPath, manifest)
	summary, err := auditVisualEvidence(semanticPath, manifestPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AllCriteriaResolved ||
		summary.VisualFidelityPassed ||
		summary.GateCReady {
		t.Fatalf("unreviewed scaffold was upgraded to pass: %+v", summary)
	}
}

func TestBuildVisualManifestScaffoldRequiresAttributedEvaluator(t *testing.T) {
	dir := t.TempDir()
	semanticPath, _ := writeVisualFixture(t, dir, "target-t", "pass")
	var semantic semanticFunctionalSummary
	readVisualJSON(t, semanticPath, &semantic)

	if _, err := buildVisualManifestScaffold(
		semantic,
		"visual-scaffold-run",
		visualEvaluator{},
		1440,
		1000,
	); err == nil {
		t.Fatal("scaffold without evaluator provenance was accepted")
	}
}

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

func TestAuditVisualEvidenceRejectsReplaySourceDigestMismatch(t *testing.T) {
	dir := t.TempDir()
	semanticPath, manifestPath := writeVisualFixture(t, dir, "target-t", "pass")

	var manifest visualManifest
	readVisualJSON(t, manifestPath, &manifest)
	manifest.Captures[0].SourceSHA256 = strings.Repeat("0", 64)
	writeVisualJSON(t, manifestPath, manifest)

	if _, err := auditVisualEvidence(semanticPath, manifestPath, dir); err == nil {
		t.Fatal("Visual capture with mismatched replay source digest was accepted")
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
		Artifacts:  testVisualSourceArtifacts(),
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
		sourceArtifact := requiredCaptureSources[id]
		captures = append(captures, visualCapture{
			ID:             id,
			Surface:        expected.Surface,
			State:          expected.State,
			File:           file,
			MediaType:      "image/png",
			ViewportWidth:  1440,
			ViewportHeight: 1000,
			SourceArtifact: sourceArtifact,
			SourceSHA256:   testVisualSourceDigest(sourceArtifact),
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

func testVisualSourceArtifacts() []sourceArtifactDigest {
	files := []string{
		"projection-index.html",
		"custom-submit-get.html",
		"custom-submit-success.html",
		"custom-submit-authorization-denied.html",
		"custom-submit-validation-not-satisfied.html",
		"custom-submit-execution-error.html",
	}
	result := make([]sourceArtifactDigest, 0, len(files))
	for _, file := range files {
		result = append(result, sourceArtifactDigest{
			File:   file,
			SHA256: testVisualSourceDigest(file),
		})
	}
	return result
}

func testVisualSourceDigest(file string) string {
	switch file {
	case "projection-index.html":
		return strings.Repeat("1", 64)
	case "custom-submit-get.html":
		return strings.Repeat("2", 64)
	case "custom-submit-success.html":
		return strings.Repeat("3", 64)
	case "custom-submit-authorization-denied.html":
		return strings.Repeat("4", 64)
	case "custom-submit-validation-not-satisfied.html":
		return strings.Repeat("5", 64)
	case "custom-submit-execution-error.html":
		return strings.Repeat("6", 64)
	default:
		return ""
	}
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

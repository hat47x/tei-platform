package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReplayBundleVerifiesSameRunArtifactsAndServesExactBytes(t *testing.T) {
	dir := t.TempDir()
	summaryPath, sources := writeReplayFixture(t, dir)

	bundle, err := loadReplayBundle(summaryPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.manifest.SourceRunIdentity != "semantic-run-1" ||
		bundle.manifest.HandoverBaselineID != "B" ||
		bundle.manifest.BaselineRepositoryRevision != "baseline-r" ||
		bundle.manifest.TargetBuildRevision != "target-t" ||
		len(bundle.manifest.Captures) != len(requiredReplaySources) {
		t.Fatalf("replay manifest=%+v", bundle.manifest)
	}

	handler := newReplayHandler(bundle)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/capture/custom-submit-success", nil),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("capture status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != sources["custom-submit-success.html"] {
		t.Fatalf("replay bytes changed: got=%q want=%q", recorder.Body.String(), sources["custom-submit-success.html"])
	}
	if recorder.Header().Get("X-TEI-Visual-Capture-ID") != "custom-submit-success" ||
		recorder.Header().Get("X-TEI-Target-Build-Revision") != "target-t" ||
		recorder.Header().Get("X-TEI-Source-SHA256") != digestString(sources["custom-submit-success.html"]) {
		t.Fatalf("capture headers=%v", recorder.Header())
	}
}

func TestReplayHandlerExposesManifestAndRejectsUnknownCapture(t *testing.T) {
	dir := t.TempDir()
	summaryPath, _ := writeReplayFixture(t, dir)
	bundle, err := loadReplayBundle(summaryPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	handler := newReplayHandler(bundle)

	manifestRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		manifestRecorder,
		httptest.NewRequest(http.MethodGet, "/capture-manifest.json", nil),
	)
	if manifestRecorder.Code != http.StatusOK {
		t.Fatalf("manifest status=%d", manifestRecorder.Code)
	}
	if !strings.Contains(manifestRecorder.Body.String(), "first-web-target-visual-replay") ||
		!strings.Contains(manifestRecorder.Body.String(), "projection-index.html") {
		t.Fatalf("manifest body=%s", manifestRecorder.Body.String())
	}

	unknownRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		unknownRecorder,
		httptest.NewRequest(http.MethodGet, "/capture/unknown", nil),
	)
	if unknownRecorder.Code != http.StatusNotFound {
		t.Fatalf("unknown capture status=%d", unknownRecorder.Code)
	}

	methodRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		methodRecorder,
		httptest.NewRequest(http.MethodPost, "/capture/projection-index", nil),
	)
	if methodRecorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST capture status=%d", methodRecorder.Code)
	}
}

func TestLoadReplayBundleRejectsTamperedSource(t *testing.T) {
	dir := t.TempDir()
	summaryPath, _ := writeReplayFixture(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-success.html"),
		[]byte("<html>tampered</html>"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := loadReplayBundle(summaryPath, dir); err == nil {
		t.Fatal("tampered replay source was accepted")
	}
}

func TestLoadReplayBundleRejectsCredentialMaterial(t *testing.T) {
	dir := t.TempDir()
	summaryPath, sources := writeReplayFixture(t, dir)
	content := "<html>editor-secret</html>"
	if err := os.WriteFile(
		filepath.Join(dir, "custom-submit-success.html"),
		[]byte(content),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	var summary semanticFunctionalSummary
	readReplayJSON(t, summaryPath, &summary)
	for index := range summary.Artifacts {
		if summary.Artifacts[index].File == "custom-submit-success.html" {
			summary.Artifacts[index].SHA256 = digestString(content)
		}
	}
	writeReplayJSON(t, summaryPath, summary)
	_ = sources

	if _, err := loadReplayBundle(summaryPath, dir); err == nil {
		t.Fatal("credential-bearing replay source was accepted")
	}
}

func TestLoadReplayBundleRejectsIncompleteSemanticFunctionalState(t *testing.T) {
	dir := t.TempDir()
	summaryPath, _ := writeReplayFixture(t, dir)

	var summary semanticFunctionalSummary
	readReplayJSON(t, summaryPath, &summary)
	summary.FunctionalEvidence.Passed = false
	writeReplayJSON(t, summaryPath, summary)

	if _, err := loadReplayBundle(summaryPath, dir); err == nil {
		t.Fatal("incomplete Semantic / Functional source was accepted")
	}
}

func writeReplayFixture(t *testing.T, dir string) (string, map[string]string) {
	t.Helper()
	sources := map[string]string{
		"projection-index.html":                       "<html>projection index</html>",
		"custom-submit-get.html":                      "<html>custom initial</html>",
		"custom-submit-success.html":                  "<html>custom success</html>",
		"custom-submit-authorization-denied.html":     "<html>authorization denied</html>",
		"custom-submit-validation-not-satisfied.html": "<html>validation not satisfied</html>",
		"custom-submit-execution-error.html":          "<html>execution error</html>",
	}

	var artifacts []sourceArtifactDigest
	for file, content := range sources {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, sourceArtifactDigest{
			File:   file,
			SHA256: digestString(content),
		})
	}

	summary := semanticFunctionalSummary{
		EvidenceNature:             "first-web-target-semantic-functional-evidence",
		RunIdentity:                "semantic-run-1",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "baseline-r",
		TargetBuildRevision:        "target-t",
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		SourceQualification:        passState{Passed: true},
		SemanticConformance:        passState{Passed: true},
		FunctionalEvidence:         passState{Passed: true},
		VisualEvidence:             visualState{Status: "pending", Required: true},
		Artifacts:                  artifacts,
		GateCReady:                 false,
	}
	path := filepath.Join(dir, "semantic-functional-summary.json")
	writeReplayJSON(t, path, summary)
	return path, sources
}

func digestString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeReplayJSON(t *testing.T, path string, value any) {
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

func readReplayJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

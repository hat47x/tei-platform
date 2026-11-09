package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// §7 still says l2b.claimed needs both the in-process route and an explicit
// -rebuild-check. §4.30 made the reproduction check default when -build-catalog
// is given, so the second half of that sentence may be stale. This test measures
// which flags an attempt actually needs now, rather than repeating the premise.

type l2bDefaultResult struct {
	OK  bool `json:"ok"`
	L2B *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	PlanReceipt *struct {
		DerivedBy string `json:"derivedBy"`
		State     string `json:"state"`
	} `json:"planReceipt"`
	ArtifactReproducibility *struct {
		State             string `json:"state"`
		UnperformedReason string `json:"unperformedReason"`
	} `json:"artifactReproducibility"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runL2BDefault(t *testing.T, args ...string) (l2bDefaultResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil && code == 0 {
		t.Fatalf("run: %v", err)
	}
	var report l2bDefaultResult
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
			t.Fatalf("decode: %v: %s", decodeErr, out.String())
		}
	}
	return report, code
}

// With only -build-catalog (no -rebuild-check), the second build runs because
// §4.30 made it default on that route. The reproducibility record must therefore
// be performed, not unperformed.
func TestBuildCatalogAloneRunsTheSecondBuild(t *testing.T) {
	fixture := writeInProcessFixture(t)
	derived := fixture.deriveForTest(t)
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, code := runL2BDefault(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes")),
			"-report-intent-bytes", intentBytesDigestOf(t, fixture.intentPath),
			"-report-plan-fingerprint", derived.PlanFingerprint))
	if code != 0 {
		t.Fatalf("code=%d error=%+v", code, report.Error)
	}
	if report.ArtifactReproducibility == nil {
		t.Fatal("no reproducibility record")
	}
	if report.ArtifactReproducibility.State == reproducibilityUnperformed {
		t.Fatalf("the second build did not run although §4.30 made it default: %s",
			report.ArtifactReproducibility.UnperformedReason)
	}
	if report.ArtifactReproducibility.State != reproducibilityReproduced {
		t.Fatalf("state=%q, expected reproduced", report.ArtifactReproducibility.State)
	}
	// And with every condition met, the claim holds without an explicit flag.
	if report.L2B == nil {
		t.Fatal("no l2b record")
	}
	if !report.L2B.Claimed {
		t.Fatalf("the claim did not hold with only -build-catalog: unmet=%v", report.L2B.Unmet)
	}
}

// An explicit -rebuild-check=false still turns the second build off, and the
// reason says the caller chose it rather than that the flag was never passed.
func TestExplicitRebuildCheckFalseKeepsTheLighterAttempt(t *testing.T) {
	fixture := writeInProcessFixture(t)
	derived := fixture.deriveForTest(t)
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, _ := runL2BDefault(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", fixture.intentPath,
		"-build-catalog", fixture.catalogPath,
		"-build-catalog-digest", fixture.catalogSha,
		"-build-source-root", fixture.sourceRoot,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-rebuild-check=false",
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes")),
			"-report-intent-bytes", intentBytesDigestOf(t, fixture.intentPath),
			"-report-plan-fingerprint", derived.PlanFingerprint))
	if report.ArtifactReproducibility == nil {
		t.Fatal("no reproducibility record")
	}
	if report.ArtifactReproducibility.State != reproducibilityUnperformed {
		t.Fatalf("the second build ran despite -rebuild-check=false: %q",
			report.ArtifactReproducibility.State)
	}
	if !containsAny(report.ArtifactReproducibility.UnperformedReason, "-rebuild-check=false") {
		t.Fatalf("the reason does not name the caller's choice: %q",
			report.ArtifactReproducibility.UnperformedReason)
	}
	if report.L2B == nil || report.L2B.Claimed {
		t.Fatal("the claim held although reproduction was refused")
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if len(n) > 0 && bytes.Contains([]byte(haystack), []byte(n)) {
			return true
		}
	}
	return false
}

// intentBytesDigestOf is the digest of the declaration file, which is what the
// entry hashes and what the builder is asked to report.
func intentBytesDigestOf(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return digestOfBytes(raw)
}

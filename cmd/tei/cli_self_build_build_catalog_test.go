package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin the in-process resolution route (SB-C0 §4.22 stage 2).
//
// The point of the route is that the entry derives the plan itself from a
// caller-named catalog instead of starting a resolver child and reading its
// receipt. What it must not do is weaken the refusals the child route makes, so
// every refusal is tested here too.

const inProcessTestdata = "../../tools/reference-build-authoring-intent/testdata"

func inProcessCatalog(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(inProcessTestdata, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func inProcessIntent(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(inProcessTestdata, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func inProcessSourceRoot(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(inProcessTestdata, "src"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func inProcessCatalogDigest(t *testing.T, path string) string {
	t.Helper()
	digest, err := digestOfFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

type inProcessReport struct {
	OK                     bool `json:"ok"`
	Qualified              bool `json:"qualified"`
	BuildCatalogResolution *struct {
		State                 string `json:"state"`
		CatalogPath           string `json:"catalogPath"`
		CatalogDigest         string `json:"catalogDigest"`
		DeclaredCatalogDigest string `json:"declaredCatalogDigest"`
		IntentFingerprint     string `json:"intentFingerprint"`
		PlanFingerprint       string `json:"planFingerprint"`
		L2BClaimed            bool   `json:"l2bClaimed"`
		RuntimeClaimed        bool   `json:"runtimeAuthorityClaimed"`
		Boundary              string `json:"boundary"`
		FailureCause          string `json:"failureCause"`
	} `json:"buildCatalogResolution"`
	PlanReceipt *struct {
		State           string `json:"state"`
		Authority       string `json:"authority"`
		PlanFingerprint string `json:"planFingerprint"`
	} `json:"planReceipt"`
	PlanRecompute *struct {
		State                 string `json:"state"`
		RecomputedFingerprint string `json:"recomputedFingerprint"`
		Matched               bool   `json:"matched"`
	} `json:"planRecompute"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runInProcess(t *testing.T, args ...string) (inProcessReport, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("emitter error: %v", err)
	}
	var report inProcessReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// The entry derives the plan itself and records that it did, with the catalog
// it used and the digest it measured.
func TestInProcessDerivesThePlanFromTheNamedCatalog(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	writeArtifactAndArgs(t, output)
	catalog := inProcessCatalog(t)
	digest := inProcessCatalogDigest(t, catalog)

	report, code := runInProcess(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", inProcessIntent(t),
		"-build-catalog", catalog,
		"-build-catalog-digest", digest,
		"-build-source-root", inProcessSourceRoot(t),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", mustDigest(t, output)))
	if code != 0 {
		t.Fatalf("the in-process attempt was refused: code=%d %+v", code, report.Error)
	}
	if report.BuildCatalogResolution == nil {
		t.Fatal("the in-process route was not recorded")
	}
	res := report.BuildCatalogResolution
	if res.State != intentResolutionResolved {
		t.Fatalf("the catalog did not resolve: %+v", res)
	}
	if res.CatalogDigest != digest || res.DeclaredCatalogDigest != digest {
		t.Fatalf("the catalog digest was not recorded on both sides: %+v", res)
	}
	if res.IntentFingerprint == "" || res.PlanFingerprint == "" {
		t.Fatalf("the entry's own readings are missing: %+v", res)
	}
	// The entry derived the plan, which is what this route adds.
	if report.PlanRecompute == nil || report.PlanRecompute.State != planRecomputeMatched {
		t.Fatalf("the entry did not confirm its own derivation: %+v", report.PlanRecompute)
	}
	// What the route must not claim.
	if res.L2BClaimed || res.RuntimeClaimed || report.Qualified {
		t.Fatalf("the route claimed authority it does not have: %+v", res)
	}
	if res.Boundary == "" {
		t.Fatal("the route recorded no boundary statement")
	}
}

// Naming only one half of the pair is refused before anything starts: a catalog
// without its digest leaves "which catalog" unanswerable.
func TestInProcessRefusesACatalogWithoutItsDigest(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	report, code := runInProcess(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", inProcessIntent(t),
		"-build-catalog", inProcessCatalog(t),
		"-state", filepath.Join(dir, "state"),
		"-output", output)
	if code != 2 {
		t.Fatalf("a catalog without its digest was accepted: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "build-catalog-digest") {
		t.Fatalf("the refusal did not name the missing flag: %+v", report.Error)
	}
	if _, err := os.Stat(output); err == nil {
		t.Fatal("an artifact was produced despite the refusal")
	}
}

func TestInProcessRefusesADigestWithoutACatalog(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)

	report, code := runInProcess(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", inProcessIntent(t),
		"-build-catalog-digest", "sha256:deadbeef",
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "artifact"))
	if code != 2 {
		t.Fatalf("a digest without a catalog was accepted: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "build-catalog") {
		t.Fatalf("the refusal did not name the flag: %+v", report.Error)
	}
}

// A declared digest that does not match the catalog's measured digest is
// refused: the pair exists so the record names the catalog that was used.
func TestInProcessRefusesAMismatchedCatalogDigest(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	wrong := "sha256:" + strings.Repeat("0", 64)
	report, code := runInProcess(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", inProcessIntent(t),
		"-build-catalog", inProcessCatalog(t),
		"-build-catalog-digest", wrong,
		"-state", filepath.Join(dir, "state"),
		"-output", output)
	if code != 1 {
		t.Fatalf("a mismatched digest was accepted: code=%d", code)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "does not match") {
		t.Fatalf("the refusal did not say the digest disagreed: %+v", report.Error)
	}
	if _, err := os.Stat(output); err == nil {
		t.Fatal("an artifact was produced despite the digest mismatch")
	}
}

// A declared expectation that the derived plan does not meet stops the attempt,
// exactly as it does on the child route.
func TestInProcessHonoursTheExpectedPlanFingerprint(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	catalog := inProcessCatalog(t)

	report, code := runInProcess(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", inProcessIntent(t),
		"-build-catalog", catalog,
		"-build-catalog-digest", inProcessCatalogDigest(t, catalog),
		"-expected-plan-fingerprint", "sha256:"+strings.Repeat("1", 64),
		"-state", filepath.Join(dir, "state"),
		"-output", output)
	if code != 1 {
		t.Fatalf("a plan that missed the declared expectation was accepted: code=%d", code)
	}
	if report.Error == nil {
		t.Fatal("no reason was stated")
	}
}

// writeArtifactAndArgs creates the artifact the child will produce, so the test
// can compute the digest the child is asked to report.
func writeArtifactAndArgs(t *testing.T, output string) {
	t.Helper()
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
}

func mustDigest(t *testing.T, path string) string {
	t.Helper()
	// The artifact does not exist yet; the child writes these exact bytes.
	sum := digestOfBytes([]byte("artifact-bytes"))
	return sum
}

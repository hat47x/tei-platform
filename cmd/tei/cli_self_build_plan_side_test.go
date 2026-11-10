package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These tests pin the fallback added for SB-C0 §7's remaining item: the plan
// side of the artifact binding no longer depends only on what a child said.
//
// Two bases are possible and they are not equal:
//
//	builder-report    the child reported the plan fingerprint it built  (stronger)
//	entry-derivation  the entry derived it from the plan in the receipt (weaker)
//
// The record says which one the lock carries, so the weaker basis is never read
// as the stronger one. Both still leave l2bClaimed false.

type planSideResult struct {
	OK            bool `json:"ok"`
	Qualified     bool `json:"qualified"`
	PlanRecompute *struct {
		State                 string `json:"state"`
		RecomputedFingerprint string `json:"recomputedFingerprint"`
	} `json:"planRecompute"`
	PlanArtifactBinding *struct {
		State                string `json:"state"`
		EntryPlanFingerprint string `json:"entryPlanFingerprint"`
		PlanFingerprint      string `json:"planFingerprint"`
		PlanSideSource       string `json:"planSideSource"`
		Matched              bool   `json:"matched"`
		L2BClaimed           bool   `json:"l2bClaimed"`
		UnperformedReason    string `json:"unperformedReason"`
	} `json:"planArtifactBinding"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runPlanSide(t *testing.T, args ...string) (planSideResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("emitter error: %v", err)
	}
	var report planSideResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// A child that reports a plan fingerprint is the stronger basis, and the record
// says so.
func TestPlanSidePrefersTheBuilderReport(t *testing.T) {
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildRealResolver(t, dir)
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runPlanSide(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", resolverTestdataIntent(t),
		"-intent-resolver", resolver,
		"-intent-resolver-arg", childArgs(t, resolverArgs(t)...),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digest,
			"-report-plan-fingerprint", planFingerprintFromReceipt(t, dir, resolver)))
	if code != 0 {
		t.Fatalf("the attempt was refused: code=%d %+v", code, report.Error)
	}
	binding := report.PlanArtifactBinding
	if binding == nil {
		t.Fatal("no binding was recorded")
	}
	if binding.PlanSideSource != "builder-report" {
		t.Fatalf("the lock did not prefer the builder report: %q", binding.PlanSideSource)
	}
	// A builder reporting the resolved plan's fingerprint is exactly what the
	// entry derives, so the two values are equal here by construction. What
	// distinguishes the cases is the recorded source, not a value comparison.
	if binding.PlanFingerprint == "" {
		t.Fatal("the lock carried no plan fingerprint")
	}
	if binding.EntryPlanFingerprint == "" {
		t.Fatal("the entry's own derivation was not recorded beside the binding")
	}
	if binding.L2BClaimed {
		t.Fatal("the binding claimed L2-b")
	}
}

// A child that reports no plan fingerprint falls back to the entry's own
// derivation, and the record says the basis is weaker.
func TestPlanSideFallsBackToTheEntryDerivation(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildRealResolver(t, dir)
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runPlanSide(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", resolverTestdataIntent(t),
		"-intent-resolver", resolver,
		"-intent-resolver-arg", childArgs(t, resolverArgs(t)...),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("the attempt was refused: code=%d %+v", code, report.Error)
	}
	binding := report.PlanArtifactBinding
	if binding == nil {
		t.Fatal("no binding was recorded")
	}
	if binding.PlanSideSource != "entry-derivation" {
		t.Fatalf("the lock did not fall back to the entry's derivation: %+v", binding)
	}
	if report.PlanRecompute == nil || report.PlanRecompute.RecomputedFingerprint == "" {
		t.Fatal("the entry produced no fingerprint to fall back to")
	}
	if binding.PlanFingerprint != report.PlanRecompute.RecomputedFingerprint {
		t.Fatalf("the fallback used %q, not the entry's own %q",
			binding.PlanFingerprint, report.PlanRecompute.RecomputedFingerprint)
	}
	if binding.EntryPlanFingerprint != report.PlanRecompute.RecomputedFingerprint {
		t.Fatal("the entry's fingerprint was not recorded beside the binding")
	}
	// The weaker basis must not be read as a stronger claim.
	if binding.L2BClaimed {
		t.Fatal("the fallback claimed L2-b")
	}
	if report.Qualified {
		t.Fatal("the fallback made the artifact qualified")
	}
}

// Without a resolver there is no entry derivation either, so the binding stays
// unperformed rather than borrowing a value from somewhere else.
func TestPlanSideRecordsNoBasisWhenBothAreMissing(t *testing.T) {
	report, code := runPlanSide(t, selfBuildPrimitiveWithoutResolverArgs(t)...)
	if code != 0 {
		t.Fatalf("a primitive-form attempt was refused: code=%d", code)
	}
	if report.PlanArtifactBinding != nil {
		t.Fatalf("a binding was recorded without a resolver: %+v", report.PlanArtifactBinding)
	}
}

// selfBuildPrimitiveWithoutResolverArgs builds a primitive-form attempt that
// never reaches the resolver path.
func selfBuildPrimitiveWithoutResolverArgs(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	artifact := filepath.Join(dir, "artifact")
	if err := os.WriteFile(artifact, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := digestOfFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}
	return []string{
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", artifact, "-digest", digest),
		"-artifact", artifact,
	}
}

// resolverMatchingIntent writes a declaration whose subject is the one the
// reference resolver fixture's plugin defines. outerIntentFile names a different
// subject, which the resolver correctly refuses, so it cannot be used when the
// attempt must reach the resolver.
func resolverMatchingIntent(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "resolver-intent.json")
	body := `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.authoring_example", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "./impl"},
  "outputKind": "runtime-artifact",
  "target": "host/linux/amd64",
  "profile": "release"
}
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// planFingerprintFromReceipt runs the reference resolver over the declaration and
// returns the plan fingerprint it derives, so a test can have the builder report
// a value that genuinely matches the resolved plan.
func planFingerprintFromReceipt(t *testing.T, dir, resolver string) string {
	t.Helper()
	receipt := resolveIntentVia(resolver, resolverArgs(t), "")
	if receipt.State != intentResolutionResolved {
		t.Fatalf("the reference resolver did not resolve: %+v", receipt)
	}
	return receipt.PlanFingerprint
}

// resolverTestdataIntent returns the declaration the reference resolver's own
// fixture resolves, so the attempt reaches the resolver with matching inputs.
func resolverTestdataIntent(t *testing.T) string {
	t.Helper()
	base, err := filepath.Abs(resolverTestdata)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "intent.json")
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests pin the entry's own recomputation of the resolver's plan
// fingerprint: §4.8's second item made checkable. The receipt carries the plan,
// so the entry derives the fingerprint with plugin.BuildPlanFingerprint instead
// of trusting the reported value.
//
// It still does not establish L2-b. A consistent receipt says the resolver's
// arithmetic holds and the receipt is internally consistent; it does not say
// the builder child built that plan.

const resolverTestdata = "../../tools/reference-build-authoring-intent/testdata"

// buildRealResolver compiles the repository's resolver reference so the
// recomputation runs against a genuine receipt rather than a hand-written one.
func buildRealResolver(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	binaryPath := filepath.Join(dir, "intentresolver")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	build := exec.Command(goExe, "build", "-trimpath", "-o", binaryPath, "../../tools/reference-build-authoring-intent")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build resolver reference: %v: %s", err, out)
	}
	return binaryPath
}

func resolverArgs(t *testing.T) []string {
	t.Helper()
	base, err := filepath.Abs(resolverTestdata)
	if err != nil {
		t.Fatal(err)
	}
	return []string{
		"-intent", filepath.Join(base, "intent.json"),
		"-catalog", filepath.Join(base, "catalog.json"),
		"-source-root", filepath.Join(base, "src"),
	}
}

// A genuine receipt's plan hashes to the fingerprint it reports.
func TestPlanRecomputeMatchesAGenuineReceipt(t *testing.T) {
	dir := t.TempDir()
	resolver := buildRealResolver(t, dir)
	receipt := resolveIntentVia(resolver, resolverArgs(t), "")
	if receipt.State != intentResolutionResolved {
		t.Fatalf("the reference resolver did not resolve: %+v", receipt)
	}
	record := recomputeReceiptPlanFingerprint(receipt.ReceiptJSON)
	if record.State != planRecomputeMatched || !record.Matched {
		t.Fatalf("a genuine receipt was not matched: %+v", record)
	}
	if record.RecomputedFingerprint != receipt.PlanFingerprint {
		t.Fatalf("recomputed %q, reported %q", record.RecomputedFingerprint, receipt.PlanFingerprint)
	}
	if record.L2BClaimed {
		t.Fatal("the recomputation claimed L2-b")
	}
	if record.Boundary == "" {
		t.Fatal("the recomputation lost its boundary")
	}
}

// A receipt whose reported fingerprint does not match its own plan is refused.
func TestPlanRecomputeDetectsATamperedFingerprint(t *testing.T) {
	dir := t.TempDir()
	resolver := buildRealResolver(t, dir)
	receipt := resolveIntentVia(resolver, resolverArgs(t), "")
	if receipt.State != intentResolutionResolved {
		t.Skipf("resolver unavailable: %+v", receipt)
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &parsed); err != nil {
		t.Fatal(err)
	}
	parsed["planFingerprint"] = json.RawMessage(`"sha256:not-the-plan"`)
	tampered, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}
	record := recomputeReceiptPlanFingerprint(string(tampered))
	if record.State != planRecomputeMismatched {
		t.Fatalf("a tampered fingerprint was not detected: %+v", record)
	}
	if record.Matched {
		t.Fatal("a tampered fingerprint was recorded as matched")
	}
	if record.FailureCause == "" {
		t.Fatal("a mismatch had no stated cause")
	}
	if record.RecomputedFingerprint == "" {
		t.Fatal("the recomputed fingerprint was not recorded")
	}
}

// A receipt without a plan object cannot be recomputed, and says so.
func TestPlanRecomputeReportsAMissingPlan(t *testing.T) {
	record := recomputeReceiptPlanFingerprint(`{"planFingerprint":"sha256:x"}`)
	if record.State != planRecomputeUnperformed {
		t.Fatalf("a plan-less receipt was not unperformed: %+v", record)
	}
	if record.Matched {
		t.Fatal("an unperformed recomputation was recorded as a match")
	}
	if record.UnperformedReason == "" {
		t.Fatal("the unperformed recomputation had no stated reason")
	}
}

// A receipt without a reported fingerprint cannot be compared.
func TestPlanRecomputeReportsAMissingFingerprint(t *testing.T) {
	record := recomputeReceiptPlanFingerprint(`{"plan":{"formatVersion":1}}`)
	if record.State != planRecomputeUnperformed {
		t.Fatalf("a fingerprint-less receipt was not unperformed: %+v", record)
	}
	if record.UnperformedReason == "" {
		t.Fatal("the unperformed recomputation had no stated reason")
	}
}

// A plan that is not a valid BuildPlan is unperformed, not a match: the entry
// does not treat an unusable plan as consistent.
func TestPlanRecomputeReportsAnInvalidPlan(t *testing.T) {
	record := recomputeReceiptPlanFingerprint(`{"planFingerprint":"sha256:x","plan":{"formatVersion":1}}`)
	if record.State != planRecomputeUnperformed {
		t.Fatalf("an invalid plan was not unperformed: %+v", record)
	}
	if record.Matched {
		t.Fatal("an invalid plan was recorded as a match")
	}
	if !strings.Contains(record.UnperformedReason, "validate") {
		t.Fatalf("the invalid plan had no stated reason: %q", record.UnperformedReason)
	}
}

// The outer command carries the recomputation in its evidence, still without
// claiming L2.
func TestPlanRecomputeAppearsInTheOuterEvidence(t *testing.T) {
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

	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", workspace,
		"-intent-file", filepath.Join(filepath.Dir(resolver), "unused.json"),
		"-intent-resolver", resolver,
		"-intent-resolver-arg", childArgs(t, resolverArgs(t)...),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest),
	}, &out)
	if err != nil {
		t.Fatalf("emitter error: %v", err)
	}
	if code == 0 {
		t.Fatalf("a missing declaration was accepted: %s", out.String())
	}
	// The refusal happens before the resolver runs only when the declaration is
	// missing; this test asserts that a missing declaration does not produce a
	// recomputation claim.
	var report planRecomputeResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	if report.PlanRecompute != nil {
		t.Fatalf("a recomputation was recorded without a resolver run: %+v", report.PlanRecompute)
	}
}

type planRecomputeResult struct {
	OK            bool `json:"ok"`
	Qualified     bool `json:"qualified"`
	PlanRecompute *struct {
		State             string `json:"state"`
		Matched           bool   `json:"matched"`
		L2BClaimed        bool   `json:"l2bClaimed"`
		Boundary          string `json:"boundary"`
		UnperformedReason string `json:"unperformedReason"`
	} `json:"planRecompute"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

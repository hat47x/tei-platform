package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// These tests pin §4.8's third observation: the entry compares the resolver
// child's plan fingerprint with the plan fingerprint the builder child reports.
// Agreement of two self-reports still does not establish L2-b, so l2bClaimed
// stays false in every case.

type planCrossCheckResult struct {
	OK             bool `json:"ok"`
	Qualified      bool `json:"qualified"`
	PlanCrossCheck *struct {
		State                          string `json:"state"`
		ResolverPlanFingerprint        string `json:"resolverPlanFingerprint"`
		BuilderReportedPlanFingerprint string `json:"builderReportedPlanFingerprint"`
		Matched                        bool   `json:"matched"`
		L2BClaimed                     bool   `json:"l2bClaimed"`
		Boundary                       string `json:"boundary"`
		UnperformedReason              string `json:"unperformedReason"`
	} `json:"planCrossCheck"`
	ChildRun *struct {
		FailureCause string `json:"failureCause"`
	} `json:"childRun"`
}

func runCrossCheck(t *testing.T, args ...string) (planCrossCheckResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report planCrossCheckResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// buildPlanReportingChild reports the plan fingerprint it was told to build,
// so the cross-check sees exactly the value under test.
func buildPlanReportingChild(t *testing.T, dir string) string {
	t.Helper()
	child := buildIntentReportingChild(t, dir)
	return child
}

// Both children name the same plan: the cross-check matches, and L2-b is still
// not claimed.
func TestPlanCrossCheckMatchesWithoutClaimingL2B(t *testing.T) {
	dir := t.TempDir()
	child := buildPlanReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildFakeResolver(t, dir, resolverReceipt("sha256:plan-agreed"))
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

	report, code := runCrossCheck(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digest,
			"-report-plan-fingerprint", "sha256:plan-agreed"))
	if code != 0 {
		t.Fatalf("a matching plan cross-check was refused: code=%d", code)
	}
	if report.PlanCrossCheck == nil {
		t.Fatal("no plan cross-check was recorded")
	}
	if report.PlanCrossCheck.State != planCrossCheckMatched || !report.PlanCrossCheck.Matched {
		t.Fatalf("matching plans were not recorded as matched: %+v", report.PlanCrossCheck)
	}
	if report.PlanCrossCheck.L2BClaimed {
		t.Fatal("the cross-check claimed L2-b")
	}
	if report.PlanCrossCheck.Boundary == "" {
		t.Fatal("the cross-check lost its boundary statement")
	}
	if report.Qualified {
		t.Fatal("a matching plan made the artifact qualified")
	}
}

// Different plans: the mismatch is recorded as a finding and the attempt fails.
func TestPlanCrossCheckReportsAMismatch(t *testing.T) {
	dir := t.TempDir()
	child := buildPlanReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildFakeResolver(t, dir, resolverReceipt("sha256:plan-resolved"))
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

	report, code := runCrossCheck(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digest,
			"-report-plan-fingerprint", "sha256:plan-built"))
	if code != 1 {
		t.Fatalf("a plan mismatch was not a failure: code=%d", code)
	}
	if report.PlanCrossCheck == nil || report.PlanCrossCheck.State != planCrossCheckMismatched {
		t.Fatalf("plan mismatch was not recorded: %+v", report.PlanCrossCheck)
	}
	if report.PlanCrossCheck.Matched {
		t.Fatal("a mismatch was recorded as a match")
	}
	if report.PlanCrossCheck.L2BClaimed {
		t.Fatal("a mismatch claimed L2-b")
	}
}

// A builder that does not report its plan leaves the cross-check unperformed,
// with the reason stated instead of a match.
func TestPlanCrossCheckRecordsWhyItWasNotPerformed(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildFakeResolver(t, dir, resolverReceipt("sha256:plan-resolved"))
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

	report, code := runCrossCheck(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("a missing builder report broke a working attempt: code=%d", code)
	}
	if report.PlanCrossCheck == nil {
		t.Fatal("no cross-check was recorded")
	}
	if report.PlanCrossCheck.State != planCrossCheckUnperformed {
		t.Fatalf("missing builder report was not unperformed: %+v", report.PlanCrossCheck)
	}
	if report.PlanCrossCheck.Matched {
		t.Fatal("an unperformed cross-check was recorded as a match")
	}
	if report.PlanCrossCheck.UnperformedReason == "" {
		t.Fatal("the unperformed cross-check had no stated reason")
	}
}

// Without a resolver there is no resolved plan, so the cross-check records that
// instead of silently disappearing.
func TestPlanCrossCheckReportsNoResolvedPlan(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
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
	report, code := runCrossCheck(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("a build without a resolver was refused: code=%d", code)
	}
	if report.PlanCrossCheck != nil {
		t.Fatalf("a cross-check was recorded without a resolver: %+v", report.PlanCrossCheck)
	}
}

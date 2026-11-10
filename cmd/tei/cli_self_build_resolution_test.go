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

// These tests pin the §4.7 I-2 connection: intent resolution is delegated to an
// exact resolver child, and the entry records the receipt it received without
// claiming L2 (§4.8 splits the claim into L2-a hand-off and L2-b interpretation,
// and both stay false here).

type planReceiptResult struct {
	SchemaVersion string `json:"schemaVersion"`
	OK            bool   `json:"ok"`
	Qualified     bool   `json:"qualified"`
	Error         *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	PlanReceipt *struct {
		State               string `json:"state"`
		Authority           string `json:"authority"`
		PlanFingerprint     string `json:"planFingerprint"`
		ExpectedPlan        string `json:"expectedPlanFingerprint"`
		PlanMatchesExpected bool   `json:"planMatchesExpected"`
		L2Claimed           bool   `json:"l2Claimed"`
		ResolverDigest      string `json:"resolverDigest"`
		FailureCause        string `json:"failureCause"`
	} `json:"planReceipt"`
	ChildRun *struct {
		CrossCheckOK bool `json:"artifactCrossCheckMatched"`
	} `json:"childRun"`
}

func runOuterWithResolver(t *testing.T, args ...string) (planReceiptResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report planReceiptResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// buildFakeResolver writes a resolver whose receipt is controlled by the
// arguments, so each refusal condition can be exercised without a real
// resolver's catalogs.
func buildFakeResolver(t *testing.T, dir, receiptJSON string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "fakeintentresolver.go")
	body := "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\n" +
		"const receipt = `" + receiptJSON + "`\n\n" +
		"func main() {\n\tfmt.Println(receipt)\n\tos.Exit(0)\n}\n"
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(dir, "fakeintentresolver")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	build := exec.Command(goExe, "build", "-trimpath", "-o", binaryPath, source)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fake resolver: %v: %s", err, out)
	}
	return binaryPath
}

func resolverReceipt(planFingerprint string) string {
	return `{"formatVersion":1,"authority":"derived-cache","intentFingerprint":"sha256:intent","planFingerprint":"` +
		planFingerprint + `","sourceClosureFingerprint":"sha256:closure"}`
}

// A resolver receipt with a matching declared plan lets the build proceed, and
// the entry still does not claim L2.
func TestSelfBuildOuterAcceptsAResolvedPlan(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
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

	report, code := runOuterWithResolver(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-expected-plan-fingerprint", "sha256:plan-agreed",
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("a matching resolved plan was refused: code=%d %+v", code, report.Error)
	}
	if report.PlanReceipt == nil {
		t.Fatal("the resolver receipt was not recorded")
	}
	if report.PlanReceipt.State != intentResolutionResolved {
		t.Fatalf("unexpected receipt state: %s", report.PlanReceipt.State)
	}
	if report.PlanReceipt.Authority != derivedCacheAuthority {
		t.Fatalf("authority was not recorded: %q", report.PlanReceipt.Authority)
	}
	if !report.PlanReceipt.PlanMatchesExpected {
		t.Fatal("a matching plan was not recorded as matching")
	}
	if report.PlanReceipt.ResolverDigest == "" {
		t.Fatal("the resolver's own digest was not recorded")
	}
	// The claim boundary: the entry never asserts L2.
	if report.PlanReceipt.L2Claimed {
		t.Fatal("the entry claimed L2")
	}
	if report.Qualified {
		t.Fatal("a resolved plan made the artifact qualified")
	}
}

// A plan that differs from the caller-declared expectation stops before the
// builder starts.
func TestSelfBuildOuterRefusesAMismatchedPlan(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildFakeResolver(t, dir, resolverReceipt("sha256:plan-actual"))
	report, code := runOuterWithResolver(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-expected-plan-fingerprint", "sha256:plan-declared",
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("a mismatched plan was not refused: code=%d", code)
	}
	if report.Error == nil || report.Error.Code != selfBuildCodePlanMismatch {
		t.Fatalf("plan mismatch had no stated code: %+v", report.Error)
	}
	if report.ChildRun != nil {
		t.Fatal("a plan mismatch started the builder")
	}
	if report.PlanReceipt == nil || report.PlanReceipt.PlanMatchesExpected {
		t.Fatalf("plan mismatch was recorded as a match: %+v", report.PlanReceipt)
	}
}

// A receipt that does not identify itself as a derived cache is refused: the
// entry does not read unknown output as a resolved plan.
func TestSelfBuildOuterRefusesAReceiptWithoutTheKnownAuthority(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	resolver := buildFakeResolver(t, dir,
		`{"formatVersion":1,"authority":"canonical","intentFingerprint":"sha256:intent","planFingerprint":"sha256:plan"}`)
	report, code := runOuterWithResolver(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("an unknown-authority receipt was accepted: code=%d", code)
	}
	if report.Error == nil || report.Error.Code != selfBuildCodeIntentResolutionFailed {
		t.Fatalf("unknown authority had no stated code: %+v", report.Error)
	}
	if report.PlanReceipt == nil || !strings.Contains(report.PlanReceipt.FailureCause, "derived-cache") {
		t.Fatalf("unknown authority had no stated cause: %+v", report.PlanReceipt)
	}
}

// A resolver that fails stops the attempt, and the failure is recorded rather
// than silently skipped.
func TestSelfBuildOuterRefusesAFailedResolver(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "failingresolver.go")
	if err := os.WriteFile(source, []byte("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\tfmt.Fprintln(os.Stderr, \"resolver refused\")\n\tos.Exit(3)\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := filepath.Join(dir, "failingresolver")
	if runtime.GOOS == "windows" {
		resolver += ".exe"
	}
	build := exec.Command(goExe, "build", "-trimpath", "-o", resolver, source)
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failing resolver: %v: %s", err, out)
	}

	report, code := runOuterWithResolver(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-intent-resolver", resolver,
		"-state", filepath.Join(dir, "state"),
		"-output", filepath.Join(dir, "out"))
	if code != 2 {
		t.Fatalf("a failed resolver was not refused: code=%d", code)
	}
	if report.PlanReceipt == nil || report.PlanReceipt.State != intentResolutionRefused {
		t.Fatalf("failed resolver was not recorded as refused: %+v", report.PlanReceipt)
	}
	if !strings.Contains(report.PlanReceipt.FailureCause, "exited with 3") {
		t.Fatalf("failed resolver cause was not recorded: %q", report.PlanReceipt.FailureCause)
	}
	if report.ChildRun != nil {
		t.Fatal("a failed resolver started the builder")
	}
}

// Without a resolver the entry records nothing about a plan, and it must not
// present the absence as a resolution.
func TestSelfBuildOuterRecordsNoPlanWithoutAResolver(t *testing.T) {
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
	report, code := runOuterWithResolver(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("a build without a resolver was refused: code=%d %+v", code, report.Error)
	}
	if report.PlanReceipt != nil {
		t.Fatalf("a plan receipt was recorded without a resolver: %+v", report.PlanReceipt)
	}
	if report.Qualified {
		t.Fatal("an unresolvable declaration made the artifact qualified")
	}
}

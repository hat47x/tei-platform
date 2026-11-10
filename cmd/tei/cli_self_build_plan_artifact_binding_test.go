package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// These tests pin §4.8(c): the entry binds the builder's reported plan
// fingerprint to the artifact digest as one validated BuildArtifactLock, and
// confirms that artifact digest against the bytes on disk. l2bClaimed stays
// false in every case.

// The child writes the artifact and reports both its digest and a plan
// fingerprint, so the entry has real values to bind.
const bindingChildSource = `package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	artifact, planFP := "", ""
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-artifact-out":
			artifact = os.Args[i+1]
			i++
		case "-report-plan-fingerprint":
			planFP = os.Args[i+1]
			i++
		}
	}
	if artifact == "" {
		fmt.Fprintln(os.Stderr, "missing -artifact-out")
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	body := []byte("binding-artifact")
	if err := os.WriteFile(artifact, body, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	sum := sha256.Sum256(body)
	fmt.Printf("{\"schemaVersion\":\"tei.probe.builder/v0\",\"artifactDigest\":\"sha256:%s\",\"planFingerprint\":%q}\n",
		hex.EncodeToString(sum[:]), planFP)
}
`

func buildBindingChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "bindingchild.go")
	if err := os.WriteFile(source, []byte(bindingChildSource), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "bindingchild")
	if runtime.GOOS == "windows" {
		child += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", child, source)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build child fixture: %v: %s", err, out)
	}
	return child
}

type bindingResult struct {
	OK                  bool `json:"ok"`
	Qualified           bool `json:"qualified"`
	PlanArtifactBinding *struct {
		State                string `json:"state"`
		PlanFingerprint      string `json:"planFingerprint"`
		ArtifactDigest       string `json:"artifactDigest"`
		ArtifactDigestOnDisk string `json:"artifactDigestOnDisk"`
		LockFingerprint      string `json:"lockFingerprint"`
		Matched              bool   `json:"matched"`
		L2BClaimed           bool   `json:"l2bClaimed"`
		Boundary             string `json:"boundary"`
		UnperformedReason    string `json:"unperformedReason"`
		FailureCause         string `json:"failureCause"`
	} `json:"planArtifactBinding"`
	PlanReceipt *struct {
		State           string `json:"state"`
		PlanFingerprint string `json:"planFingerprint"`
	} `json:"planReceipt"`
	PlanCrossCheck *struct {
		State string `json:"state"`
	} `json:"planCrossCheck"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// resolverIntentPath is the declaration the reference resolver resolves. The
// entry and the resolver must read the same declaration, otherwise the resolver
// refuses before any binding can be formed.
func resolverIntentPath(t *testing.T) string {
	t.Helper()
	base, err := filepath.Abs(resolverTestdata)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "intent.json")
}

func runBindingShape(t *testing.T, args ...string) (bindingResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report bindingResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// A builder that reports the resolver's plan and produces the artifact it
// reports makes the binding `bound`. The resolver is the repository's real
// reference, so the plan identity comes from a genuine receipt.
func TestPlanArtifactBindingBindsAMatchingReport(t *testing.T) {
	dir := t.TempDir()
	resolver := buildRealResolver(t, dir)
	resolved := resolveIntentVia(resolver, resolverArgs(t), "")
	if resolved.State != intentResolutionResolved {
		t.Fatalf("the reference resolver did not resolve: %+v", resolved)
	}
	child := buildBindingChild(t, dir)
	output := filepath.Join(dir, "artifact")
	report, code := runBindingShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, dir),
		"-intent-file", resolverIntentPath(t),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-report-plan-fingerprint", resolved.PlanFingerprint),
		"-intent-resolver", resolver,
		"-intent-resolver-arg", childArgs(t, resolverArgs(t)...),
	)
	if code != 0 {
		t.Fatalf("binding did not complete: code=%d %+v", code, report.Error)
	}
	binding := report.PlanArtifactBinding
	if binding == nil {
		t.Fatal("no plan-to-artifact binding was recorded")
	}
	if binding.State != planArtifactBound || !binding.Matched {
		t.Fatalf("expected bound, got %+v", binding)
	}
	if binding.LockFingerprint == "" {
		t.Fatal("the lock fingerprint was not recorded")
	}
	if binding.ArtifactDigest == "" || binding.ArtifactDigestOnDisk == "" {
		t.Fatalf("artifact digests were not both recorded: %+v", binding)
	}
	if binding.ArtifactDigest != binding.ArtifactDigestOnDisk {
		t.Fatal("the reported artifact digest was not confirmed against disk")
	}
	if binding.L2BClaimed {
		t.Fatal("L2-b was claimed")
	}
	if binding.Boundary == "" {
		t.Fatal("binding lost its boundary statement")
	}
	if report.Qualified {
		t.Fatal("qualified was claimed")
	}
}

// A child that reports a plan fingerprint the resolver never produced would
// bind the artifact to the wrong plan, so the binding refuses.
func TestPlanArtifactBindingRejectsAMismatchedPlanFingerprint(t *testing.T) {
	dir := t.TempDir()
	resolver := buildRealResolver(t, dir)
	child := buildBindingChild(t, dir)
	output := filepath.Join(dir, "artifact")
	report, code := runBindingShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, dir),
		"-intent-file", resolverIntentPath(t),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-report-plan-fingerprint", "sha256:not-the-resolved-plan"),
		"-intent-resolver", resolver,
		"-intent-resolver-arg", childArgs(t, resolverArgs(t)...),
	)
	binding := report.PlanArtifactBinding
	if binding == nil {
		t.Fatal("no plan-to-artifact binding was recorded")
	}
	if binding.State != planArtifactMismatched {
		t.Fatalf("expected mismatched, got %+v", binding)
	}
	if binding.L2BClaimed {
		t.Fatal("L2-b was claimed")
	}
	if binding.FailureCause == "" {
		t.Fatal("mismatch had no stated cause")
	}
	if code == 0 {
		t.Fatal("a mismatched plan binding still exited 0")
	}
}

// Without a resolver there is no plan identity, so no binding is written at all
// rather than an unexplained empty one.
func TestPlanArtifactBindingIsAbsentWithoutAResolver(t *testing.T) {
	dir := t.TempDir()
	child := buildBindingChild(t, dir)
	output := filepath.Join(dir, "artifact")
	report, _ := runBindingShape(t,
		"-child", child,
		"-workspace", outerWorkspace(t, dir),
		"-intent-file", resolverIntentPath(t),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-report-plan-fingerprint", "sha256:any"),
	)
	if report.PlanArtifactBinding != nil {
		t.Fatal("a binding was recorded without a resolver")
	}
	if report.PlanCrossCheck != nil {
		t.Fatal("a cross-check was recorded without a resolver")
	}
	if report.PlanReceipt != nil {
		t.Fatal("a receipt was recorded without a resolver")
	}
}

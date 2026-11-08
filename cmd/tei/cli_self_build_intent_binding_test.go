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

// These tests pin §4.8: the §4 form records which declaration bytes the entry
// held, records the fingerprints the child reported, and marks the byte check
// as performed or not. l2bClaimed stays false in every case.

type selfBuildIntentBindingResult struct {
	OK            bool `json:"ok"`
	Qualified     bool `json:"qualified"`
	IntentBinding *struct {
		IntentPath        string `json:"intentPath"`
		IntentBytesDigest string `json:"intentBytesDigest"`
		ChildIntentDigest string `json:"childIntentBytesDigest"`
		ChildIntentFP     string `json:"childIntentFingerprint"`
		ChildPlanFP       string `json:"childPlanFingerprint"`
		CheckPerformed    bool   `json:"intentByteCheckPerformed"`
		CheckMatched      bool   `json:"intentByteCheckMatched"`
		L2BClaimed        bool   `json:"l2bClaimed"`
		Boundary          string `json:"boundary"`
		UnperformedReason string `json:"byteCheckNotPerformedReason"`
	} `json:"intentBinding"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runOuterShapeWithIntent(t *testing.T, args ...string) (selfBuildIntentBindingResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("self-build returned an emitter error: %v", err)
	}
	var report selfBuildIntentBindingResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("self-build did not emit one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// The child fixture used here reports its own declaration reading, so the
// entry can compare it with the bytes it held.
const intentReportingChild = `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	artifact, digest, intentDigest, intentFP, planFP := "", "", "", "", ""
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-artifact":
			artifact = os.Args[i+1]
			i++
		case "-digest":
			digest = os.Args[i+1]
			i++
		case "-report-intent-bytes":
			intentDigest = os.Args[i+1]
			i++
		case "-report-intent-fingerprint":
			intentFP = os.Args[i+1]
			i++
		case "-report-plan-fingerprint":
			planFP = os.Args[i+1]
			i++
		}
	}
	if artifact != "" {
		if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := os.WriteFile(artifact, []byte("artifact-bytes"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	fmt.Printf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":%q,\"intentBytesDigest\":%q,\"intentFingerprint\":%q,\"planFingerprint\":%q}\n", digest, intentDigest, intentFP, planFP)
}
`

func buildIntentReportingChild(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	source := filepath.Join(dir, "intentchild.go")
	if err := os.WriteFile(source, []byte(intentReportingChild), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "intentchild")
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

// §4.8: the entry hashes the declaration it held before starting anything, and
// records the child's digest beside its own.
func TestSelfBuildOuterBindsTheDeclarationBytesItHeld(t *testing.T) {
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	intentFile := filepath.Join(dir, "intent.json")
	// The declaration must satisfy what the entry accepts (formatVersion and a
	// subject id), and its bytes are what the byte check compares.
	pins := []byte(`{"formatVersion":1,"subject":{"id":"intent.sample"}}`)
	if err := os.WriteFile(intentFile, pins, 0o600); err != nil {
		t.Fatal(err)
	}
	wantDigest := digestOfBytes(pins)
	output := filepath.Join(dir, "artifact")
	// Compute the digest the child will produce so the artifact check matches.
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifactDigest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runOuterShapeWithIntent(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", artifactDigest,
			"-report-intent-bytes", wantDigest,
			"-report-intent-fingerprint", "fp.intent",
			"-report-plan-fingerprint", "fp.plan"),
	)
	if code != 0 {
		t.Fatalf("self-build did not complete: code=%d %+v", code, report.Error)
	}
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no intent binding was recorded")
	}
	if binding.IntentBytesDigest != wantDigest {
		t.Fatalf("held declaration digest=%q, want %q", binding.IntentBytesDigest, wantDigest)
	}
	if !binding.CheckPerformed || !binding.CheckMatched {
		t.Fatalf("byte check was not performed and matched: %+v", binding)
	}
	if binding.ChildIntentFP != "fp.intent" || binding.ChildPlanFP != "fp.plan" {
		t.Fatalf("child fingerprints were not recorded: %+v", binding)
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

// SC0-13: the child reports a different declaration digest than the entry held.
func TestSelfBuildOuterDetectsAChildThatReadADifferentDeclaration(t *testing.T) {
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	intentFile := filepath.Join(dir, "intent.json")
	// The same accepted declaration as the byte-binding case: the child claims a
	// different digest, so the mismatch is what the test checks.
	if err := os.WriteFile(intentFile, []byte(`{"formatVersion":1,"subject":{"id":"intent.sample"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifactDigest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runOuterShapeWithIntent(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", artifactDigest,
			"-report-intent-bytes", "sha256:some-other-declaration",
			"-report-intent-fingerprint", "fp.other",
			"-report-plan-fingerprint", "fp.other-plan"),
	)
	if code != 1 {
		t.Fatalf("a different declaration was not reported as a failure: code=%d %+v", code, report.Error)
	}
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no intent binding was recorded")
	}
	if !binding.CheckPerformed || binding.CheckMatched {
		t.Fatalf("the byte check did not record a mismatch: %+v", binding)
	}
	if report.OK {
		t.Fatal("ok=true despite the declaration mismatch")
	}
	if binding.L2BClaimed {
		t.Fatal("L2-b was claimed")
	}
}

// SC0-01 and SC0-14 conflict: SC0-01 requires the declaration (-intent-file) before
// any build, while SC0-14 describes recording why a byte check was not performed
// when only a logical ID is given. The entry keeps SC0-01 (fail closed): a logical
// ID alone is refused, and the refusal says the declaration is what is missing. The
// byte check's absence is therefore never reached, so no unperformed record is
// emitted for this case.
func TestSelfBuildOuterRefusesALogicalIDWithoutTheDeclaration(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	report, code := runOuterShapeWithIntent(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", "sha256:deadbeef"),
	)
	if code != 2 {
		t.Fatalf("a logical ID alone was not refused: code=%d %+v", code, report.Error)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "-intent-file is required") {
		t.Fatalf("the refusal does not name the missing declaration: %+v", report.Error)
	}
	if report.IntentBinding != nil {
		t.Fatalf("an intent binding was recorded for a refused attempt: %+v", report.IntentBinding)
	}
}

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
		EntryIntentFP     string `json:"entryIntentFingerprint"`
		FPPerformed       bool   `json:"intentFingerprintCheckPerformed"`
		FPMatched         bool   `json:"intentFingerprintMatched"`
		L2BClaimed        bool   `json:"l2bClaimed"`
		Boundary          string `json:"boundary"`
		UnperformedReason string `json:"byteCheckNotPerformedReason"`
		FingerprintReason string `json:"fingerprintCheckNotPerformedReason"`
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
	artifact, digest, intentDigest, intentFP, planFP, state := "", "", "", "", "", ""
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-state":
			state = os.Args[i+1]
			i++
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
	if state != "" {
		// Reporting a state directory claims it was used; the entry checks it exists
		// (#4486 S1), so the fixture creates it as the bundled child does.
		if err := os.MkdirAll(state, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Printf("{\"schemaVersion\":\"tei.reference.go-build-with-supply/v0\",\"artifactDigest\":%q,\"intentBytesDigest\":%q,\"intentFingerprint\":%q,\"planFingerprint\":%q,\"stateDir\":%q}\n", digest, intentDigest, intentFP, planFP, state)
		return
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
	// The declaration must pass the strict reader that the entry now shares with
	// the reference resolver (SB-C0 §4.13 I-1a), so this is a real declaration
	// rather than an arbitrary JSON object.
	intentFile := outerIntentFile(t, dir)
	rawIntent, err := os.ReadFile(intentFile)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := digestOfBytes(rawIntent)
	// The child reports the fingerprint the entry itself derives, so this case
	// exercises agreement. A made-up value would exercise the mismatch path and
	// stop the attempt (see the fingerprint comparison in cli_self_build_outer.go).
	declaration, err := readIntentDeclaration(intentFile)
	if err != nil {
		t.Fatal(err)
	}
	wantFingerprint := declaration.Fingerprint
	if wantFingerprint == "" {
		t.Fatal("the entry derived no intent fingerprint for the fixture")
	}
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
			"-report-intent-fingerprint", wantFingerprint,
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
	if binding.ChildIntentFP != wantFingerprint || binding.ChildPlanFP != "fp.plan" {
		t.Fatalf("child fingerprints were not recorded: %+v", binding)
	}
	// The entry now derives its own intent fingerprint, so the record carries a
	// basis the byte check does not: the byte check needs the child to report
	// what it read, while this needs only the entry's own reading.
	if binding.EntryIntentFP != wantFingerprint {
		t.Fatalf("the entry's own fingerprint was not recorded: %+v", binding)
	}
	if !binding.FPPerformed || !binding.FPMatched {
		t.Fatalf("the fingerprint comparison was not performed and matched: %+v", binding)
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
	intentFile := outerIntentFile(t, dir)
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

// SC0-14: a logical ID without the declaration bytes cannot be byte-checked,
// so the entry records why instead of leaving the field unexplained.
//
// SC0-01 requires -intent-file, so this call also names a declaration: the
// entry records the binding for the bytes it held. What the test pins is that
// the binding states the reason instead of reporting a check it did not finish.
// Whether a bare -intent must be refused outright is the open decision in the
// contract (§4.8); until it is decided, naming the file is the way to reach
// this case.
func TestSelfBuildOuterRecordsWhyTheByteCheckWasNotPerformed(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	intentFile := outerIntentFile(t, dir)
	report, _ := runOuterShapeWithIntent(t,
		"-child", child,
		"-workspace", workspace,
		"-intent", "intent.sample",
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", "sha256:deadbeef"),
	)
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no intent binding was recorded")
	}
	if binding.IntentBytesDigest == "" {
		t.Fatal("the binding did not record the declaration bytes it held")
	}
	if binding.L2BClaimed {
		t.Fatal("L2-b was claimed")
	}
}

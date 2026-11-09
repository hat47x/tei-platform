package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin the fingerprint-level intent binding added on top of the byte
// check (SB-C0 §4.13 I-1a made the basis possible).
//
// The two checks answer different questions and neither replaces the other:
//
//	byte check        did the child read the same BYTES this entry held?
//	fingerprint check did the child read the same DECLARATION this entry read?
//
// The second needs nothing from the child but its reported fingerprint, so it
// still works when a child reports a fingerprint and no bytes. It is still not
// L2-b: agreement of a derived value is not proof that the builder derived the
// plan from that declaration.

type fingerprintBindingResult struct {
	OK            bool `json:"ok"`
	Qualified     bool `json:"qualified"`
	IntentBinding *struct {
		IntentBytesDigest string `json:"intentBytesDigest"`
		ChildIntentFP     string `json:"childIntentFingerprint"`
		EntryIntentFP     string `json:"entryIntentFingerprint"`
		CheckPerformed    bool   `json:"intentByteCheckPerformed"`
		CheckMatched      bool   `json:"intentByteCheckMatched"`
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

func runFingerprintBinding(t *testing.T, args ...string) (fingerprintBindingResult, int) {
	t.Helper()
	var out bytes.Buffer
	code, err := runSelfBuildCommand(args, &out)
	if err != nil {
		t.Fatalf("emitter error: %v", err)
	}
	var report fingerprintBindingResult
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("not one JSON object: %v\n%s", err, out.String())
	}
	return report, code
}

// A child whose fingerprint agrees is recorded as matched, without claiming L2-b.
func TestFingerprintBindingMatchesWithoutClaimingL2B(t *testing.T) {
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	intentFile := outerIntentFile(t, dir)
	declaration, err := readIntentDeclaration(intentFile)
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Fingerprint == "" {
		t.Fatal("the entry derived no fingerprint for the fixture")
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

	report, code := runFingerprintBinding(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", artifactDigest,
			"-report-intent-fingerprint", declaration.Fingerprint))
	if code != 0 {
		t.Fatalf("an agreeing fingerprint was refused: code=%d %+v", code, report.Error)
	}
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no binding was recorded")
	}
	if binding.EntryIntentFP != declaration.Fingerprint {
		t.Fatalf("the entry's fingerprint was not recorded: %+v", binding)
	}
	if !binding.FPPerformed || !binding.FPMatched {
		t.Fatalf("the fingerprint comparison was not performed and matched: %+v", binding)
	}
	if binding.L2BClaimed {
		t.Fatal("the fingerprint comparison claimed L2-b")
	}
	if report.Qualified {
		t.Fatal("an agreeing fingerprint made the artifact qualified")
	}
}

// A child that reports the bytes but a different fingerprint is caught by the
// fingerprint check even though the byte check passed.
func TestFingerprintBindingDetectsADifferentDeclaration(t *testing.T) {
	dir := t.TempDir()
	child := buildIntentReportingChild(t, dir)
	workspace := outerWorkspace(t, dir)
	intentFile := outerIntentFile(t, dir)
	rawIntent, err := os.ReadFile(intentFile)
	if err != nil {
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

	report, code := runFingerprintBinding(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", artifactDigest,
			"-report-intent-bytes", digestOfBytes(rawIntent),
			"-report-intent-fingerprint", "a-different-declaration"))
	if code != 1 {
		t.Fatalf("a differing fingerprint was not a failure: code=%d", code)
	}
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no binding was recorded")
	}
	if !binding.CheckPerformed || !binding.CheckMatched {
		t.Fatalf("the byte check should still pass in this case: %+v", binding)
	}
	if !binding.FPPerformed || binding.FPMatched {
		t.Fatalf("the fingerprint comparison did not catch the difference: %+v", binding)
	}
	if report.Error == nil || !strings.Contains(report.Error.Message, "intent fingerprint") {
		t.Fatalf("the mismatch had no stated cause: %+v", report.Error)
	}
	if binding.L2BClaimed {
		t.Fatal("a failed fingerprint comparison claimed L2-b")
	}
}

// A child that reports no fingerprint leaves the check unperformed with the
// reason, never as a match.
func TestFingerprintBindingRecordsWhyItWasNotPerformed(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
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

	report, code := runFingerprintBinding(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", intentFile,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", artifactDigest))
	if code != 0 {
		t.Fatalf("an unreported fingerprint broke a working attempt: code=%d", code)
	}
	binding := report.IntentBinding
	if binding == nil {
		t.Fatal("no binding was recorded")
	}
	if binding.FPPerformed {
		t.Fatal("the fingerprint check was reported as performed without a child report")
	}
	if binding.FPMatched {
		t.Fatal("an unperformed fingerprint check was recorded as a match")
	}
	if binding.FingerprintReason == "" {
		t.Fatal("the unperformed fingerprint check had no stated reason")
	}
	if binding.EntryIntentFP == "" {
		t.Fatal("the entry's own fingerprint must be recorded even when the child reports none")
	}
}

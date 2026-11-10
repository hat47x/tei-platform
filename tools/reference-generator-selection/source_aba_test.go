package generatorselection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// EB-02 (#2595): the Python candidate is run under a test-only wrapper that
// swaps the script bytes (D1 -> D2) after the pre-case digest check and before
// the interpreter starts, then restores D1 before the post-case digest check.
// The pre/post digests are both D1, so the digest comparison cannot see the swap.
//
// This file characterizes the current behavior. The wrapper is a test-only
// execution path and does not prove how an unmodified run behaves. The
// assertions on the Passed value and on Execution.SourceDigest record what the
// product reports today; they are expected to flip when EB-04 (an isolated
// execution snapshot, or a fail-closed claim) lands.

type abaObservation struct {
	passed          bool
	rejectReason    string
	candidateDigest string
	executedDigests []string // digest of the script bytes the wrapper handed to the interpreter
	recordedDigests []string // Execution.SourceDigest per case
	caseCount       int
}

func runScriptUnderWrapper(t *testing.T, mode string) (d1, d2 string, obs abaObservation) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the wrapper is a POSIX shell script")
	}
	if _, rt := ResolveScriptRuntime(LanguagePython); !rt.Available {
		t.Skip("python3 is unavailable")
	}
	for _, tool := range []string{"sh", "sha256sum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is unavailable", tool)
		}
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "python-template", "1", filepath.Join(candidates, "python-template"))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(generated.Path, generated.Manifest.Source)
	d1Bytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	// D2 differs in bytes and keeps the observable output, so a passing run
	// cannot be told apart from D1 by the result alone.
	d2Bytes := append(append([]byte(nil), d1Bytes...), []byte("\n# D2: bytes differ from the declared source\n")...)
	d1, d2 = digest(d1Bytes), digest(d2Bytes)
	if d1 == d2 {
		t.Fatal("D1 and D2 must differ")
	}

	work := t.TempDir()
	d2Path := filepath.Join(work, "d2.py")
	d1Path := filepath.Join(work, "d1.py")
	logPath := filepath.Join(work, "executed.log")
	for path, data := range map[string][]byte{d1Path: d1Bytes, d2Path: d2Bytes} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var body string
	switch mode {
	case "D1 fixed":
		body = ""
	case "D2 fixed":
		body = fmt.Sprintf("cp %q \"$2\"\n", d2Path)
	case "D1-D2-D1":
		body = fmt.Sprintf("cp %q \"$2\"\n", d2Path)
	default:
		t.Fatalf("unknown mode %q", mode)
	}
	restore := ""
	if mode == "D1-D2-D1" {
		restore = fmt.Sprintf("cp %q \"$2\"\n", d1Path)
	}
	wrapper := "#!/bin/sh\n" + body +
		fmt.Sprintf("sha256sum \"$2\" | cut -d' ' -f1 >> %q\n", logPath) +
		"\"$@\"\nrc=$?\n" + restore + "exit $rc\n"
	wrapperPath := filepath.Join(work, "wrapper.sh")
	if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o700); err != nil {
		t.Fatal(err)
	}
	if mode == "D2 fixed" {
		// the declared digest must match the bytes on disk at check time, so the
		// control declares D2 and ships D2.
		if err := os.WriteFile(sourcePath, d2Bytes, 0o600); err != nil {
			t.Fatal(err)
		}
		writeManifest(t, generated.Path, "python-template", generated.Manifest.Source, d2Bytes)
	}

	receipt, err := SelectForTargetWithExecution(context.Background(), workspaceDir, contract, fixtures, candidates, "", []string{"python-template"}, ExecutionOptions{CommandPrefix: []string{"sh", wrapperPath}})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Candidates) != 1 {
		t.Fatalf("candidates=%+v", receipt.Candidates)
	}
	c := receipt.Candidates[0]
	obs = abaObservation{passed: c.Passed, rejectReason: c.RejectReason, candidateDigest: c.SourceDigest, caseCount: len(c.Results)}
	for _, r := range c.Results {
		if r.Execution != nil {
			obs.recordedDigests = append(obs.recordedDigests, r.Execution.SourceDigest)
		}
	}
	if raw, err := os.ReadFile(logPath); err == nil {
		for _, line := range strings.Fields(string(raw)) {
			obs.executedDigests = append(obs.executedDigests, "sha256:"+line)
		}
	}
	return d1, d2, obs
}

func allEqual(values []string, want string) bool {
	if len(values) == 0 {
		return false
	}
	for _, v := range values {
		if v != want {
			return false
		}
	}
	return true
}

func TestScriptABAControlsAttributeSourceByDigest(t *testing.T) {
	t.Run("D1 fixed", func(t *testing.T) {
		d1, _, obs := runScriptUnderWrapper(t, "D1 fixed")
		if !obs.passed || obs.candidateDigest != d1 || !allEqual(obs.executedDigests, d1) || !allEqual(obs.recordedDigests, d1) {
			t.Fatalf("D1 control: d1=%s obs=%+v", d1, obs)
		}
	})
	t.Run("D2 fixed", func(t *testing.T) {
		_, d2, obs := runScriptUnderWrapper(t, "D2 fixed")
		if !obs.passed || obs.candidateDigest != d2 || !allEqual(obs.executedDigests, d2) || !allEqual(obs.recordedDigests, d2) {
			t.Fatalf("D2 control: d2=%s obs=%+v", d2, obs)
		}
	})
}

// Characterization: the interpreter read D2 for every case, the pre/post digest
// checks saw D1 only, and the receipt attributes the execution to D1.
func TestScriptABAIsNotDetectedAndAttributedToDeclaredSource(t *testing.T) {
	d1, d2, obs := runScriptUnderWrapper(t, "D1-D2-D1")
	if obs.caseCount == 0 || len(obs.executedDigests) != obs.caseCount {
		t.Fatalf("wrapper observation is incomplete: %+v", obs)
	}
	if !allEqual(obs.executedDigests, d2) {
		t.Fatalf("the interpreter did not receive D2 (D1=%s D2=%s): %+v", d1, d2, obs)
	}
	if !obs.passed || obs.rejectReason != "" {
		t.Fatalf("current behavior changed: the ABA candidate is now rejected (%q); update this characterization together with EB-04", obs.rejectReason)
	}
	if obs.candidateDigest != d1 || !allEqual(obs.recordedDigests, d1) {
		t.Fatalf("current attribution changed: candidate=%s recorded=%v D1=%s", obs.candidateDigest, obs.recordedDigests, d1)
	}
	// Recorded for the reader: Execution.SourceDigest says D1 while D2 ran.
	t.Logf("EB-02 characterization: executed=%s recorded=%s (receipt Passed=%v)", d2, d1, obs.passed)
}

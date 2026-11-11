package generatorselection

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// EB-02/EB-04 (#2595): the Python candidate is run under a test-only wrapper
// that swaps script bytes (D1 -> D2) after the pre-case check and before the
// interpreter starts, then restores D1 before the post-case check.
//
// The selector runs a script from an execution snapshot written from the bytes
// it verified, so the wrapper has two places to attack:
//   - the declared path in the candidate directory: the interpreter no longer
//     reads it, so D1 runs and the candidate passes with a correct attribution;
//   - the snapshot handed to the interpreter ("$2"): the rewrite changes the
//     modification time of the snapshot, so the candidate is rejected.
//
// The wrapper is a test-only execution path. A process with the same OS
// privileges that also restores the modification time is not detected; that
// is OS isolation (#2515) and is not claimed here. The wrapper's sha256sum of
// "$2" is the independent observation of the bytes handed to the interpreter.

type abaObservation struct {
	passed          bool
	rejectReason    string
	candidateDigest string
	executedDigests []string // digest of the script bytes the wrapper handed to the interpreter
	recordedDigests []string // Execution.SourceDigest per case
	caseCount       int
}

func runScriptUnderWrapper(t *testing.T, language, mode string) (d1, d2 string, obs abaObservation) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the wrapper is a POSIX shell script")
	}
	strategy, ext, comment := "python-template", ".py", "#"
	if language == LanguageJavaScript {
		strategy, ext, comment = "javascript-template", ".js", "//"
	}
	if _, rt := ResolveScriptRuntime(language); !rt.Available {
		t.Skipf("%s interpreter is unavailable", language)
	}
	for _, tool := range []string{"sh", "sha256sum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is unavailable", tool)
		}
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, strategy, "1", filepath.Join(candidates, strategy))
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
	d2Bytes := append(append([]byte(nil), d1Bytes...), []byte("\n"+comment+" D2: bytes differ from the declared source\n")...)
	d1, d2 = digest(d1Bytes), digest(d2Bytes)
	if d1 == d2 {
		t.Fatal("D1 and D2 must differ")
	}

	work := t.TempDir()
	d2Path := filepath.Join(work, "d2"+ext)
	d1Path := filepath.Join(work, "d1"+ext)
	logPath := filepath.Join(work, "executed.log")
	for path, data := range map[string][]byte{d1Path: d1Bytes, d2Path: d2Bytes} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var body, restore string
	switch mode {
	case "D1 fixed", "D2 fixed":
	case "declared path D1-D2-D1":
		body = fmt.Sprintf("cp %q %q\n", d2Path, sourcePath)
		restore = fmt.Sprintf("cp %q %q\n", d1Path, sourcePath)
	case "snapshot D1-D2-D1":
		body = fmt.Sprintf("cp %q \"$2\"\n", d2Path)
		restore = fmt.Sprintf("cp %q \"$2\"\n", d1Path)
	case "symlink retarget":
		// The declared source is a symlink inside the candidate directory. The
		// wrapper points it at D2 for the run and back at D1 afterwards.
		dir := filepath.Dir(sourcePath)
		if err := os.Rename(sourcePath, filepath.Join(dir, "real"+ext)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "other"+ext), d2Bytes, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("real"+ext, sourcePath); err != nil {
			t.Skipf("symlink is unavailable: %v", err)
		}
		body = fmt.Sprintf("ln -sfn other%s %q\n", ext, sourcePath)
		restore = fmt.Sprintf("ln -sfn real%s %q\n", ext, sourcePath)
	case "two-level symlink retarget":
		// declared -> hop -> real. The wrapper retargets the second hop only.
		dir := filepath.Dir(sourcePath)
		if err := os.Rename(sourcePath, filepath.Join(dir, "real"+ext)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "other"+ext), d2Bytes, 0o600); err != nil {
			t.Fatal(err)
		}
		hop := filepath.Join(dir, "hop"+ext)
		if err := os.Symlink("real"+ext, hop); err != nil {
			t.Skipf("symlink is unavailable: %v", err)
		}
		if err := os.Symlink("hop"+ext, sourcePath); err != nil {
			t.Skipf("symlink is unavailable: %v", err)
		}
		body = fmt.Sprintf("ln -sfn other%s %q\n", ext, hop)
		restore = fmt.Sprintf("ln -sfn real%s %q\n", ext, hop)
	default:
		t.Fatalf("unknown mode %q", mode)
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
		// A handwritten manifest: the Generator provenance of the template would
		// reject D2 as "does not match the declared generator".
		raw, err := json.Marshal(CandidateManifest{
			Schema: CandidateSchema, ID: generated.Manifest.ID, Origin: "handwritten", Language: language,
			Source: generated.Manifest.Source, SourceDigest: d2,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(generated.Path, "candidate.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	receipt, err := SelectForTargetWithExecution(context.Background(), workspaceDir, contract, fixtures, candidates, "", []string{strategy}, ExecutionOptions{CommandPrefix: []string{"sh", wrapperPath}})
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

var abaLanguages = []string{LanguagePython, LanguageJavaScript}

func TestScriptABAControlsAttributeSourceByDigest(t *testing.T) {
	for _, language := range abaLanguages {
		t.Run(language+"/D1 fixed", func(t *testing.T) {
			d1, _, obs := runScriptUnderWrapper(t, language, "D1 fixed")
			if !obs.passed || obs.candidateDigest != d1 || !allEqual(obs.executedDigests, d1) || !allEqual(obs.recordedDigests, d1) {
				t.Fatalf("D1 control: d1=%s obs=%+v", d1, obs)
			}
		})
		t.Run(language+"/D2 fixed", func(t *testing.T) {
			_, d2, obs := runScriptUnderWrapper(t, language, "D2 fixed")
			if !obs.passed || obs.candidateDigest != d2 || !allEqual(obs.executedDigests, d2) || !allEqual(obs.recordedDigests, d2) {
				t.Fatalf("D2 control: d2=%s obs=%+v", d2, obs)
			}
		})
	}
}

// A D1 -> D2 -> D1 rewrite of the bytes handed to the interpreter is rejected:
// D2 ran for the first case, the digest is D1 again afterwards, and the changed
// modification time of the snapshot gives the rewrite away.
func TestScriptABAOnExecutionSnapshotIsRejected(t *testing.T) {
	for _, language := range abaLanguages {
		t.Run(language, func(t *testing.T) {
			d1, d2, obs := runScriptUnderWrapper(t, language, "snapshot D1-D2-D1")
			if len(obs.executedDigests) != 1 || obs.executedDigests[0] != d2 {
				t.Fatalf("the wrapper did not hand D2 to the interpreter for exactly one case (D1=%s D2=%s): %+v", d1, d2, obs)
			}
			if obs.passed || !strings.Contains(obs.rejectReason, "source changed during execution after case") {
				t.Fatalf("the ABA rewrite of the execution snapshot was not rejected: %+v", obs)
			}
			if obs.caseCount != 1 {
				t.Fatalf("no further case may run after the rewrite was detected: %+v", obs)
			}
		})
	}
}

// A D1 -> D2 -> D1 rewrite of the declared path in the candidate directory, or
// a retargeted symlink (one or two hops) at that path, does not reach the
// interpreter: every case runs the verified D1 bytes, so the D1 attribution in
// the receipt is correct.
func TestScriptChangeOfDeclaredPathDoesNotReachInterpreter(t *testing.T) {
	for _, language := range abaLanguages {
		for _, mode := range []string{"declared path D1-D2-D1", "symlink retarget", "two-level symlink retarget"} {
			t.Run(language+"/"+mode, func(t *testing.T) {
				d1, _, obs := runScriptUnderWrapper(t, language, mode)
				if obs.caseCount == 0 || len(obs.executedDigests) != obs.caseCount {
					t.Fatalf("wrapper observation is incomplete: %+v", obs)
				}
				if !allEqual(obs.executedDigests, d1) {
					t.Fatalf("the interpreter received bytes other than the verified D1=%s: %+v", d1, obs)
				}
				if !obs.passed || obs.candidateDigest != d1 || !allEqual(obs.recordedDigests, d1) {
					t.Fatalf("the verified D1 run was not qualified and attributed to D1=%s: %+v", d1, obs)
				}
			})
		}
	}
}

package teimdreader

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// This deliberately demonstrates a limit of the reference -candidate-shape
// flag: the binary self-reports an embedded constant, not an externally
// verified binding between the Builder's actual output and its executed logic.
// The forged binary is a negative witness, not a TEI-generated deliverable.
func TestCandidateShapeSelfReportCannotQualifyActualArtifact(t *testing.T) {
	candidate, err := ParseRuleCandidate(fixtureSource(t, "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := GenerateGo(candidate)
	if err != nil {
		t.Fatal(err)
	}
	expectedShape, err := CandidateShapeFingerprint(candidate)
	if err != nil {
		t.Fatal(err)
	}

	// A post-generation substitution changes the actual predicate while
	// preserving the constant the executable advertises to its caller.
	const original = "e.from == nodeID"
	const substituted = "e.to == nodeID"
	if bytes.Count(generated, []byte(original)) != 1 {
		t.Fatal("the actual generated predicate was not found exactly once")
	}
	tampered := bytes.Replace(generated, []byte(original), []byte(substituted), 1)
	if bytes.Equal(generated, tampered) {
		t.Fatal("the negative case did not alter the generated Go")
	}

	type observation struct {
		shape   string
		digest  [32]byte
		results map[string][]string
	}
	observations := map[string]observation{}
	for _, tc := range []struct {
		name   string
		source []byte
	}{
		{"original", generated},
		{"predicate-substituted", tampered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sourcePath := filepath.Join(dir, "generated.go")
			binaryName := "generated"
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}
			binaryPath := filepath.Join(dir, binaryName)
			if err := os.WriteFile(sourcePath, tc.source, 0600); err != nil {
				t.Fatal(err)
			}
			build := exec.Command("go", "build", "-o", binaryPath, sourcePath)
			build.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GO111MODULE=off")
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build negative witness: %v\n%s", err, output)
			}
			actualBinary, err := os.ReadFile(binaryPath)
			if err != nil {
				t.Fatal(err)
			}
			shapeCall := exec.Command(binaryPath, "-candidate-shape")
			shapeOut, err := shapeCall.Output()
			if err != nil {
				t.Fatalf("read self-reported shape: %v", err)
			}
			shape := strings.TrimSpace(string(shapeOut))
			if shape != expectedShape {
				t.Fatalf("self-reported shape changed: %q, want %q", shape, expectedShape)
			}
			resultOut, err := exec.Command(binaryPath).Output()
			if err != nil {
				t.Fatalf("run compiled binary: %v", err)
			}
			var result map[string][]string
			if err := json.Unmarshal(resultOut, &result); err != nil {
				t.Fatalf("actual result is not JSON: %v", err)
			}
			observations[tc.name] = observation{
				shape: shape, digest: sha256.Sum256(actualBinary), results: result,
			}
		})
	}
	originalObserved := observations["original"]
	substitutedObserved := observations["predicate-substituted"]
	if originalObserved.shape != substitutedObserved.shape {
		t.Fatal("this witness requires identical self-reported shape")
	}
	if originalObserved.digest == substitutedObserved.digest {
		t.Fatal("different executed Go unexpectedly produced an identical binary digest")
	}
	if reflect.DeepEqual(originalObserved.results, substitutedObserved.results) {
		t.Fatal("predicate substitution did not affect actual Node outputs")
	}
	// Comparing externally observed output and disk artifact bytes detects the
	// deliberate substitution. No permission/meaning qualification is issued.
}

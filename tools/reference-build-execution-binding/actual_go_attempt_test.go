package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

// A Go build invocation is an observed attempt, not a BuildExecutionResult
// identity. This test-only witness is deliberately not an authority claim.
type actualGoAttemptWitness struct {
	Attempt           string `json:"attempt"`
	PlanFingerprint   string `json:"planFingerprint"`
	ArtifactDigest    string `json:"artifactDigest"`
	ResultFingerprint string `json:"resultFingerprint"`
}

func actualGoDigestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

// Execute the actual Go command twice, then bind both observed output files
// into the existing domain-neutral Plan/ArtifactSet/Result primitives.
// The compiler can reuse its cache; this is not independent from toolchain
// trust, or proof that the full source/toolchain closure was captured.
func TestActualGoBuildAttemptsBindToReferenceResult(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go executable is unavailable: %v", err)
	}
	builderDigest, err := actualGoDigestFile(goTool)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	sourceBytes := []byte("package main\nfunc main() { println(\"tei-reference\") }\n")
	if err := os.WriteFile(source, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceHash := sha256.Sum256(sourceBytes)
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "go-source", ID: "main", Locator: source,
			Digest: "sha256:" + hex.EncodeToString(sourceHash[:]),
		}},
		Builder: plugin.BuilderIdentity{
			ID:      core.MustCanonicalID("tei.builder.go_attempt_reference"),
			Version: "1.0.0", ContractVersion: "tei.reference.go-build/v1",
			ArtifactDigest: builderDigest,
		},
		Recipe:         plugin.BuildRecipe{Kind: "go-build", Entrypoint: "main.go"},
		OutputContract: "go-executable", Target: runtime.GOOS + "/" + runtime.GOARCH,
		Parameters: []plugin.BuildParameter{
			{Name: "flags", Value: "-trimpath -buildvcs=false -ldflags=-buildid="},
		},
	}
	planFP, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}

	var observations []actualGoAttemptWitness
	var sets []ArtifactSet
	for _, id := range []string{"first", "second"} {
		output := filepath.Join(dir, id, "app")
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(goTool, "build", "-trimpath", "-buildvcs=false",
			"-ldflags=-buildid=", "-o", output, source)
		cmd.Env = append(os.Environ(),
			"GO111MODULE=off", "GOWORK=off", "GOFLAGS=",
			"GOTOOLCHAIN=local", "GOOS="+runtime.GOOS,
			"GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
		if outputBytes, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s actual Go build failed: %v: %s", id, err, outputBytes)
		}
		digest, err := actualGoDigestFile(output)
		if err != nil {
			t.Fatal(err)
		}
		set := ArtifactSet{
			FormatVersion: artifactSetFormatVersion,
			Members: []ArtifactMember{{
				Role: "executable", Format: "application/octet-stream",
				Digest: digest, Locator: output,
			}},
		}
		setFP, err := artifactSetFingerprint(set)
		if err != nil {
			t.Fatal(err)
		}
		resultFP, err := buildExecutionResultFingerprint(BuildExecutionResult{
			FormatVersion:            buildExecutionResultFormatVersion,
			ExecutionPlanFingerprint: planFP, ArtifactSetFingerprint: setFP,
		})
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, actualGoAttemptWitness{
			Attempt: id, PlanFingerprint: planFP,
			ArtifactDigest: digest, ResultFingerprint: resultFP,
		})
		sets = append(sets, set)
	}
	if len(observations) != 2 || observations[0].Attempt == observations[1].Attempt {
		t.Fatal("two build commands were not recorded as distinct attempts")
	}
	comparison, err := compareArtifactSets(sets[0], sets[1])
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.Reproducible ||
		observations[0].ArtifactDigest != observations[1].ArtifactDigest ||
		observations[0].ResultFingerprint != observations[1].ResultFingerprint {
		t.Fatalf("two actual Go build invocations yielded different content: %+v, %+v, %+v",
			observations[0], observations[1], comparison)
	}

	// Two observed Go commands with matching exact output can yield the
	// same content identity. Pinning alone is not artifact Acceptance.
	pinned, err := pinReproducibleArtifactSet(sets[0], sets[1])
	if err != nil {
		t.Fatalf("reproducible actual Go artifacts were not pinned: %v", err)
	}
	setIdentity, err := artifactSetFingerprint(sets[0])
	if err != nil {
		t.Fatal(err)
	}
	if pinned != setIdentity || pinned != comparison.FirstFingerprint {
		t.Fatalf("pinned identity %q differs from the actual Go artifact set %q", pinned, setIdentity)
	}

	// A synthetically changed role on an observed output must be rejected
	// rather than silently treating its matching bytes as the same Set.
	tampered := sets[1]
	tampered.Members = append([]ArtifactMember(nil), sets[1].Members...)
	tampered.Members[0].Role = "unexpected-executable"
	rejected, err := pinReproducibleArtifactSet(sets[0], tampered)
	if err == nil || rejected != "" {
		t.Fatalf("tampered actual Go Artifact Set was pinned: identity=%q err=%v", rejected, err)
	}
	drift, err := compareArtifactSets(sets[0], tampered)
	if err != nil {
		t.Fatal(err)
	}
	if drift.Reproducible || len(drift.MissingFromSecond) != 1 || len(drift.ExtraInSecond) != 1 {
		t.Fatalf("role tampering lost structured mismatch Evidence: %+v", drift)
	}

	// The same artifact bytes under a different recipe must NOT inherit the
	// previously observed Result identity merely because its digest matches.
	changedPlan := plan
	changedPlan.Recipe.Entrypoint = "different-entrypoint.go"
	changedPlanFP, err := buildExecutionPlanFingerprint(changedPlan)
	if err != nil {
		t.Fatal(err)
	}
	setFP, err := artifactSetFingerprint(sets[0])
	if err != nil {
		t.Fatal(err)
	}
	changedResultFP, err := buildExecutionResultFingerprint(BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: changedPlanFP, ArtifactSetFingerprint: setFP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if changedResultFP == observations[0].ResultFingerprint {
		t.Fatal("content digest alone incorrectly replaced the exact build plan identity")
	}

	// Preserve a structured witness and verify what was actually written.
	// t.TempDir is intentionally ephemeral; CI artifact retention is separate.
	witnessPath := filepath.Join(dir, "actual-go-attempts.json")
	serialized, err := json.Marshal(observations)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(witnessPath, serialized, 0o600); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(witnessPath)
	if err != nil {
		t.Fatal(err)
	}
	var replay []actualGoAttemptWitness
	if err := json.Unmarshal(saved, &replay); err != nil {
		t.Fatal(err)
	}
	if len(replay) != 2 || replay[0] != observations[0] || replay[1] != observations[1] {
		t.Fatal(fmt.Sprintf("written attempt witnesses could not be replayed: %+v", replay))
	}
}

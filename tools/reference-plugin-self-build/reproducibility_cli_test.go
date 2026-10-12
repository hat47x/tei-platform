package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const selfBuildCLITestChildEnv = "TEI_SELF_BUILD_CLI_TEST_CHILD"

// Exercise the same exit-code and stdout writer as the production CLI in a
// subprocess. The mismatch itself is injected synthetically; this is not a
// claim that an actual Go builder produced two different outputs.
func TestSelfBuildCLIProcessFailureBoundary(t *testing.T) {
	if mode := os.Getenv(selfBuildCLITestChildEnv); mode != "" {
		var err error
		switch mode {
		case "mismatch":
			first := sha256.Sum256([]byte("first synthetic artifact"))
			second := sha256.Sum256([]byte("second synthetic artifact"))
			err = fmt.Errorf("self-build bootstrap: %w", nonReproducibleSelfBuild(
				"transform-stage1-stage2-output",
				"sha256:"+hex.EncodeToString(first[:]),
				"sha256:"+hex.EncodeToString(second[:]),
			))
		case "execution-error":
			err = errors.New("compiler exited before a second output was available")
		default:
			t.Fatalf("unexpected subprocess mode %q", mode)
		}
		os.Exit(writeSelfBuildCLIResult(os.Stdout, os.Stderr, evidence{}, err))
	}

	for _, tc := range []struct {
		name             string
		mode             string
		mismatchExpected bool
	}{
		{name: "compared outputs differ", mode: "mismatch", mismatchExpected: true},
		{name: "build failed before comparison", mode: "execution-error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSelfBuildCLIProcessFailureBoundary$")
			cmd.Env = append(os.Environ(), selfBuildCLITestChildEnv+"="+tc.mode)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr

			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("expected nonzero CLI exit 1, got err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("nonzero CLI exit lost its stderr diagnostic")
			}
			if !tc.mismatchExpected {
				if stdout.Len() != 0 {
					t.Fatalf("ordinary execution failure emitted mismatch Evidence: %q", stdout.String())
				}
				return
			}

			var record selfBuildMismatchEvidence
			decoder := json.NewDecoder(&stdout)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&record); err != nil {
				t.Fatalf("CLI stdout is not a readable mismatch Evidence JSON: %v", err)
			}
			var trailing any
			if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
				t.Fatalf("mismatch CLI printed extra or malformed output: %v", err)
			}
			if record.Schema != selfBuildMismatchSchema ||
				record.Stage != "transform-stage1-stage2-output" ||
				record.Outcome != "reproducibility-failure" ||
				record.FirstArtifactDigest == record.SecondArtifactDigest ||
				!strings.HasPrefix(record.FirstArtifactDigest, "sha256:") ||
				!strings.HasPrefix(record.SecondArtifactDigest, "sha256:") ||
				record.Reproducible ||
				record.ArtifactDigest != "" ||
				record.ArtifactAccepted ||
				record.ArtifactExportAllowed ||
				record.ArtifactLockAllowed {
				t.Fatalf("non-reproducible output was incorrectly accepted: %+v", record)
			}
		})
	}
}

func TestSelfBuildCLIResultPreservesSuccessfulEvidence(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := writeSelfBuildCLIResult(&stdout, &stderr, evidence{
		Schema: "test-reference-success",
	}, nil)
	if exitCode != 0 || stderr.Len() != 0 {
		t.Fatalf("successful CLI outcome was changed: exit=%d stderr=%q", exitCode, stderr.String())
	}
	var record evidence
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Schema != "test-reference-success" {
		t.Fatalf("successful Evidence was altered: %q", record.Schema)
	}
}

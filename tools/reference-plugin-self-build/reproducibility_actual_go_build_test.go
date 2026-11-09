package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const actualBuildChildMode = "TEI_TEST_ACTUAL_BUILD_CHILD_MODE"
const actualBuildChildDir = "TEI_TEST_ACTUAL_BUILD_CHILD_DIR"

// This deliberately changes a linker value between two genuine Go builds.
// It verifies that real artifact differences cross the same file-comparison
// and CLI failure boundary as the self-build seed-builder. It does NOT prove
// that identical build inputs are spontaneously non-reproducible.
func TestActualGoBuildComparisonFailureBoundary(t *testing.T) {
	if mode := os.Getenv(actualBuildChildMode); mode != "" {
		dir := os.Getenv(actualBuildChildDir)
		sourcePath := filepath.Join(dir, "main.go")
		program := "package main\nimport \"fmt\"\nvar variant = \"default\"\nfunc main() { fmt.Print(variant) }\n"
		if err := os.WriteFile(sourcePath, []byte(program), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		goPath, err := exec.LookPath("go")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}

		firstPath := filepath.Join(dir, "first-builder")
		secondPath := filepath.Join(dir, "second-builder")
		if runtime.GOOS == "windows" {
			firstPath += ".exe"
			secondPath += ".exe"
		}
		build := func(path, variant string) error {
			cmd := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false",
				"-ldflags=-X main.variant="+variant, "-o", path, sourcePath)
			cmd.Env = append(os.Environ(), "GO111MODULE=off", "GOTOOLCHAIN=local", "GOFLAGS=")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("actual Go builder failed: %w: %s", err, output)
			}
			return nil
		}

		if err := build(firstPath, "first"); err != nil {
			os.Exit(writeSelfBuildCLIResult(os.Stdout, os.Stderr, evidence{}, err))
		}
		if mode == "build-error" {
			if err := os.WriteFile(sourcePath, []byte("package main\nfunc main( {\n"), 0o600); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
		secondVariant := "first"
		if mode == "mismatch" {
			secondVariant = "second"
		}
		if err := build(secondPath, secondVariant); err != nil {
			os.Exit(writeSelfBuildCLIResult(os.Stdout, os.Stderr, evidence{}, err))
		}
		for _, item := range []struct {
			path string
			want string
		}{{firstPath, "first"}, {secondPath, secondVariant}} {
			output, err := exec.Command(item.path).Output()
			if err != nil || string(output) != item.want {
				fmt.Fprintf(os.Stderr, "actual built binary failed verification: %v\n", err)
				os.Exit(2)
			}
		}

		firstDigest, secondDigest, comparisonErr := compareSelfBuildFileDigests(
			"transform-seed-builder", firstPath, secondPath,
		)
		if mode == "mismatch" && comparisonErr == nil {
			fmt.Fprintln(os.Stderr, "deliberate binary mismatch not detected")
			os.Exit(2)
		}
		if mode == "same" && comparisonErr != nil {
			fmt.Fprintln(os.Stderr, "identical build inputs unexpectedly differed:", comparisonErr)
			os.Exit(2)
		}
		if comparisonErr != nil {
			os.Exit(writeSelfBuildCLIResult(os.Stdout, os.Stderr, evidence{}, comparisonErr))
		}
		if firstDigest != secondDigest {
			fmt.Fprintln(os.Stderr, "matching binary digests unexpectedly differ")
			os.Exit(2)
		}
		os.Exit(writeSelfBuildCLIResult(os.Stdout, os.Stderr, evidence{Schema: "test-real-go-build-success"}, nil))
	}

	for _, tc := range []struct {
		name         string
		mode         string
		exitCode     int
		jsonEvidence bool
	}{
		{name: "deliberate genuine binary mismatch", mode: "mismatch", exitCode: 1, jsonEvidence: true},
		{name: "equal actual binaries", mode: "same", exitCode: 0},
		{name: "second compilation fails", mode: "build-error", exitCode: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command(os.Args[0], "-test.run=^TestActualGoBuildComparisonFailureBoundary$")
			cmd.Env = append(os.Environ(),
				actualBuildChildMode+"="+tc.mode,
				actualBuildChildDir+"="+dir,
			)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			runErr := cmd.Run()
			if tc.exitCode == 0 {
				if runErr != nil {
					t.Fatalf("actual equal builds failed: %v; stderr=%q", runErr, stderr.String())
				}
			} else {
				var exitErr *exec.ExitError
				if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != tc.exitCode {
					t.Fatalf("unexpected exit: %v; stderr=%q", runErr, stderr.String())
				}
				if stderr.Len() == 0 {
					t.Fatal("failed build lost its stderr diagnostic")
				}
			}

			if tc.mode == "build-error" {
				if stdout.Len() != 0 {
					t.Fatalf("compiler failure was misclassified as reproducibility Evidence: %q", stdout.String())
				}
				return
			}
			decoder := json.NewDecoder(&stdout)
			decoder.DisallowUnknownFields()
			if tc.jsonEvidence {
				var record selfBuildMismatchEvidence
				if err := decoder.Decode(&record); err != nil {
					t.Fatalf("no structured mismatch Evidence for actual binaries: %v", err)
				}
				if record.Schema != selfBuildMismatchSchema ||
					record.Stage != "transform-seed-builder" ||
					record.Outcome != "reproducibility-failure" ||
					record.FirstArtifactDigest == record.SecondArtifactDigest ||
					!strings.HasPrefix(record.FirstArtifactDigest, "sha256:") ||
					!strings.HasPrefix(record.SecondArtifactDigest, "sha256:") ||
					record.Reproducible ||
					record.ArtifactDigest != "" ||
					record.ArtifactAccepted ||
					record.ArtifactExportAllowed ||
					record.ArtifactLockAllowed {
					t.Fatalf("actual mismatched binaries were accepted: %+v", record)
				}
			} else {
				var record evidence
				if err := decoder.Decode(&record); err != nil || record.Schema != "test-real-go-build-success" {
					t.Fatalf("successful actual builds did not preserve JSON: %v / %q", err, stdout.String())
				}
			}
			var additional any
			if err := decoder.Decode(&additional); !errors.Is(err, io.EOF) {
				t.Fatalf("CLI must produce exactly one JSON object: %v", err)
			}
		})
	}
}

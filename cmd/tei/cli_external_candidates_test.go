package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The R3-A acceptance test runs the actual tei executable. Calling
// runExternalCandidatesCommand or operations.ExternalCandidates directly is
// not evidence that the advertised command is really reachable.
func TestExternalCandidatesR3AActualCLIProcess(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build tei CLI: %v\n%s", err, out)
	}

	fixture := filepath.Join("..", "..", "internal", "externalanalysis", "testdata", "rea-6.8.0", "analysis.json")
	if _, err := os.Stat(fixture); err != nil {
		t.Fatalf("REA captured fixture: %v", err)
	}
	const (
		nodeID     = "jsrg_node_237478f2a27c9614a522cbfc7ab60fa96943a458d164945d25af634f1240b652"
		relationID = "jsrg_relation_0f6757a241c56fd84ecaec91f538d8297d222eb5625e4cdbcb2de7603f330718"
		evidenceID = "ev_a88c70bcd066b2f04fdf97cb349369f0779b9c68cc4ac6c0e5f38f3539477a93"
		digest     = "sha256:7ea3780d8dc23d00c380145af62d63bf194e39decbaefab3d82c5a3b4134bba9"
	)
	base := []string{"external-candidates", "-evidence-path", fixture, "-tool-version", "6.8.0",
		"-observation-id", nodeID, "-relation-id", relationID}
	run := func(argv ...string) (int, externalCandidatesCLIResponse, string, string) {
		t.Helper()
		cmd := exec.Command(binary, argv...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		exit := 0
		if err := cmd.Run(); err != nil {
			var failure *exec.ExitError
			if !errors.As(err, &failure) {
				t.Fatalf("run tei CLI: %v", err)
			}
			exit = failure.ExitCode()
		}
		var response externalCandidatesCLIResponse
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("non-JSON CLI response: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
		}
		return exit, response, stdout.String(), stderr.String()
	}

	t.Run("selected invoked received", func(t *testing.T) {
		exit, response, _, stderr := run(base...)
		if exit != 0 || !response.OK || response.Error != nil || stderr != "" {
			t.Fatalf("success exit=%d response=%+v stderr=%q", exit, response, stderr)
		}
		if response.SchemaVersion != "tei.cli.external-candidates/v0" || response.Result == nil {
			t.Fatalf("envelope: %+v", response)
		}
		view := response.Result
		if view.SchemaVersion != "tei.operations.external-candidates/v0" || view.Effect != "read-only" {
			t.Fatalf("operation view: %+v", view)
		}
		if view.Attempt.EvidenceID != evidenceID || view.Attempt.ReceivedDigest != digest ||
			view.Attempt.SourceRevision == "" || view.Attempt.ReadAt == "" {
			t.Fatalf("source/received attempt: %+v", view.Attempt)
		}
		if view.Observation == nil || view.Observation.NodeID != nodeID ||
			view.Relation == nil || view.Relation.RelationID != relationID ||
			view.Relation.From == view.Relation.To {
			t.Fatalf("selected graph identity/direction: %+v", view)
		}
		if view.Coverage == nil || view.Coverage.Status != "unknown" ||
			len(view.Unknowns) == 0 || view.Unselected.Observations == 0 ||
			view.Unselected.Relations == 0 || len(view.AuthorityLimits) == 0 {
			t.Fatalf("coverage/unknown/authority limits were discarded: %+v", view)
		}
	})

	t.Run("matching pins", func(t *testing.T) {
		argv := append(append([]string{}, base...), "-expected-evidence-id", evidenceID,
			"-expected-received-digest", digest)
		exit, response, _, _ := run(argv...)
		if exit != 0 || !response.OK {
			t.Fatalf("matching pinned bytes: exit=%d error=%+v", exit, response.Error)
		}
	})

	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantCode   string
	}{
		{"missing ID", base[:len(base)-2], 2, "INVALID_ARGUMENT"},
		{"unknown flag", append(append([]string{}, base...), "-made-up", "x"), 2, "INVALID_ARGUMENT"},
		{"positional input", append(append([]string{}, base...), "extra"), 2, "INVALID_ARGUMENT"},
		{"missing local file", []string{"external-candidates", "-evidence-path", filepath.Join(t.TempDir(), "absent.json"), "-tool-version", "6.8.0", "-observation-id", nodeID, "-relation-id", relationID}, 2, "EXTERNAL_CANDIDATE_READ_FAILED"},
		{"pin changed", append(append([]string{}, base...), "-expected-received-digest", "sha256:incorrect"), 1, "EXTERNAL_CANDIDATE_STALE"},
		{"unknown relation", append(append([]string{}, base[:len(base)-1]...), "not-a-relation"), 1, "EXTERNAL_CANDIDATE_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exit, response, _, stderr := run(tt.args...)
			if exit != tt.wantExit || response.OK || response.Error == nil || response.Error.Code != tt.wantCode || stderr != "" {
				t.Fatalf("exit=%d response=%+v stderr=%q want exit=%d code=%s", exit, response, stderr, tt.wantExit, tt.wantCode)
			}
		})
	}

	t.Run("invalid saved evidence", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "invalid-evidence.json")
		if err := os.WriteFile(file, []byte("not JSON"), 0o600); err != nil {
			t.Fatal(err)
		}
		argv := append([]string{}, base...)
		argv[2] = file
		exit, response, _, _ := run(argv...)
		if exit != 1 || response.Error == nil || response.Error.Code != "EXTERNAL_CANDIDATE_UNUSABLE" {
			t.Fatalf("invalid Evidence must not be a candidate: exit=%d response=%+v", exit, response)
		}
	})

	t.Run("help and read-only discovery", func(t *testing.T) {
		cmd := exec.Command(binary, "external-candidates", "-h")
		output, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "-evidence-path") || !strings.Contains(string(output), "-relation-id") {
			t.Fatalf("CLI help: %v\n%s", err, output)
		}
		cmd = exec.Command(binary, "commands")
		out, err := cmd.Output()
		if err != nil || !bytes.Contains(out, []byte(`"name":"external-candidates"`)) {
			t.Fatalf("read-only discovery did not advertise the executable entry: %v
%s", err, out)
		}
	})
}

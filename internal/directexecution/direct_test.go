package directexecution_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/directexecution"
)

func buildTestBinary(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hello\") }\n"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	bin := filepath.Join(dir, "test-bin")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Dir = t.TempDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v: %s", err, out)
	}
	return bin
}

func TestDirectExecutionProducesEvidence(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(input, []byte("hello-world"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "output.json")
	if err := os.WriteFile(outputPath, []byte(`{"result":"ok"}`), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}

	binary := buildTestBinary(t, t.TempDir())
	evidence, err := directexecution.Run(directexecution.Options{
		Command:        binary,
		Args:           []string{},
		InputPath:      input,
		OutputPath:     outputPath,
		Stdin:          bytes.NewReader(nil),
		FunctionID:     "pure-compute-v1",
		SourceRevision: "test-rev",
		Toolchain:      "go1.25.0",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if evidence.Schema != directexecution.EvidenceSchema {
		t.Errorf("Schema = %q, want %q", evidence.Schema, directexecution.EvidenceSchema)
	}
	if !evidence.ExecutionRequestAccepted.Accepted {
		t.Error("ExecutionRequestAccepted.Accepted = false")
	}
	if !evidence.WorkerExecuted.Executed {
		t.Error("WorkerExecuted.Executed = false")
	}
	if !evidence.ActualOutputVerified.Verified {
		t.Error("ActualOutputVerified.Verified = false")
	}
	if evidence.ExecutionEligibility.Eligible != true {
		t.Error("ExecutionEligibility.Eligible = false")
	}
	if evidence.PlacementAuthority.Granted != true {
		t.Error("PlacementAuthority.Granted = false")
	}

	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded directexecution.Evidence
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Schema != evidence.Schema {
		t.Error("Evidence is not stable across JSON round-trip")
	}
}

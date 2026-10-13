package main

import (
	"os/exec"
	"testing"
)

// Temporary diagnostic: obtain gofmt's exact diff from a trusted CI Go runner.
func TestDiagnosticsNewArtifactProbeGofmt(t *testing.T) {
	out, err := exec.Command("gofmt", "-d", "docs_only_actual_artifacts_test.go").CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt diagnostic execution: %v: %s", err, out)
	}
	if len(out) != 0 {
		t.Fatalf("actual gofmt drift:\n%s", out)
	}
}

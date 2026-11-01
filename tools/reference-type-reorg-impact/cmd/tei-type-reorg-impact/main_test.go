package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateCandidateMode(t *testing.T) {
	tests := []struct {
		name         string
		split        bool
		candidateOut string
		wantError    bool
	}{
		{name: "read-only split", split: true},
		{name: "merge impact", split: false},
		{name: "merge candidate", split: false, candidateOut: "candidate.json"},
		{name: "split candidate is refused", split: true, candidateOut: "candidate.json", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCandidateMode(tc.split, tc.candidateOut)
			if (err != nil) != tc.wantError {
				t.Fatalf("validateCandidateMode(split=%v, candidateOut=%q) error=%v, wantError=%v", tc.split, tc.candidateOut, err, tc.wantError)
			}
		})
	}
}

// Exercise the actual CLI entrypoint, not only its option validator. In
// particular the invalid option combination must fail before workspace I/O.
func TestCLIBlocksSplitCandidateBeforeWorkspaceRead(t *testing.T) {
	if testing.Short() {
		t.Skip("CLI subprocess smoke test")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "tei-type-reorg-impact")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	candidate := filepath.Join(dir, "must-not-exist.json")
	workspace := filepath.Join(dir, "missing-workspace")
	cmd := exec.Command(binary,
		"-workspace", workspace,
		"-split", "-from", "sales.order_line_id",
		"-into", "sales.order_id,sales.order_status",
		"-candidate-out", candidate,
	)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("split candidate CLI unexpectedly succeeded: %s", output)
	}
	if !strings.Contains(string(output), "not supported with -split") {
		t.Fatalf("expected split candidate rejection, got: %s", output)
	}
	if strings.Contains(string(output), "load Workspace") {
		t.Fatalf("split candidate reached Workspace I/O: %s", output)
	}
	if _, err := os.Stat(candidate); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate file was created or stat failed unexpectedly: %v", err)
	}

	// The read-only split mode remains available. This sample is the same
	// source Workspace used in the reference-type-reorg-impact README.
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	readOnly := exec.Command(binary, "-workspace", root,
		"-split", "-from", "sales.order_line_id",
		"-into", "sales.order_id,sales.order_status")
	impactJSON, err := readOnly.CombinedOutput()
	if err != nil {
		t.Fatalf("read-only split failed: %v\n%s", err, impactJSON)
	}
	var impact struct {
		ReviewRequired bool  `json:"reviewRequired"`
		Applied        bool  `json:"applied"`
		Attributes     []any `json:"attributes"`
		Diagnostics    []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(impactJSON, &impact); err != nil {
		t.Fatalf("read-only split did not return impact JSON: %v\n%s", err, impactJSON)
	}
	if !impact.ReviewRequired || impact.Applied || len(impact.Attributes) == 0 {
		t.Fatalf("unexpected read-only split impact: %+v", impact)
	}

	// Merge candidate export must remain usable. Candidate generation is
	// not an acceptance decision; the synthetic revision is never accepted.
	mergeOut := filepath.Join(dir, "merge-candidate.json")
	merge := exec.Command(binary, "-workspace", root,
		"-from", "sales.order_line_id", "-into", "sales.order_id",
		"-candidate-out", mergeOut,
		"-base-revision", "sha256:"+strings.Repeat("a", 64))
	if output, err := merge.CombinedOutput(); err != nil {
		t.Fatalf("merge candidate CLI regressed: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(mergeOut)
	if err != nil {
		t.Fatalf("merge candidate was not created: %v", err)
	}
	var candidateDoc struct {
		Delta []struct {
			Op            string `json:"op"`
			AttributeType string `json:"attributeType"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(raw, &candidateDoc); err != nil {
		t.Fatalf("merge candidate is not JSON: %v", err)
	}
	if len(candidateDoc.Delta) == 0 {
		t.Fatal("merge candidate contains no changes")
	}
	for _, d := range candidateDoc.Delta {
		if d.Op != "set-attribute-type" || d.AttributeType != "sales.order_id" {
			t.Fatalf("unexpected merge candidate delta: %+v", d)
		}
	}
}

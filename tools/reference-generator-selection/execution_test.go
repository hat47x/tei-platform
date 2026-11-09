package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaseResultsRecordExecutionEvidence(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, candidate := range receipt.Candidates {
		if len(candidate.Results) != 10 {
			t.Fatalf("%s has %d results", candidate.ID, len(candidate.Results))
		}
		for _, result := range candidate.Results {
			if result.Execution == nil {
				t.Fatalf("%s/%s has no execution evidence", candidate.ID, result.CaseID)
			}
			if result.Execution.ExitCode != 0 {
				t.Fatalf("%s/%s exit code=%d", candidate.ID, result.CaseID, result.Execution.ExitCode)
			}
			if result.Execution.StdoutDigest == "" || result.Execution.StderrDigest == "" {
				t.Fatalf("%s/%s has incomplete stdout/stderr digests", candidate.ID, result.CaseID)
			}
			checked++
		}
	}
	if checked != 10*len(receipt.Candidates) {
		t.Fatalf("expected %d case executions, checked %d", 10*len(receipt.Candidates), checked)
	}
}

func TestWrappedSelectionRejectsSeparateLineageRunner(t *testing.T) {
	workspace, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := Generate(contract, "template", filepath.Join(candidates, "template"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := generated.Manifest
	manifest.Lineage = &Lineage{Depth: 1, GeneratorSource: "must-not-execute.go"}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated.Path, "candidate.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	receipt, err := SelectForTargetWithExecution(context.Background(), workspace, contract, fixtures, candidates, "", []string{"template"}, ExecutionOptions{CommandPrefix: []string{"must-not-execute"}})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Candidates) != 1 || len(receipt.Candidates[0].Results) != 0 || !strings.Contains(receipt.Candidates[0].RejectReason, "lineage execution is unsupported") {
		t.Fatalf("lineage escaped wrapper boundary: %+v", receipt)
	}
}

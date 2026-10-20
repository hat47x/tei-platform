package generatorselection

import (
	"context"
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
	if checked != 30 {
		t.Fatalf("expected 30 case executions, checked %d", checked)
	}
}

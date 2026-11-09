package generatorselection

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func paths(t *testing.T) (workspaceDir, contract, fixtures, candidates string) {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "tools", "reference-generator-selection", "testdata")
	return filepath.Join(root, "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json"),
		filepath.Join(base, "candidates")
}

func TestSelectsTwoIndependentGeneratorsAndRejectsTheDangerousOne(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("selection did not qualify any candidate: %+v", receipt)
	}
	if receipt.GrantsRuntimeAuthority || receipt.ProvesRealProvider {
		t.Fatal("selection claimed authority or a real provider")
	}
	qualified := map[string]bool{}
	for _, id := range receipt.Qualified {
		qualified[id] = true
	}
	if !qualified["template"] || !qualified["alternative"] {
		t.Fatalf("independent correct candidates were not qualified: %+v", receipt.Qualified)
	}
	if qualified["dangerous"] {
		t.Fatal("dangerous candidate was qualified")
	}
	if receipt.WorkspaceRevision == "" || receipt.FixtureDigest == "" || receipt.SuiteVersion == "" {
		t.Fatalf("receipt lost provenance: %+v", receipt)
	}
	var dangerous CandidateReceipt
	for _, candidate := range receipt.Candidates {
		if candidate.ID == "dangerous" {
			dangerous = candidate
		}
	}
	if dangerous.Passed || dangerous.RejectReason == "" {
		t.Fatalf("dangerous candidate was not rejected with a reason: %+v", dangerous)
	}
	failed := map[string]bool{}
	for _, result := range dangerous.Results {
		if !result.Passed {
			failed[result.CaseID] = true
		}
	}
	for _, want := range []string{"half-up-tie", "missing-quantity", "currency-mismatch", "precision-overflow"} {
		if !failed[want] {
			t.Errorf("dangerous candidate did not fail fixture %s: %+v", want, dangerous.Results)
		}
	}
}

func TestCandidateWithTamperedSourceIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	temp := t.TempDir()
	sourceDir := filepath.Join(candidates, "template")
	targetDir := filepath.Join(temp, "template")
	if err := os.CopyFS(targetDir, os.DirFS(sourceDir)); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(targetDir, "main.go")
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, append(content, "\n// tampered\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, temp)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Candidates) != 1 || receipt.Candidates[0].Passed {
		t.Fatalf("tampered candidate was not rejected: %+v", receipt)
	}
	if !strings.Contains(receipt.Candidates[0].RejectReason, "source digest mismatch") {
		t.Fatalf("unexpected rejection reason: %+v", receipt.Candidates[0])
	}
}

func TestContractWithUnknownAcceptedAttributeIsRejected(t *testing.T) {
	workspaceDir, contractPath, fixtures, candidates := paths(t)
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), `"attribute": "sales.order_line.quantity"`, `"attribute": "sales.order_line.not_a_field"`, 1)
	badPath := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(badPath, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Select(context.Background(), workspaceDir, badPath, fixtures, candidates); err == nil {
		t.Fatal("contract with an unknown accepted attribute was accepted")
	}
}

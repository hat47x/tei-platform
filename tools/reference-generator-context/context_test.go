package generatorcontext

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

func paths(t *testing.T) (workspace, contract, fixtures string) {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "tools", "reference-generator-selection", "testdata")
	return filepath.Join(root, "reference", "workspace"),
		filepath.Join(base, "line-amount", "contract.json"),
		filepath.Join(base, "line-amount", "fixtures.json")
}

func TestBuildProducesBoundedContractContext(t *testing.T) {
	workspaceDir, contract, fixtures := paths(t)
	context, raw, err := Build(workspaceDir, contract, fixtures, 32768)
	if err != nil {
		t.Fatal(err)
	}
	if context.Schema != Schema || context.Contract.ID != "sales.line-amount" {
		t.Fatalf("unexpected context: %+v", context)
	}
	if len(context.Contract.Inputs) != 2 || len(context.Fixtures.Cases) != 10 {
		t.Fatalf("unexpected context size: %+v", context)
	}
	if context.Workspace.Revision == "" || context.Contract.Digest == "" || context.Fixtures.Digest == "" {
		t.Fatalf("missing revision or digest: %+v", context)
	}
	if len(raw) != context.ActualBytes || len(raw) > context.MaxBytes {
		t.Fatalf("actual=%d raw=%d max=%d", context.ActualBytes, len(raw), context.MaxBytes)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	workspaceDir, contract, fixtures := paths(t)
	_, first, err := Build(workspaceDir, contract, fixtures, 32768)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := Build(workspaceDir, contract, fixtures, 32768)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("context is not deterministic")
	}
}

func TestBuildRejectsTinyBudget(t *testing.T) {
	workspaceDir, contract, fixtures := paths(t)
	if _, _, err := Build(workspaceDir, contract, fixtures, 100); err == nil {
		t.Fatal("tiny budget was accepted")
	}
}

func TestBuildRejectsFixtureContractMismatch(t *testing.T) {
	workspaceDir, contract, fixtures := paths(t)
	raw, err := os.ReadFile(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	var changed map[string]any
	if err := json.Unmarshal(raw, &changed); err != nil {
		t.Fatal(err)
	}
	changed["contractId"] = "other.contract"
	modified, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixtures.json")
	if err := os.WriteFile(path, modified, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Build(workspaceDir, contract, path, 32768); err == nil {
		t.Fatal("fixture contract mismatch was accepted")
	}
}

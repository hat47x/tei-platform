package generatorselection

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAllIsDeterministicAndBindsContract(t *testing.T) {
	_, contractPath, _, _ := paths(t)
	firstRoot := filepath.Join(t.TempDir(), "first")
	secondRoot := filepath.Join(t.TempDir(), "second")
	first, err := GenerateAll(contractPath, firstRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateAll(contractPath, secondRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) || len(first) != 3 {
		t.Fatalf("unexpected generated set: %d and %d", len(first), len(second))
	}
	wantDigest, err := ContractDigest(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range first {
		if first[i].Manifest.ID != second[i].Manifest.ID {
			t.Fatalf("strategy order changed: %q vs %q", first[i].Manifest.ID, second[i].Manifest.ID)
		}
		for _, name := range []string{"main.go", "candidate.json"} {
			a, err := os.ReadFile(filepath.Join(first[i].Path, name))
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(filepath.Join(second[i].Path, name))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("%s/%s is not deterministic", first[i].Manifest.ID, name)
			}
		}
		if first[i].Manifest.Generator == nil {
			t.Fatalf("%s has no generator provenance", first[i].Manifest.ID)
		}
		if first[i].Manifest.Generator.ContractDigest != wantDigest {
			t.Fatalf("%s contract digest=%q want %q", first[i].Manifest.ID, first[i].Manifest.Generator.ContractDigest, wantDigest)
		}
		source, err := os.ReadFile(filepath.Join(first[i].Path, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		if first[i].SourceDigest != digest(source) {
			t.Fatalf("%s source digest does not match its bytes", first[i].Manifest.ID)
		}
	}
}

func TestGeneratedCandidatesAreSelectable(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateAll(contract, candidates); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("generated candidates were not selectable: %+v", receipt)
	}
	qualified := map[string]bool{}
	for _, id := range receipt.Qualified {
		qualified[id] = true
	}
	if !qualified["template"] || !qualified["alternative"] || qualified["dangerous"] {
		t.Fatalf("unexpected qualified set: %+v", receipt.Qualified)
	}
	for _, candidate := range receipt.Candidates {
		if candidate.Generator == nil {
			t.Fatalf("%s lost generator provenance in the receipt", candidate.ID)
		}
	}
}

func TestGeneratedCandidateWithAnotherContractIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateAll(contract, candidates); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(contract)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), `"scale": 2`, `"scale": 3`, 1)
	changedPath := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(changedPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, changedPath, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("candidates generated from another contract were accepted")
	}
	for _, candidate := range receipt.Candidates {
		if candidate.Passed || !strings.Contains(candidate.RejectReason, "contract digest mismatch") {
			t.Fatalf("candidate %s was not rejected for contract mismatch: %+v", candidate.ID, candidate)
		}
	}
}

func TestGenerateRefusesExistingOutput(t *testing.T) {
	_, contract, _, _ := paths(t)
	out := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(contract, "template", out); err == nil {
		t.Fatal("existing output directory was accepted")
	}
}

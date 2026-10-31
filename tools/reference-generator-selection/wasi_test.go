package generatorselection

import (
	"context"
	"path/filepath"
	"testing"
)

func TestWasiTemplateIsSelectable(t *testing.T) {
	if _, _, ok := resolveWasiRuntime(); !ok {
		t.Skip("node is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "wasi-template", "1", filepath.Join(candidates, "wasi-template"))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Language != LanguageWasi || generated.Manifest.Source != "main.go" {
		t.Fatalf("unexpected wasi manifest: %+v", generated.Manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "wasi-template" {
		t.Fatalf("wasi candidate was not selected: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if candidate.Language != LanguageWasi || candidate.BinaryDigest == "" {
		t.Fatalf("wasi candidate was not recorded: %+v", candidate)
	}
	if candidate.Runtime == nil || !candidate.Runtime.Available || candidate.Runtime.Language != LanguageWasi {
		t.Fatalf("wasi runtime evidence is missing: %+v", candidate.Runtime)
	}
	if candidate.BuildEvidence == nil || candidate.BuildEvidence.GOOS != "wasip1" ||
		candidate.BuildEvidence.GOARCH != "wasm" ||
		candidate.BuildEvidence.Reproducibility != "path-independent-twice" ||
		candidate.BuildEvidence.ReproducedDigest != candidate.BinaryDigest {
		t.Fatalf("wasi build evidence is missing: %+v", candidate.BuildEvidence)
	}
	for _, result := range candidate.Results {
		if !result.Passed {
			t.Fatalf("wasi candidate failed fixture %s: %+v", result.CaseID, result)
		}
	}
}

func TestGoAndWasiCandidatesProduceSameFixtureOutputs(t *testing.T) {
	if _, _, ok := resolveWasiRuntime(); !ok {
		t.Skip("node is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateVersion(contract, "wasi-template", "1", filepath.Join(candidates, "wasi-template")); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	qualified := map[string]bool{}
	for _, id := range receipt.Qualified {
		qualified[id] = true
	}
	if !qualified["template"] || !qualified["wasi-template"] {
		t.Fatalf("go and wasi candidates were not both qualified: %+v", receipt.Qualified)
	}
	results := map[string]map[string]Result{}
	for _, candidate := range receipt.Candidates {
		results[candidate.ID] = map[string]Result{}
		for _, result := range candidate.Results {
			results[candidate.ID][result.CaseID] = result.Got
		}
	}
	for caseID, goResult := range results["template"] {
		wasiResult, ok := results["wasi-template"][caseID]
		if !ok {
			t.Fatalf("wasi candidate is missing case %s", caseID)
		}
		if goResult != wasiResult {
			t.Fatalf("case %s differs: go=%+v wasi=%+v", caseID, goResult, wasiResult)
		}
	}
}

package generatorselection

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestWasmTemplateIsSelectable(t *testing.T) {
	if _, _, _, ok := resolveWasmRuntime(); !ok {
		t.Skip("node or Go wasm_exec_node.js is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "wasm-template", "1", filepath.Join(candidates, "wasm-template"))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Language != LanguageWasm || generated.Manifest.Source != "main.go" {
		t.Fatalf("unexpected wasm manifest: %+v", generated.Manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "wasm-template" {
		t.Fatalf("wasm candidate was not selected: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if candidate.Language != LanguageWasm || candidate.BinaryDigest == "" {
		t.Fatalf("wasm candidate was not recorded as a wasm artifact: %+v", candidate)
	}
	if candidate.Runtime == nil || !candidate.Runtime.Available || candidate.Runtime.Language != LanguageWasm {
		t.Fatalf("wasm runtime evidence is missing: %+v", candidate.Runtime)
	}
	if candidate.BuildEvidence == nil || candidate.BuildEvidence.Reproducibility != "path-independent-twice" ||
		candidate.BuildEvidence.ReproducedDigest != candidate.BinaryDigest {
		t.Fatalf("wasm build reproducibility evidence is missing: %+v", candidate.BuildEvidence)
	}
	for _, result := range candidate.Results {
		if !result.Passed {
			t.Fatalf("wasm candidate failed fixture %s: %+v", result.CaseID, result)
		}
	}
}

func TestGoAndWasmCandidatesProduceSameFixtureOutputs(t *testing.T) {
	if _, _, _, ok := resolveWasmRuntime(); !ok {
		t.Skip("node or Go wasm_exec_node.js is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateVersion(contract, "wasm-template", "1", filepath.Join(candidates, "wasm-template")); err != nil {
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
	if !qualified["template"] || !qualified["wasm-template"] {
		t.Fatalf("go and wasm candidates were not both qualified: %+v", receipt.Qualified)
	}
	results := map[string]map[string]Result{}
	for _, candidate := range receipt.Candidates {
		results[candidate.ID] = map[string]Result{}
		for _, result := range candidate.Results {
			results[candidate.ID][result.CaseID] = result.Got
		}
	}
	for caseID, goResult := range results["template"] {
		if wasmResult := results["wasm-template"][caseID]; wasmResult != goResult {
			t.Fatalf("case %s differs: go=%+v wasm=%+v", caseID, goResult, wasmResult)
		}
	}
}

func TestUnavailableWasmRuntimeIsNotSuccess(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "wasm-template", "1", filepath.Join(candidates, "wasm-template")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Candidates) != 1 {
		t.Fatalf("missing wasm runtime was treated as success: %+v", receipt)
	}
	if !strings.Contains(receipt.Candidates[0].RejectReason, "runtime unavailable") {
		t.Fatalf("unexpected rejection reason: %q", receipt.Candidates[0].RejectReason)
	}
}

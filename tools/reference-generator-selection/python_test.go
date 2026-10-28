package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonTemplateIsSelectable(t *testing.T) {
	if _, _, ok := findPython(); !ok {
		t.Skip("python3/python is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "python-template", "1", filepath.Join(candidates, "python-template"))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Language != LanguagePython || generated.Manifest.Source != "main.py" {
		t.Fatalf("unexpected python manifest: %+v", generated.Manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "python-template" {
		t.Fatalf("python candidate was not selected: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if candidate.Language != LanguagePython || candidate.BinaryDigest != "" {
		t.Fatalf("python candidate was recorded as a native artifact: %+v", candidate)
	}
	if candidate.Runtime == nil || !candidate.Runtime.Available || candidate.Runtime.Language != LanguagePython {
		t.Fatalf("python runtime evidence is missing: %+v", candidate.Runtime)
	}
	if len(candidate.Results) != 10 {
		t.Fatalf("unexpected case count: %d", len(candidate.Results))
	}
	for _, result := range candidate.Results {
		if !result.Passed {
			t.Fatalf("python candidate failed fixture %s: %+v", result.CaseID, result)
		}
	}
}

func TestGoAndPythonCandidatesProduceSameFixtureOutputs(t *testing.T) {
	if _, _, ok := findPython(); !ok {
		t.Skip("python3/python is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateVersion(contract, "python-template", "1", filepath.Join(candidates, "python-template")); err != nil {
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
	if !qualified["template"] || !qualified["python-template"] {
		t.Fatalf("go and python candidates were not both qualified: %+v", receipt.Qualified)
	}
	results := map[string]map[string]Result{}
	for _, candidate := range receipt.Candidates {
		results[candidate.ID] = map[string]Result{}
		for _, result := range candidate.Results {
			results[candidate.ID][result.CaseID] = result.Got
		}
	}
	for caseID, goResult := range results["template"] {
		if pythonResult := results["python-template"][caseID]; pythonResult != goResult {
			t.Fatalf("case %s differs: go=%+v python=%+v", caseID, goResult, pythonResult)
		}
	}
}

func TestUnavailablePythonRuntimeIsNotSuccess(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "python-template", "1", filepath.Join(candidates, "python-template")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Candidates) != 1 {
		t.Fatalf("missing python runtime was treated as success: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if !strings.Contains(candidate.RejectReason, "runtime unavailable") {
		t.Fatalf("unexpected rejection reason: %q", candidate.RejectReason)
	}
	if candidate.Runtime == nil || candidate.Runtime.Available {
		t.Fatalf("missing runtime evidence: %+v", candidate.Runtime)
	}
}

func TestCandidateLanguageMustMatchSourceExtension(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(candidates, "template", "candidate.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest CandidateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Language = LanguagePython
	changed, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Candidates[0].RejectReason, "source extension") {
		t.Fatalf("language/source mismatch was not rejected: %+v", receipt.Candidates[0])
	}
}

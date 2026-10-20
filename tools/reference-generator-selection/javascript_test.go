package generatorselection

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestJavaScriptTemplateIsSelectable(t *testing.T) {
	if _, _, ok := findJavaScript(); !ok {
		t.Skip("node is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "javascript-template", "1", filepath.Join(candidates, "javascript-template"))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Language != LanguageJavaScript || generated.Manifest.Source != "main.js" {
		t.Fatalf("unexpected javascript manifest: %+v", generated.Manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "javascript-template" {
		t.Fatalf("javascript candidate was not selected: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if candidate.Language != LanguageJavaScript || candidate.BinaryDigest != "" {
		t.Fatalf("javascript candidate was recorded as a native artifact: %+v", candidate)
	}
	if candidate.Runtime == nil || !candidate.Runtime.Available || candidate.Runtime.Language != LanguageJavaScript {
		t.Fatalf("javascript runtime evidence is missing: %+v", candidate.Runtime)
	}
	for _, result := range candidate.Results {
		if !result.Passed {
			t.Fatalf("javascript candidate failed fixture %s: %+v", result.CaseID, result)
		}
	}
}

func TestGoAndJavaScriptCandidatesProduceSameFixtureOutputs(t *testing.T) {
	if _, _, ok := findJavaScript(); !ok {
		t.Skip("node is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateVersion(contract, "javascript-template", "1", filepath.Join(candidates, "javascript-template")); err != nil {
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
	if !qualified["template"] || !qualified["javascript-template"] {
		t.Fatalf("go and javascript candidates were not both qualified: %+v", receipt.Qualified)
	}
	results := map[string]map[string]Result{}
	for _, candidate := range receipt.Candidates {
		results[candidate.ID] = map[string]Result{}
		for _, result := range candidate.Results {
			results[candidate.ID][result.CaseID] = result.Got
		}
	}
	for caseID, goResult := range results["template"] {
		if javascriptResult := results["javascript-template"][caseID]; javascriptResult != goResult {
			t.Fatalf("case %s differs: go=%+v javascript=%+v", caseID, goResult, javascriptResult)
		}
	}
}

func TestUnavailableJavaScriptRuntimeIsNotSuccess(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "javascript-template", "1", filepath.Join(candidates, "javascript-template")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Candidates) != 1 {
		t.Fatalf("missing node runtime was treated as success: %+v", receipt)
	}
	if !strings.Contains(receipt.Candidates[0].RejectReason, "runtime unavailable") {
		t.Fatalf("unexpected rejection reason: %q", receipt.Candidates[0].RejectReason)
	}
}

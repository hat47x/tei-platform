package generatorselection

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestJavaTemplateIsSelectable(t *testing.T) {
	if _, _, _, ok := findJava(); !ok {
		t.Skip("javac/java is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := GenerateVersion(contract, "java-template", "1", filepath.Join(candidates, "java-template"))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Language != LanguageJava || generated.Manifest.Source != "Main.java" {
		t.Fatalf("unexpected java manifest: %+v", generated.Manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "java-template" {
		t.Fatalf("java candidate was not selected: %+v", receipt)
	}
	candidate := receipt.Candidates[0]
	if candidate.Language != LanguageJava || candidate.BinaryDigest == "" {
		t.Fatalf("java candidate was not recorded as a compiled artifact: %+v", candidate)
	}
	if candidate.Runtime == nil || !candidate.Runtime.Available || candidate.Runtime.Language != LanguageJava {
		t.Fatalf("java runtime evidence is missing: %+v", candidate.Runtime)
	}
	for _, result := range candidate.Results {
		if !result.Passed {
			t.Fatalf("java candidate failed fixture %s: %+v", result.CaseID, result)
		}
	}
}

func TestGoAndJavaCandidatesProduceSameFixtureOutputs(t *testing.T) {
	if _, _, _, ok := findJava(); !ok {
		t.Skip("javac/java is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template")); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateVersion(contract, "java-template", "1", filepath.Join(candidates, "java-template")); err != nil {
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
	if !qualified["template"] || !qualified["java-template"] {
		t.Fatalf("go and java candidates were not both qualified: %+v", receipt.Qualified)
	}
	results := map[string]map[string]Result{}
	for _, candidate := range receipt.Candidates {
		results[candidate.ID] = map[string]Result{}
		for _, result := range candidate.Results {
			results[candidate.ID][result.CaseID] = result.Got
		}
	}
	for caseID, goResult := range results["template"] {
		if javaResult := results["java-template"][caseID]; javaResult != goResult {
			t.Fatalf("case %s differs: go=%+v java=%+v", caseID, goResult, javaResult)
		}
	}
}

func TestUnavailableJavaRuntimeIsNotSuccess(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	if _, err := GenerateVersion(contract, "java-template", "1", filepath.Join(candidates, "java-template")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "")
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Candidates) != 1 {
		t.Fatalf("missing javac/java was treated as success: %+v", receipt)
	}
	if !strings.Contains(receipt.Candidates[0].RejectReason, "runtime unavailable") {
		t.Fatalf("unexpected rejection reason: %q", receipt.Candidates[0].RejectReason)
	}
}

package generatorselection

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordedPythonCandidateIsSelectable(t *testing.T) {
	if _, _, ok := findPython(); !ok {
		t.Skip("python3/python is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	sourceDir := filepath.Join(t.TempDir(), "python-source")
	generated, err := GenerateVersion(contract, "python-template", "1", sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation")
	out := filepath.Join(t.TempDir(), "recorded-python")
	manifest, err := Record(RecordOptions{
		ID: "recorded-python", Provider: "recorded-fixture", Model: "fixture-v0", Language: LanguagePython,
		ContractPath: contract, SourcePath: filepath.Join(generated.Path, "main.py"),
		PromptPath:  filepath.Join(base, "prompt.txt"),
		ContextPath: filepath.Join(base, "context.json"),
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Language != LanguagePython || manifest.Source != "main.py" {
		t.Fatalf("unexpected recorded manifest: %+v", manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "recorded-python" {
		t.Fatalf("recorded python candidate was not selected: %+v", receipt)
	}
	if receipt.Candidates[0].Language != LanguagePython || receipt.Candidates[0].Runtime == nil || !receipt.Candidates[0].Runtime.Available {
		t.Fatalf("recorded python candidate lost runtime evidence: %+v", receipt.Candidates[0])
	}
}

func TestRecordRejectsLanguageSourceMismatch(t *testing.T) {
	_, contract, _, candidates := paths(t)
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation")
	_, err := Record(RecordOptions{
		ID: "mismatch", Provider: "p", Model: "m", Language: LanguagePython,
		ContractPath: contract, SourcePath: filepath.Join(candidates, "template", "main.go"),
		PromptPath:  filepath.Join(base, "prompt.txt"),
		ContextPath: filepath.Join(base, "context.json"),
	}, filepath.Join(t.TempDir(), "mismatch"))
	if err == nil || !strings.Contains(err.Error(), "source extension") {
		t.Fatalf("language/source mismatch was accepted: %v", err)
	}
}

func TestExternalCommandRecordsPythonCandidate(t *testing.T) {
	if _, _, ok := findPython(); !ok {
		t.Skip("python3/python is not available")
	}
	workspaceDir, contract, fixtures, _ := paths(t)
	sourceDir := filepath.Join(t.TempDir(), "python-source")
	generated, err := GenerateVersion(contract, "python-template", "1", sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation")
	root := t.TempDir()
	manifest, err := GenerateWithCommand(context.Background(), ModelOptions{
		ID: "model-python", Provider: "external-command", Model: "fixture-model-v0", Language: LanguagePython,
		ContractPath: contract, PromptPath: filepath.Join(base, "prompt.txt"),
		ContextPath: filepath.Join(base, "context.json"), OutDir: filepath.Join(root, "model-python"),
		Command: fmt.Sprintf("cp %q \"$TEI_SOURCE\"", filepath.Join(generated.Path, "main.py")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Language != LanguagePython || manifest.Source != "main.py" {
		t.Fatalf("unexpected model manifest: %+v", manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "model-python" {
		t.Fatalf("external python candidate was not selected: %+v", receipt)
	}
}

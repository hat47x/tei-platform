package generatorselection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func modelOptions(t *testing.T, out, command string) ModelOptions {
	t.Helper()
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation")
	return ModelOptions{
		ID: "model-template", Provider: "external-command", Model: "fixture-model-v0",
		PromptPath: filepath.Join(base, "prompt.txt"), ContextPath: filepath.Join(base, "context.json"),
		OutDir: out, Command: command, Timeout: 30 * time.Second,
	}
}

func TestExternalCommandIsRecordedAndSelectable(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	root := t.TempDir()
	source := filepath.Join(candidates, "template", "main.go")
	command := fmt.Sprintf("cp %q \"$TEI_SOURCE\"", source)
	manifest, err := GenerateWithCommand(context.Background(), modelOptions(t, filepath.Join(root, "recorded"), command))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Generation == nil || manifest.Generation.Provider != "external-command" || manifest.Generation.OutputDigest != manifest.SourceDigest {
		t.Fatalf("unexpected generation provenance: %+v", manifest.Generation)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "model-template" {
		t.Fatalf("recorded model candidate was not selected: %+v", receipt)
	}
}

func TestExternalCommandFailureIsRejected(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "recorded")
	_, err := GenerateWithCommand(context.Background(), modelOptions(t, out, "exit 7"))
	if err == nil || !strings.Contains(err.Error(), "generation command failed") {
		t.Fatalf("failing command was not rejected: %v", err)
	}
	if _, statErr := os.Stat(out); statErr == nil {
		t.Fatal("failed command left an output directory")
	}
}

func TestExternalCommandWithoutSourceIsRejected(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "recorded")
	_, err := GenerateWithCommand(context.Background(), modelOptions(t, out, "true"))
	if err == nil || !strings.Contains(err.Error(), "did not write TEI_SOURCE") {
		t.Fatalf("command without source was not rejected: %v", err)
	}
}

func TestExternalCommandRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "existing")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, _, candidates := paths(t)
	source := filepath.Join(candidates, "template", "main.go")
	_, err := GenerateWithCommand(context.Background(), modelOptions(t, out, fmt.Sprintf("cp %q \"$TEI_SOURCE\"", source)))
	if err == nil {
		t.Fatal("existing output directory was accepted")
	}
}

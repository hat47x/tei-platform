package generatorselection

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recordFixture(t *testing.T, out string) CandidateManifest {
	t.Helper()
	_, _, _, candidates := paths(t)
	manifest, err := Record(RecordOptions{
		ID: "recorded-template", Provider: "recorded-fixture", Model: "fixture-v0",
		SourcePath:  filepath.Join(candidates, "template", "main.go"),
		PromptPath:  filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation", "prompt.txt"),
		ContextPath: filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "generation", "context.json"),
		Note:        "recorded test source",
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestRecordedCandidateIsSelectableAndKeepsGenerationProvenance(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	root := t.TempDir()
	manifest := recordFixture(t, filepath.Join(root, "recorded"))
	if manifest.Generation == nil || manifest.Generation.PromptPath == "" || manifest.Generation.ContextPath == "" {
		t.Fatalf("recorded manifest lost generation paths: %+v", manifest.Generation)
	}
	if manifest.Generation.OutputDigest != manifest.SourceDigest {
		t.Fatalf("recorded output digest does not match source: %+v", manifest)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "recorded-template" {
		t.Fatalf("recorded candidate was not selected: %+v", receipt)
	}
	if receipt.Candidates[0].Generation == nil || receipt.Candidates[0].Generation.PromptDigest == "" {
		t.Fatalf("receipt lost generation provenance: %+v", receipt.Candidates[0])
	}
}

func TestRecordedPromptTamperIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	root := t.TempDir()
	recordFixture(t, filepath.Join(root, "recorded"))
	if err := os.WriteFile(filepath.Join(root, "recorded", "prompt.txt"), []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("tampered prompt was accepted")
	}
	reason := receipt.Candidates[0].RejectReason
	if !strings.Contains(reason, "generation file verification failed") || !strings.Contains(reason, "prompt digest mismatch") {
		t.Fatalf("unexpected rejection reason: %q", reason)
	}
}

func TestRecordRefusesExistingOutput(t *testing.T) {
	_, _, _, candidates := paths(t)
	out := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Record(RecordOptions{
		ID: "recorded", Provider: "p", Model: "m",
		SourcePath:  filepath.Join(candidates, "template", "main.go"),
		PromptPath:  filepath.Join(candidates, "template", "candidate.json"),
		ContextPath: filepath.Join(candidates, "template", "candidate.json"),
	}, out)
	if err == nil {
		t.Fatal("existing output directory was accepted")
	}
}

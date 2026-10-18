package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func metaOptions(t *testing.T, out string) MetaOptions {
	t.Helper()
	_, contract, _, _ := paths(t)
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata")
	return MetaOptions{
		ID: "meta-template", Provider: "recorded-fixture", Model: "meta-fixture-v0",
		ContractPath:        contract,
		GeneratorSourcePath: filepath.Join(base, "meta", "generator.go"),
		PromptPath:          filepath.Join(base, "generation", "prompt.txt"),
		ContextPath:         filepath.Join(base, "generation", "context.json"),
		OutDir:              out, Timeout: 2 * time.Minute,
	}
}

func TestMetaGeneratorCreatesSelectableCandidate(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	root := t.TempDir()
	out := filepath.Join(root, "meta-template")
	manifest, err := GenerateWithMetaGenerator(context.Background(), metaOptions(t, out))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Lineage == nil || manifest.Lineage.Depth != 1 || manifest.Lineage.GeneratorSource != "generator.go" {
		t.Fatalf("unexpected lineage: %+v", manifest.Lineage)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Qualified) != 1 || receipt.Qualified[0] != "meta-template" {
		t.Fatalf("meta candidate was not selected: %+v", receipt)
	}
	if receipt.Candidates[0].Lineage == nil || receipt.Candidates[0].Lineage.Depth != 1 {
		t.Fatalf("receipt lost lineage: %+v", receipt.Candidates[0])
	}
}

func TestMetaCandidateWithTamperedGeneratorSourceIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	root := t.TempDir()
	out := filepath.Join(root, "meta-template")
	if _, err := GenerateWithMetaGenerator(context.Background(), metaOptions(t, out)); err != nil {
		t.Fatal(err)
	}
	generatorPath := filepath.Join(out, "generator.go")
	content, err := os.ReadFile(generatorPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(generatorPath, append(content, "\n// tampered\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("tampered generator source was accepted")
	}
	if !strings.Contains(receipt.Candidates[0].RejectReason, "generator source digest mismatch") {
		t.Fatalf("unexpected rejection reason: %q", receipt.Candidates[0].RejectReason)
	}
}

func TestMetaDepthTwoIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, _ := paths(t)
	root := t.TempDir()
	out := filepath.Join(root, "meta-template")
	if _, err := GenerateWithMetaGenerator(context.Background(), metaOptions(t, out)); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(out, "candidate.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest CandidateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Lineage == nil {
		t.Fatal("manifest has no lineage")
	}
	manifest.Lineage.Depth = 2
	changed, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Candidates[0].RejectReason, "depth 2") {
		t.Fatalf("depth 2 was not rejected: %+v", receipt.Candidates[0])
	}
}

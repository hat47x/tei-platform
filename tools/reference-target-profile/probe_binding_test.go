package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func TestProbeUsesSelectedIdentityInsteadOfDirectoryName(t *testing.T) {
	if _, runtime := generatorselection.ResolveScriptRuntime(generatorselection.LanguagePython); !runtime.Available {
		t.Skip("python runtime unavailable")
	}
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	strategy := "python-long-running-template"
	if _, err := generatorselection.Generate(contract, strategy, filepath.Join(candidates, "actual-selected")); err != nil {
		t.Fatal(err)
	}
	unselected, err := generatorselection.Generate(contract, "python-template", filepath.Join(candidates, strategy))
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "unselected-executed")
	source := fmt.Sprintf("from pathlib import Path\nPath(%q).write_text('executed')\n", marker)
	if err := os.WriteFile(filepath.Join(unselected.Path, unselected.Manifest.Source), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.GeneratorID = strategy
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(profilePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Run(context.Background(), RunOptions{ProfilePath: profilePath, WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("unselected directory was executed: marker error=%v", err)
	}
	if receipt.ProcessModel.CandidateID != strategy || receipt.ProcessModel.Binding != "source-digest" || receipt.ProcessModel.SourceDigest == "" {
		t.Fatalf("missing selected source binding: %+v", receipt.ProcessModel)
	}
	if !receipt.Passed || !receipt.Selection.Passed || receipt.ProcessModel.Processes != 1 || receipt.ProcessModel.Responses != 10 {
		t.Fatalf("selected identity did not reach probe: %+v", receipt)
	}
}

func selectedProbeFixture(t *testing.T, strategy string) (Profile, RunOptions, generatorselection.Receipt, string) {
	t.Helper()
	workspace, contract, fixtures, _ := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	candidateDir := filepath.Join(candidates, strategy)
	if _, err := generatorselection.Generate(contract, strategy, candidateDir); err != nil {
		t.Fatal(err)
	}
	selection, err := generatorselection.Select(context.Background(), workspace, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Passed {
		t.Fatalf("selection did not pass: %+v", selection)
	}
	return Profile{GeneratorID: strategy, ProcessModel: ProcessLongRunning}, RunOptions{CandidatesDir: candidates, FixturePath: fixtures}, selection, candidateDir
}

func TestProbeRejectsChangedSourceAfterSelection(t *testing.T) {
	if _, runtime := generatorselection.ResolveScriptRuntime(generatorselection.LanguagePython); !runtime.Available {
		t.Skip("python runtime unavailable")
	}
	profile, opts, selection, dir := selectedProbeFixture(t, "python-long-running-template")
	marker := filepath.Join(t.TempDir(), "changed-executed")
	source := []byte(fmt.Sprintf("from pathlib import Path\nPath(%q).write_text('executed')\n", marker))
	if err := os.WriteFile(filepath.Join(dir, "main.py"), source, 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest generatorselection.CandidateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.SourceDigest = digestBytes(source)
	raw, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "candidate.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	evidence := enforceProcessModel(context.Background(), profile, opts, selection)
	if evidence.Passed || evidence.Processes != 0 || !strings.Contains(evidence.Reason, "source digest") {
		t.Fatalf("changed source reached probe: %+v", evidence)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("changed source was executed: %v", err)
	}
}

func TestProbeRejectsBinaryDigestMismatch(t *testing.T) {
	profile, opts, selection, _ := selectedProbeFixture(t, "long-running-template")
	selection.Candidates[0].BinaryDigest = "sha256:" + strings.Repeat("0", 64)
	evidence := enforceProcessModel(context.Background(), profile, opts, selection)
	if evidence.Passed || evidence.Processes != 0 || !strings.Contains(evidence.Reason, "binary digest") {
		t.Fatalf("different executable reached probe: %+v", evidence)
	}
}

func TestProbeExcludesUnselectedGoFiles(t *testing.T) {
	profile, opts, selection, dir := selectedProbeFixture(t, "long-running-template")
	marker := filepath.Join(t.TempDir(), "extra-executed")
	extra := fmt.Sprintf("package main\nimport \"os\"\nfunc init(){os.WriteFile(%q,[]byte(\"executed\"),0600)}\n", marker)
	if err := os.WriteFile(filepath.Join(dir, "extra.go"), []byte(extra), 0600); err != nil {
		t.Fatal(err)
	}
	evidence := enforceProcessModel(context.Background(), profile, opts, selection)
	if evidence.CandidateID != selection.Candidates[0].ID || evidence.SourceDigest != selection.Candidates[0].SourceDigest || evidence.BinaryDigest != selection.Candidates[0].BinaryDigest || evidence.Binding != "source-and-binary-digest" {
		t.Fatalf("missing selected executable binding: %+v", evidence)
	}
	if !evidence.Passed || evidence.Processes != 1 {
		t.Fatalf("selected executable did not pass: %+v", evidence)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("unselected Go file was executed: %v", err)
	}
}

func TestProbeRejectsSourceBytesChangedWithoutManifest(t *testing.T) {
	_, opts, selection, dir := selectedProbeFixture(t, "long-running-template")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := snapshotProbeSource(opts.CandidatesDir, t.TempDir(), selection.Candidates[0])
	if err == nil || !strings.Contains(err.Error(), "actual source digest") {
		t.Fatalf("changed source bytes accepted: %v", err)
	}
}

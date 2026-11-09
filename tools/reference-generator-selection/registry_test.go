package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registryPaths(t *testing.T) (workspaceDir, contract, fixtures, candidates, registry string) {
	t.Helper()
	workspaceDir, contract, fixtures, candidates = paths(t)
	registry = filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "registry.json")
	return
}

func TestRegistryResolvesActiveAndRejectsRevoked(t *testing.T) {
	_, contract, _, _, registryPath := registryPaths(t)
	registry, err := LoadRegistry(registryPath, contract)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := ActiveEntry(registry, "template")
	if err != nil || entry.Version != "1" || entry.Status != RegistryActive {
		t.Fatalf("active entry was not resolved: %+v %v", entry, err)
	}
	if _, err := ActiveEntry(registry, "dangerous"); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked generator was not rejected: %v", err)
	}
	if _, err := ActiveEntry(registry, "missing"); err == nil {
		t.Fatal("missing generator was resolved")
	}
}

func TestSelectionWithRegistryRejectsRevokedCandidate(t *testing.T) {
	workspaceDir, contract, fixtures, candidates, registryPath := registryPaths(t)
	receipt, err := SelectWithRegistry(context.Background(), workspaceDir, contract, fixtures, candidates, registryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("registry selection did not qualify any candidate: %+v", receipt)
	}
	qualified := map[string]bool{}
	for _, id := range receipt.Qualified {
		qualified[id] = true
	}
	if !qualified["template"] || !qualified["alternative"] || qualified["dangerous"] {
		t.Fatalf("unexpected qualified set: %+v", receipt.Qualified)
	}
	for _, candidate := range receipt.Candidates {
		if candidate.ID == "dangerous" && !strings.Contains(candidate.RejectReason, "revoked") {
			t.Fatalf("revoked candidate was not rejected by the registry: %+v", candidate)
		}
	}
}

func TestRegistrySourceDigestMismatchIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, candidates, registryPath := registryPaths(t)
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	for i := range registry.Entries {
		if registry.Entries[i].ID == "template" {
			registry.Entries[i].SourceDigest = "sha256:" + strings.Repeat("0", 64)
		}
	}
	changed, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	changedPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(changedPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := SelectWithRegistry(context.Background(), workspaceDir, contract, fixtures, candidates, changedPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range receipt.Candidates {
		if candidate.ID == "template" {
			if candidate.Passed || !strings.Contains(candidate.RejectReason, "source digest mismatch") {
				t.Fatalf("registry source digest mismatch was not rejected: %+v", candidate)
			}
			return
		}
	}
	t.Fatal("template candidate was not evaluated")
}

func TestRegistryDuplicateEntryIsRejected(t *testing.T) {
	_, contract, _, _, registryPath := registryPaths(t)
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	registry.Entries = append(registry.Entries, registry.Entries[0])
	changed, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	changedPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(changedPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(changedPath, contract); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate registry entry was accepted: %v", err)
	}
}

func TestRegistryContractDigestMismatchIsRejected(t *testing.T) {
	_, contract, _, _, registryPath := registryPaths(t)
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	var registry Registry
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	registry.ContractDigest = "sha256:" + strings.Repeat("0", 64)
	changed, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	changedPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(changedPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(changedPath, contract); err == nil || !strings.Contains(err.Error(), "contract digest") {
		t.Fatalf("contract digest mismatch was accepted: %v", err)
	}
}

package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lifecycleRegistryPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata", "registry-lifecycle.json")
}

func writeRegistry(t *testing.T, registry Registry) string {
	t.Helper()
	raw, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func loadLifecycleRegistry(t *testing.T) (Registry, string) {
	t.Helper()
	_, contract, _, _, _ := registryPaths(t)
	path := lifecycleRegistryPath(t)
	registry, err := LoadRegistry(path, contract)
	if err != nil {
		t.Fatal(err)
	}
	return registry, contract
}

func TestRegistryV1ResolvesActiveTipAndDistinguishesRevoked(t *testing.T) {
	registry, _ := loadLifecycleRegistry(t)
	entry, err := ActiveEntry(registry, "template")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != "2" || entry.Status != RegistryActive {
		t.Fatalf("active tip was not resolved: %+v", entry)
	}
	if _, err := ActiveEntry(registry, "dangerous"); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked generator was not distinguished: %v", err)
	}
	if entry, ok := Entry(registry, "template", "1"); !ok || entry.Status != RegistrySuperseded {
		t.Fatalf("superseded version was not retained: %+v %v", entry, ok)
	}
}

func TestRegistryV1RejectsBrokenLifecycles(t *testing.T) {
	base, _ := loadLifecycleRegistry(t)
	find := func(registry Registry, id, version string) int {
		for i := range registry.Entries {
			if registry.Entries[i].ID == id && registry.Entries[i].Version == version {
				return i
			}
		}
		return -1
	}
	tests := []struct {
		name   string
		mutate func(*Registry)
		want   string
	}{
		{
			name: "superseded version without successor",
			mutate: func(registry *Registry) {
				index := find(*registry, "template", "2")
				registry.Entries = append(registry.Entries[:index], registry.Entries[index+1:]...)
			},
			want: "no successor",
		},
		{
			name: "missing supersedes target",
			mutate: func(registry *Registry) {
				registry.Entries[find(*registry, "template", "2")].Supersedes = "9"
			},
			want: "supersedes missing version",
		},
		{
			name: "active version cannot be superseded",
			mutate: func(registry *Registry) {
				registry.Entries[find(*registry, "template", "1")].Status = RegistryActive
			},
			want: "multiple active versions",
		},
		{
			name: "revoked entry needs a reason",
			mutate: func(registry *Registry) {
				registry.Entries[find(*registry, "dangerous", "1")].Reason = ""
			},
			want: "must state a reason",
		},
		{
			name: "replacement cycle",
			mutate: func(registry *Registry) {
				registry.Entries[find(*registry, "template", "1")].Supersedes = "2"
				registry.Entries[find(*registry, "template", "2")].Status = RegistrySuperseded
			},
			want: "cycle",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := base
			registry.Entries = append([]RegistryEntry(nil), base.Entries...)
			tc.mutate(&registry)
			path := writeRegistry(t, registry)
			_, contract, _, _, _ := registryPaths(t)
			_, err := LoadRegistry(path, contract)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("broken lifecycle was accepted: %v", err)
			}
		})
	}
}

func TestSelectionWithLifecycleRegistryRejectsSupersededVersion(t *testing.T) {
	workspaceDir, contract, fixtures, candidates, _ := registryPaths(t)
	receipt, err := SelectWithRegistry(context.Background(), workspaceDir, contract, fixtures, candidates, lifecycleRegistryPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("lifecycle selection did not qualify the current candidate: %+v", receipt)
	}
	for _, candidate := range receipt.Candidates {
		switch candidate.ID {
		case "template":
			if candidate.Passed || !strings.Contains(candidate.RejectReason, "superseded") {
				t.Fatalf("superseded template version was not rejected: %+v", candidate)
			}
		case "alternative":
			if !candidate.Passed {
				t.Fatalf("current alternative version was rejected: %+v", candidate)
			}
		case "dangerous":
			if candidate.Passed || !strings.Contains(candidate.RejectReason, "revoked") {
				t.Fatalf("revoked candidate was not rejected: %+v", candidate)
			}
		}
	}
}

func TestReconcileReportsLifecycleImpacts(t *testing.T) {
	_, contract, _, candidates, _ := registryPaths(t)
	receipt, err := Reconcile(lifecycleRegistryPath(t), contract, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("reconcile passed while a superseded and a revoked candidate remained")
	}
	if receipt.GrantsRuntimeAuthority || receipt.ProvesRealProvider {
		t.Fatal("reconcile claimed authority or a real provider")
	}
	status := map[string]string{}
	for _, impact := range receipt.Candidates {
		status[impact.ID] = impact.Status
	}
	if status["template"] != ReconcileSuperseded || status["dangerous"] != ReconcileRevoked || status["alternative"] != ReconcileCurrent {
		t.Fatalf("unexpected lifecycle statuses: %+v", status)
	}
}

func TestReconcileReportsStaleVersionWhenRegistryOnlyNamesActive(t *testing.T) {
	base, contract := loadLifecycleRegistry(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	stale, err := GenerateVersion(contract, "template", "1", filepath.Join(candidates, "template"))
	if err != nil {
		t.Fatal(err)
	}
	current, err := GenerateVersion(contract, "template", "2", filepath.Join(t.TempDir(), "template-v2"))
	if err != nil {
		t.Fatal(err)
	}
	alternative, err := GenerateVersion(contract, "alternative", "1", filepath.Join(candidates, "alternative"))
	if err != nil {
		t.Fatal(err)
	}
	var registry Registry
	registry.Schema, registry.ContractID, registry.ContractVersion = base.Schema, base.ContractID, base.ContractVersion
	registry.ContractDigest = base.ContractDigest
	registry.Entries = []RegistryEntry{
		{ID: "template", Version: "1", Status: RegistrySuperseded, SourceDigest: stale.SourceDigest},
		{ID: "template", Version: "2", Status: RegistryActive, SourceDigest: current.SourceDigest, Supersedes: "1"},
		{ID: "alternative", Version: "1", Status: RegistryActive, SourceDigest: alternative.SourceDigest},
	}
	path := writeRegistry(t, registry)
	receipt, err := Reconcile(path, contract, candidates)
	if err != nil {
		t.Fatal(err)
	}
	for _, impact := range receipt.Candidates {
		if impact.ID == "template" {
			if impact.Status != ReconcileSuperseded || impact.Action != "remove" {
				t.Fatalf("superseded candidate was not classified: %+v", impact)
			}
		}
		if impact.ID == "alternative" && impact.Status != ReconcileCurrent {
			t.Fatalf("current candidate was not classified: %+v", impact)
		}
	}

	var staleRegistry Registry
	staleRegistry.Schema, staleRegistry.ContractID, staleRegistry.ContractVersion = base.Schema, base.ContractID, base.ContractVersion
	staleRegistry.ContractDigest = base.ContractDigest
	staleRegistry.Entries = []RegistryEntry{
		{ID: "template", Version: "2", Status: RegistryActive, SourceDigest: current.SourceDigest},
		{ID: "alternative", Version: "1", Status: RegistryActive, SourceDigest: alternative.SourceDigest},
	}
	stalePath := writeRegistry(t, staleRegistry)
	staleReceipt, err := Reconcile(stalePath, contract, candidates)
	if err != nil {
		t.Fatal(err)
	}
	for _, impact := range staleReceipt.Candidates {
		if impact.ID == "template" {
			if impact.Status != ReconcileStaleVersion || !impact.CanRegenerate || impact.RegenerationTarget != "2" {
				t.Fatalf("stale template was not classified for regeneration: %+v", impact)
			}
		}
	}
}

func TestReconcileReportsMissingRegisteredCandidate(t *testing.T) {
	_, contract := loadLifecycleRegistry(t)
	receipt, err := Reconcile(lifecycleRegistryPath(t), contract, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed {
		t.Fatal("reconcile passed with no candidate directory")
	}
	if len(receipt.Generators) != 3 {
		t.Fatalf("unexpected generator coverage: %+v", receipt.Generators)
	}
	for _, coverage := range receipt.Generators {
		if coverage.HasCurrent || coverage.Reason == "" {
			t.Fatalf("missing candidate was not reported: %+v", coverage)
		}
	}
}

func TestReconcilePassesAfterRegeneratingCurrentCandidates(t *testing.T) {
	base, contract := loadLifecycleRegistry(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	oldTemplate, err := GenerateVersion(contract, "template", "1", filepath.Join(t.TempDir(), "template-v1"))
	if err != nil {
		t.Fatal(err)
	}
	currentTemplate, err := GenerateVersion(contract, "template", "2", filepath.Join(candidates, "template"))
	if err != nil {
		t.Fatal(err)
	}
	currentAlternative, err := GenerateVersion(contract, "alternative", "1", filepath.Join(candidates, "alternative"))
	if err != nil {
		t.Fatal(err)
	}
	var registry Registry
	registry.Schema, registry.ContractID, registry.ContractVersion = base.Schema, base.ContractID, base.ContractVersion
	registry.ContractDigest = base.ContractDigest
	registry.Entries = []RegistryEntry{
		{ID: "template", Version: "1", Status: RegistrySuperseded, SourceDigest: oldTemplate.SourceDigest},
		{ID: "template", Version: "2", Status: RegistryActive, SourceDigest: currentTemplate.SourceDigest, Supersedes: "1"},
		{ID: "alternative", Version: "1", Status: RegistryActive, SourceDigest: currentAlternative.SourceDigest},
	}
	registryPath := writeRegistry(t, registry)
	receipt, err := Reconcile(registryPath, contract, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("reconcile did not pass for regenerated current candidates: %+v", receipt)
	}
	for _, impact := range receipt.Candidates {
		if impact.Status != ReconcileCurrent || impact.RequiresReselection {
			t.Fatalf("candidate was not current: %+v", impact)
		}
	}
}

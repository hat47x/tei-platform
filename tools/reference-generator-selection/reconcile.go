package generatorselection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ReconcileSchema = "tei.reference.generator-reconcile-receipt/v0"

const (
	ReconcileCurrent       = "current"
	ReconcileStaleVersion  = "stale-version"
	ReconcileStaleSource   = "stale-source"
	ReconcileStaleContract = "stale-contract"
	ReconcileInvalidSource = "invalid-source"
	ReconcileRevoked       = "revoked"
	ReconcileSuperseded    = "superseded"
	ReconcileUnregistered  = "unregistered"
)

// CandidateImpact states whether a candidate directory still matches the
// registry's active version. It is a lifecycle report, not a conformance test.
type CandidateImpact struct {
	ID                  string `json:"id"`
	CandidateVersion    string `json:"candidateVersion"`
	SourceDigest        string `json:"sourceDigest,omitempty"`
	Status              string `json:"status"`
	ActiveVersion       string `json:"activeVersion,omitempty"`
	ActiveSourceDigest  string `json:"activeSourceDigest,omitempty"`
	Action              string `json:"action"`
	CanRegenerate       bool   `json:"canRegenerate"`
	RegenerationTarget  string `json:"regenerationTarget,omitempty"`
	RequiresReselection bool   `json:"requiresReselection"`
	Reason              string `json:"reason,omitempty"`
}

// GeneratorCoverage records whether a registered generator id has a current
// candidate in the inspected directory.
type GeneratorCoverage struct {
	ID             string `json:"id"`
	HasActive      bool   `json:"hasActive"`
	ActiveVersion  string `json:"activeVersion,omitempty"`
	CandidateCount int    `json:"candidateCount"`
	HasCurrent     bool   `json:"hasCurrent"`
	Reason         string `json:"reason,omitempty"`
}

type ReconcileReceipt struct {
	Schema                 string              `json:"schema"`
	ContractID             string              `json:"contractId"`
	ContractVersion        string              `json:"contractVersion"`
	ContractDigest         string              `json:"contractDigest"`
	RegistryDigest         string              `json:"registryDigest"`
	Candidates             []CandidateImpact   `json:"candidates"`
	Generators             []GeneratorCoverage `json:"generators"`
	Passed                 bool                `json:"passed"`
	GrantsRuntimeAuthority bool                `json:"grantsRuntimeAuthority"`
	ProvesRealProvider     bool                `json:"provesRealProvider"`
	Notes                  []string            `json:"notes"`
}

// Reconcile compares candidate directories with the active generator versions
// in a registry. It reports stale, revoked, superseded, and unregistered
// candidates, but it never selects, builds, executes, or accepts them. A
// candidate that requires regeneration must pass selection again afterwards.
func Reconcile(registryPath, contractPath, candidatesDir string) (ReconcileReceipt, error) {
	registry, err := LoadRegistry(registryPath, contractPath)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	registryRaw, err := os.ReadFile(registryPath)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	contractDigest := digest(contractRaw)
	receipt := ReconcileReceipt{
		Schema: ReconcileSchema, ContractID: registry.ContractID, ContractVersion: registry.ContractVersion,
		ContractDigest: contractDigest, RegistryDigest: digest(registryRaw),
		Candidates: []CandidateImpact{}, Generators: []GeneratorCoverage{},
		GrantsRuntimeAuthority: false, ProvesRealProvider: false,
		Notes: []string{
			"registryのactive版と候補directoryを比較するライフサイクル報告です。",
			"候補の適合性・build・実行は確認しません。再生成後も選抜が必要です。",
			"このreceiptは意味の受理・権限・本番配備を証明しません。",
		},
	}
	entries, err := os.ReadDir(candidatesDir)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	counts := map[string]int{}
	currents := map[string]int{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		impact := inspectCandidate(filepath.Join(candidatesDir, entry.Name()), entry.Name(), registry, contractDigest)
		receipt.Candidates = append(receipt.Candidates, impact)
		counts[impact.ID]++
		if impact.Status == ReconcileCurrent {
			currents[impact.ID]++
		}
	}
	sort.Slice(receipt.Candidates, func(i, j int) bool { return receipt.Candidates[i].ID < receipt.Candidates[j].ID })

	passed := len(receipt.Candidates) > 0
	for _, id := range RegistryIDs(registry) {
		coverage := GeneratorCoverage{ID: id, CandidateCount: counts[id], HasCurrent: currents[id] == 1}
		var activeErr error
		if entry, err := ActiveEntry(registry, id); err == nil {
			coverage.HasActive = true
			coverage.ActiveVersion = entry.Version
		} else {
			activeErr = err
		}
		if !coverage.HasCurrent {
			if !coverage.HasActive {
				coverage.Reason = activeErr.Error()
			} else if coverage.CandidateCount == 0 {
				coverage.Reason = "no candidate directory for the registered generator"
			} else if currents[id] > 1 {
				coverage.Reason = "multiple current candidates for one generator"
			} else {
				coverage.Reason = "no current candidate for the active generator version"
			}
			passed = false
		}
		receipt.Generators = append(receipt.Generators, coverage)
	}
	for _, impact := range receipt.Candidates {
		if impact.RequiresReselection {
			passed = false
		}
	}
	receipt.Passed = passed
	return receipt, nil
}

func inspectCandidate(dir, fallbackID string, registry Registry, contractDigest string) CandidateImpact {
	impact := CandidateImpact{ID: fallbackID, Action: "fix-candidate", RequiresReselection: true}
	var manifest CandidateManifest
	if _, err := readJSON(filepath.Join(dir, "candidate.json"), &manifest); err != nil {
		impact.Status = ReconcileInvalidSource
		impact.Reason = "candidate manifest is unreadable: " + err.Error()
		return impact
	}
	impact.ID = manifest.ID
	if impact.ID == "" {
		impact.ID = fallbackID
	}
	impact.CandidateVersion = candidateVersion(manifest)
	impact.SourceDigest = manifest.SourceDigest
	if manifest.Schema != CandidateSchema || manifest.ID == "" || manifest.Source == "" || manifest.SourceDigest == "" {
		impact.Status = ReconcileInvalidSource
		impact.Reason = "candidate identity is incomplete"
		return impact
	}
	relative, err := candidateRelativePath(manifest.Source)
	if err != nil {
		impact.Status = ReconcileInvalidSource
		impact.Reason = "candidate source path is invalid: " + err.Error()
		return impact
	}
	source, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(relative)))
	if err != nil {
		impact.Status = ReconcileInvalidSource
		impact.Reason = "candidate source is unreadable: " + err.Error()
		return impact
	}
	if digest(source) != manifest.SourceDigest {
		impact.Status = ReconcileInvalidSource
		impact.Reason = "candidate source digest does not match its manifest"
		return impact
	}

	active, activeErr := ActiveEntry(registry, manifest.ID)
	if activeErr == nil {
		impact.ActiveVersion = active.Version
		impact.ActiveSourceDigest = active.SourceDigest
	}
	if versionEntry, ok := Entry(registry, manifest.ID, candidateVersion(manifest)); ok {
		switch versionEntry.Status {
		case RegistryRevoked:
			impact.Status = ReconcileRevoked
			impact.Action = "remove"
			impact.Reason = "the candidate version is revoked" + reasonSuffix(versionEntry.Reason)
			return impact
		case RegistrySuperseded:
			impact.Status = ReconcileSuperseded
			impact.Action = "remove"
			impact.Reason = "the candidate version is superseded" + reasonSuffix(versionEntry.Reason)
			return impact
		}
	}
	if activeErr != nil {
		switch {
		case strings.Contains(activeErr.Error(), "revoked"):
			impact.Status = ReconcileRevoked
			impact.Action = "remove"
		case strings.Contains(activeErr.Error(), "superseded"):
			impact.Status = ReconcileSuperseded
			impact.Action = "remove"
		case strings.Contains(activeErr.Error(), "not registered"):
			impact.Status = ReconcileUnregistered
			impact.Action = "register"
		default:
			impact.Status = ReconcileInvalidSource
		}
		impact.Reason = activeErr.Error()
		return impact
	}
	if manifest.Generation != nil && manifest.Generation.ContractDigest != contractDigest {
		impact.Status = ReconcileStaleContract
		impact.Action = "regenerate"
		impact.Reason = "generation provenance targets a different contract digest"
		return impact
	}
	if manifest.Generator != nil && manifest.Generator.ContractDigest != contractDigest {
		impact.Status = ReconcileStaleContract
		impact.Action = "regenerate"
		impact.Reason = "generator provenance targets a different contract digest"
		return impact
	}
	if candidateVersion(manifest) != active.Version {
		impact.Status = ReconcileStaleVersion
		impact.Action = "regenerate"
		impact.Reason = fmt.Sprintf("candidate version %s is not the active version %s",
			candidateVersion(manifest), active.Version)
		if manifest.Generator != nil {
			if _, ok := generatorTemplates[manifest.Generator.Strategy]; ok {
				impact.CanRegenerate = true
				impact.RegenerationTarget = active.Version
			}
		}
		return impact
	}
	if manifest.SourceDigest != active.SourceDigest {
		impact.Status = ReconcileStaleSource
		impact.Action = "regenerate"
		impact.Reason = "candidate source digest does not match the active registry entry"
		return impact
	}
	impact.Status = ReconcileCurrent
	impact.Action = "none"
	impact.RequiresReselection = false
	return impact
}

func reasonSuffix(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return ": " + reason
}

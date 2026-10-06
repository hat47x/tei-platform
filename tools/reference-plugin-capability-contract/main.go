package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type AuthorityScope string

const (
	ScopeAuthoringTool      AuthorityScope = "authoring-tool"
	ScopeBuildTool          AuthorityScope = "build-tool"
	ScopeApplicationRuntime AuthorityScope = "application-runtime"
	ScopeTargetLocal        AuthorityScope = "target-local"
)

func (s AuthorityScope) Valid() bool {
	switch s {
	case ScopeAuthoringTool, ScopeBuildTool, ScopeApplicationRuntime, ScopeTargetLocal:
		return true
	default:
		return false
	}
}

type Contract struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	RequestSemantics   string `json:"requestSemantics"`
	ResultSemantics    string `json:"resultSemantics"`
	FailureSemantics   string `json:"failureSemantics"`
	ConformanceProfile string `json:"conformanceProfile"`
}

type Requirement struct {
	ContractID      string `json:"contractId"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

type Provider struct {
	PluginID            string         `json:"pluginId"`
	PluginVersion       string         `json:"pluginVersion"`
	ImplementationID    string         `json:"implementationId"`
	ArtifactDigest      string         `json:"artifactDigest,omitempty"`
	Scope               AuthorityScope `json:"scope"`
	ContractID          string         `json:"contractId"`
	ContractVersion     string         `json:"contractVersion"`
	ContractFingerprint string         `json:"contractFingerprint"`
	ConformanceProfile  string         `json:"conformanceProfile"`
}

type ConformanceEvidence struct {
	PluginID            string `json:"pluginId"`
	PluginVersion       string `json:"pluginVersion"`
	ImplementationID    string `json:"implementationId"`
	ContractFingerprint string `json:"contractFingerprint"`
	ConformanceProfile  string `json:"conformanceProfile"`
	EvidenceFingerprint string `json:"evidenceFingerprint"`
}

type Wiring struct {
	ContractID       string `json:"contractId"`
	PluginID         string `json:"pluginId"`
	PluginVersion    string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
}

type ResolveInput struct {
	Scope       AuthorityScope        `json:"scope"`
	Requirement Requirement           `json:"requirement"`
	Contracts   []Contract            `json:"contracts"`
	Providers   []Provider            `json:"providers"`
	Evidence    []ConformanceEvidence `json:"evidence"`
	Wirings     []Wiring              `json:"wirings,omitempty"`
}

type Resolution struct {
	Scope                 AuthorityScope `json:"scope"`
	Contract              Contract       `json:"contract"`
	ContractFingerprint   string         `json:"contractFingerprint"`
	Provider              Provider       `json:"provider"`
	ResolutionFingerprint string         `json:"resolutionFingerprint"`
	EvidenceFingerprint   string         `json:"evidenceFingerprint"`
}

func main() {}

func resolveCapability(input ResolveInput) (Resolution, error) {
	if !input.Scope.Valid() {
		return Resolution{}, fmt.Errorf("invalid authority scope %q", input.Scope)
	}
	if input.Requirement.ContractID == "" || input.Requirement.CompatibleMajor < 0 {
		return Resolution{}, fmt.Errorf("invalid contract requirement")
	}

	contracts, err := compatibleContracts(input.Contracts, input.Requirement)
	if err != nil {
		return Resolution{}, err
	}
	wiring, err := wiringForContract(input.Wirings, input.Requirement.ContractID)
	if err != nil {
		return Resolution{}, err
	}

	for _, contract := range contracts {
		contractFingerprint := fingerprint(contract)
		providers, evidenceByProvider, err := conformantProviders(input, contract, contractFingerprint)
		if err != nil {
			return Resolution{}, err
		}

		if wiring != nil {
			for _, provider := range providers {
				if providerMatchesWiring(provider, *wiring) {
					return buildResolution(input.Scope, contract, contractFingerprint, provider, evidenceByProvider[providerKey(provider)]), nil
				}
			}
			continue
		}

		switch len(providers) {
		case 0:
			continue
		case 1:
			provider := providers[0]
			return buildResolution(input.Scope, contract, contractFingerprint, provider, evidenceByProvider[providerKey(provider)]), nil
		default:
			return Resolution{}, fmt.Errorf("capability provider is ambiguous for contract %s@%s", contract.ID, contract.Version)
		}
	}

	if wiring != nil {
		return Resolution{}, fmt.Errorf("wired provider %s@%s/%s is not conformant for contract %s", wiring.PluginID, wiring.PluginVersion, wiring.ImplementationID, input.Requirement.ContractID)
	}
	return Resolution{}, fmt.Errorf("no conformant provider for contract %s", input.Requirement.ContractID)
}

func compatibleContracts(contracts []Contract, requirement Requirement) ([]Contract, error) {
	var matches []Contract
	seen := map[string]struct{}{}
	for _, contract := range contracts {
		if contract.ID == "" || contract.Version == "" || contract.RequestSemantics == "" ||
			contract.ResultSemantics == "" || contract.FailureSemantics == "" || contract.ConformanceProfile == "" {
			return nil, fmt.Errorf("invalid capability contract %+v", contract)
		}
		version, err := parseVersion(contract.Version)
		if err != nil {
			return nil, fmt.Errorf("contract %s: %w", contract.ID, err)
		}
		key := contract.ID + "@" + contract.Version
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf("contract %s is declared more than once", key)
		}
		seen[key] = struct{}{}
		if contract.ID == requirement.ContractID && version[0] == requirement.CompatibleMajor {
			matches = append(matches, contract)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no capability contract %s compatible with major %d", requirement.ContractID, requirement.CompatibleMajor)
	}
	sort.Slice(matches, func(i, j int) bool {
		return compareVersion(matches[i].Version, matches[j].Version) > 0
	})
	return matches, nil
}

func wiringForContract(wirings []Wiring, contractID string) (*Wiring, error) {
	var selected *Wiring
	for i := range wirings {
		if wirings[i].ContractID != contractID {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("contract %s has multiple explicit wirings", contractID)
		}
		if wirings[i].PluginID == "" || wirings[i].PluginVersion == "" || wirings[i].ImplementationID == "" {
			return nil, fmt.Errorf("contract %s has incomplete explicit wiring", contractID)
		}
		copy := wirings[i]
		selected = &copy
	}
	return selected, nil
}

func conformantProviders(input ResolveInput, contract Contract, contractFingerprint string) ([]Provider, map[string]ConformanceEvidence, error) {
	var providers []Provider
	evidenceByProvider := map[string]ConformanceEvidence{}
	seen := map[string]struct{}{}
	for _, provider := range input.Providers {
		if provider.Scope != input.Scope ||
			provider.ContractID != contract.ID ||
			provider.ContractVersion != contract.Version ||
			provider.ContractFingerprint != contractFingerprint ||
			provider.ConformanceProfile != contract.ConformanceProfile {
			continue
		}
		if provider.PluginID == "" || provider.PluginVersion == "" || provider.ImplementationID == "" {
			continue
		}
		key := providerKey(provider)
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, fmt.Errorf("provider %s is declared more than once", key)
		}
		seen[key] = struct{}{}
		evidence, ok := matchingEvidence(input.Evidence, provider, contractFingerprint, contract.ConformanceProfile)
		if !ok {
			continue
		}
		providers = append(providers, provider)
		evidenceByProvider[key] = evidence
	}
	sort.Slice(providers, func(i, j int) bool {
		return providerKey(providers[i]) < providerKey(providers[j])
	})
	return providers, evidenceByProvider, nil
}

func providerMatchesWiring(provider Provider, wiring Wiring) bool {
	return provider.PluginID == wiring.PluginID &&
		provider.PluginVersion == wiring.PluginVersion &&
		provider.ImplementationID == wiring.ImplementationID
}

func buildResolution(scope AuthorityScope, contract Contract, contractFingerprint string, provider Provider, evidence ConformanceEvidence) Resolution {
	semantic := struct {
		Scope               AuthorityScope `json:"scope"`
		ContractFingerprint string         `json:"contractFingerprint"`
		PluginID            string         `json:"pluginId"`
		PluginVersion       string         `json:"pluginVersion"`
		ImplementationID    string         `json:"implementationId"`
		ArtifactDigest      string         `json:"artifactDigest,omitempty"`
	}{
		Scope:               scope,
		ContractFingerprint: contractFingerprint,
		PluginID:            provider.PluginID,
		PluginVersion:       provider.PluginVersion,
		ImplementationID:    provider.ImplementationID,
		ArtifactDigest:      provider.ArtifactDigest,
	}
	resolutionFingerprint := fingerprint(semantic)
	evidenceFingerprint := fingerprint(struct {
		ResolutionFingerprint string `json:"resolutionFingerprint"`
		EvidenceFingerprint   string `json:"evidenceFingerprint"`
	}{
		ResolutionFingerprint: resolutionFingerprint,
		EvidenceFingerprint:   evidence.EvidenceFingerprint,
	})
	return Resolution{
		Scope:                 scope,
		Contract:              contract,
		ContractFingerprint:   contractFingerprint,
		Provider:              provider,
		ResolutionFingerprint: resolutionFingerprint,
		EvidenceFingerprint:   evidenceFingerprint,
	}
}

func matchingEvidence(all []ConformanceEvidence, provider Provider, contractFingerprint, profile string) (ConformanceEvidence, bool) {
	var matches []ConformanceEvidence
	for _, evidence := range all {
		if evidence.PluginID == provider.PluginID &&
			evidence.PluginVersion == provider.PluginVersion &&
			evidence.ImplementationID == provider.ImplementationID &&
			evidence.ContractFingerprint == contractFingerprint &&
			evidence.ConformanceProfile == profile &&
			evidence.EvidenceFingerprint != "" {
			matches = append(matches, evidence)
		}
	}
	if len(matches) != 1 {
		return ConformanceEvidence{}, false
	}
	return matches[0], true
}

func providerKey(provider Provider) string {
	return strings.Join([]string{provider.PluginID, provider.PluginVersion, provider.ImplementationID, provider.ArtifactDigest}, "\x00")
}

func parseVersion(value string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version %q must be major.minor.patch", value)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("version %q contains invalid component", value)
		}
		out[i] = n
	}
	return out, nil
}

func compareVersion(left, right string) int {
	a, _ := parseVersion(left)
	b, _ := parseVersion(right)
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
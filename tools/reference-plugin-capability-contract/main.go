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
	ArtifactDigest      string `json:"artifactDigest,omitempty"`
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

const (
	capabilityResolutionPolicy           = "highest-compatible-complete-search/explicit-wiring/v0"
	capabilityResolutionLockSchema       = "tei.reference.capability-resolution-lock/v0"
	capabilityResolutionCanonicalization = "json-struct-field-order/v0"
)

type Resolution struct {
	Scope                 AuthorityScope `json:"scope"`
	ResolutionPolicy      string         `json:"resolutionPolicy"`
	Contract              Contract       `json:"contract"`
	ContractFingerprint   string         `json:"contractFingerprint"`
	Provider              Provider       `json:"provider"`
	ResolutionFingerprint string         `json:"resolutionFingerprint"`
	EvidenceFingerprint   string         `json:"evidenceFingerprint"`
}

type CapabilityResolutionLock struct {
	Schema                 string         `json:"schema"`
	Canonicalization       string         `json:"canonicalization"`
	ResolutionPolicy       string         `json:"resolutionPolicy"`
	Scope                  AuthorityScope `json:"scope"`
	Requirement            Requirement    `json:"requirement"`
	Wiring                 *Wiring        `json:"wiring,omitempty"`
	Contract               Contract       `json:"contract"`
	ContractFingerprint    string         `json:"contractFingerprint"`
	Provider               Provider       `json:"provider"`
	ResolutionFingerprint  string         `json:"resolutionFingerprint"`
	LockFingerprint        string         `json:"lockFingerprint"`
}

func main() {}

func resolveCapability(input ResolveInput) (Resolution, error) {
	if err := validateResolveInput(input); err != nil {
		return Resolution{}, err
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
		identityKey := providerIdentityKey(provider)
		if _, duplicate := seen[identityKey]; duplicate {
			return nil, nil, fmt.Errorf(
				"logical provider %s is declared more than once",
				identityKey,
			)
		}
		seen[identityKey] = struct{}{}
		evidence, ok := matchingEvidence(input.Evidence, provider, contractFingerprint, contract.ConformanceProfile)
		if !ok {
			continue
		}
		providers = append(providers, provider)
		evidenceByProvider[providerKey(provider)] = evidence
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
	resolutionFingerprint := capabilityResolutionFingerprint(
		scope,
		contract,
		contractFingerprint,
		provider,
	)
	evidenceFingerprint := fingerprint(struct {
		ResolutionFingerprint string `json:"resolutionFingerprint"`
		EvidenceFingerprint   string `json:"evidenceFingerprint"`
	}{
		ResolutionFingerprint: resolutionFingerprint,
		EvidenceFingerprint:   evidence.EvidenceFingerprint,
	})
	return Resolution{
		Scope:                 scope,
		ResolutionPolicy:      capabilityResolutionPolicy,
		Contract:              contract,
		ContractFingerprint:   contractFingerprint,
		Provider:              provider,
		ResolutionFingerprint: resolutionFingerprint,
		EvidenceFingerprint:   evidenceFingerprint,
	}
}

func capabilityResolutionFingerprint(
	scope AuthorityScope,
	contract Contract,
	contractFingerprint string,
	provider Provider,
) string {
	return fingerprint(struct {
		ResolutionPolicy      string         `json:"resolutionPolicy"`
		Scope                 AuthorityScope `json:"scope"`
		ContractID            string         `json:"contractId"`
		ContractVersion       string         `json:"contractVersion"`
		ContractFingerprint   string         `json:"contractFingerprint"`
		PluginID              string         `json:"pluginId"`
		PluginVersion         string         `json:"pluginVersion"`
		ImplementationID      string         `json:"implementationId"`
		ArtifactDigest        string         `json:"artifactDigest,omitempty"`
	}{
		ResolutionPolicy:    capabilityResolutionPolicy,
		Scope:               scope,
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: contractFingerprint,
		PluginID:            provider.PluginID,
		PluginVersion:       provider.PluginVersion,
		ImplementationID:    provider.ImplementationID,
		ArtifactDigest:      provider.ArtifactDigest,
	})
}

func validateResolveInput(input ResolveInput) error {
	if !input.Scope.Valid() {
		return fmt.Errorf("invalid authority scope %q", input.Scope)
	}
	if input.Requirement.ContractID == "" || input.Requirement.CompatibleMajor < 0 {
		return fmt.Errorf("invalid contract requirement")
	}
	for i, provider := range input.Providers {
		if provider.PluginID == "" || provider.PluginVersion == "" ||
			provider.ImplementationID == "" || !provider.Scope.Valid() ||
			provider.ContractID == "" || provider.ContractVersion == "" ||
			!validSHA256Fingerprint(provider.ContractFingerprint) ||
			!validSHA256Fingerprint(provider.ArtifactDigest) ||
			provider.ConformanceProfile == "" {
			return fmt.Errorf("provider[%d] is incomplete or invalid", i)
		}
		if _, err := parseVersion(provider.PluginVersion); err != nil {
			return fmt.Errorf("provider[%d] plugin version: %w", i, err)
		}
		if _, err := parseVersion(provider.ContractVersion); err != nil {
			return fmt.Errorf("provider[%d] contract version: %w", i, err)
		}
	}
	for i, evidence := range input.Evidence {
		if evidence.PluginID == "" || evidence.PluginVersion == "" ||
			evidence.ImplementationID == "" ||
			!validSHA256Fingerprint(evidence.ArtifactDigest) ||
			!validSHA256Fingerprint(evidence.ContractFingerprint) ||
			evidence.ConformanceProfile == "" ||
			!validSHA256Fingerprint(evidence.EvidenceFingerprint) {
			return fmt.Errorf("conformance evidence[%d] is incomplete or invalid", i)
		}
		if _, err := parseVersion(evidence.PluginVersion); err != nil {
			return fmt.Errorf("conformance evidence[%d] plugin version: %w", i, err)
		}
	}
	for i, wiring := range input.Wirings {
		if wiring.ContractID == "" || wiring.PluginID == "" ||
			wiring.PluginVersion == "" || wiring.ImplementationID == "" {
			return fmt.Errorf("wiring[%d] is incomplete", i)
		}
		if _, err := parseVersion(wiring.PluginVersion); err != nil {
			return fmt.Errorf("wiring[%d] plugin version: %w", i, err)
		}
	}
	return nil
}

func buildCapabilityResolutionLock(
	input ResolveInput,
	resolution Resolution,
) (CapabilityResolutionLock, error) {
	if err := validateResolveInput(input); err != nil {
		return CapabilityResolutionLock{}, err
	}
	requirement := input.Requirement
	wiring, err := wiringForContract(input.Wirings, requirement.ContractID)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	current, err := resolveCapability(input)
	if err != nil {
		return CapabilityResolutionLock{}, err
	}
	if current.ResolutionFingerprint != resolution.ResolutionFingerprint ||
		current.Contract != resolution.Contract ||
		current.Provider != resolution.Provider {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolution does not match current deterministic capability resolution",
		)
	}
	if resolution.ResolutionPolicy != capabilityResolutionPolicy {
		return CapabilityResolutionLock{}, fmt.Errorf(
			"resolution policy %q does not match current policy %q",
			resolution.ResolutionPolicy,
			capabilityResolutionPolicy,
		)
	}
	if !resolution.Scope.Valid() ||
		resolution.Contract.ID != requirement.ContractID ||
		!validSHA256Fingerprint(resolution.ContractFingerprint) ||
		resolution.ContractFingerprint != fingerprint(resolution.Contract) ||
		!validSHA256Fingerprint(resolution.ResolutionFingerprint) {
		return CapabilityResolutionLock{}, fmt.Errorf("resolution is not exact enough to lock")
	}
	version, err := parseVersion(resolution.Contract.Version)
	if err != nil || version[0] != requirement.CompatibleMajor {
		return CapabilityResolutionLock{}, fmt.Errorf("resolved contract is incompatible with requirement")
	}
	expectedResolutionFingerprint := capabilityResolutionFingerprint(
		resolution.Scope,
		resolution.Contract,
		resolution.ContractFingerprint,
		resolution.Provider,
	)
	if resolution.ResolutionFingerprint != expectedResolutionFingerprint {
		return CapabilityResolutionLock{}, fmt.Errorf("resolution fingerprint is stale")
	}
	lock := CapabilityResolutionLock{
		Schema:                capabilityResolutionLockSchema,
		Canonicalization:      capabilityResolutionCanonicalization,
		ResolutionPolicy:      capabilityResolutionPolicy,
		Scope:                 resolution.Scope,
		Requirement:           requirement,
		Wiring:                wiring,
		Contract:              resolution.Contract,
		ContractFingerprint:   resolution.ContractFingerprint,
		Provider:              resolution.Provider,
		ResolutionFingerprint: resolution.ResolutionFingerprint,
	}
	lock.LockFingerprint = capabilityResolutionLockFingerprint(lock)
	if err := validateCapabilityResolutionLock(lock); err != nil {
		return CapabilityResolutionLock{}, err
	}
	return lock, nil
}

func verifyCapabilityResolutionLockReplay(
	lock CapabilityResolutionLock,
	input ResolveInput,
) error {
	if err := validateCapabilityResolutionLock(lock); err != nil {
		return err
	}
	replayInput := input
	replayInput.Evidence = nil
	if err := validateResolveInput(replayInput); err != nil {
		return err
	}
	if input.Scope != lock.Scope {
		return fmt.Errorf(
			"capability lock scope %q differs from current scope %q",
			lock.Scope,
			input.Scope,
		)
	}
	if input.Requirement != lock.Requirement {
		return fmt.Errorf(
			"capability lock requirement %+v differs from current requirement %+v",
			lock.Requirement,
			input.Requirement,
		)
	}
	currentWiring, err := wiringForContract(input.Wirings, input.Requirement.ContractID)
	if err != nil {
		return err
	}
	switch {
	case lock.Wiring == nil && currentWiring != nil:
		return fmt.Errorf("capability lock was resolved without explicit wiring but current input has wiring")
	case lock.Wiring != nil && currentWiring == nil:
		return fmt.Errorf("capability lock requires explicit wiring but current input has none")
	case lock.Wiring != nil && currentWiring != nil && *lock.Wiring != *currentWiring:
		return fmt.Errorf(
			"capability lock wiring %+v differs from current wiring %+v",
			*lock.Wiring,
			*currentWiring,
		)
	}

	var exactContracts []Contract
	for _, contract := range input.Contracts {
		if contract.ID == lock.Contract.ID && contract.Version == lock.Contract.Version {
			exactContracts = append(exactContracts, contract)
		}
	}
	if len(exactContracts) != 1 {
		return fmt.Errorf(
			"capability lock requires exactly one contract declaration %s@%s",
			lock.Contract.ID,
			lock.Contract.Version,
		)
	}
	exactContract := exactContracts[0]
	if fingerprint(exactContract) != lock.ContractFingerprint ||
		exactContract != lock.Contract {
		return fmt.Errorf(
			"capability contract %s@%s drifted from lock",
			lock.Contract.ID,
			lock.Contract.Version,
		)
	}

	var exactProviders []Provider
	for _, provider := range input.Providers {
		if provider == lock.Provider {
			exactProviders = append(exactProviders, provider)
		}
	}
	if len(exactProviders) != 1 {
		return fmt.Errorf(
			"capability lock requires exact provider %s@%s/%s with artifact %q",
			lock.Provider.PluginID,
			lock.Provider.PluginVersion,
			lock.Provider.ImplementationID,
			lock.Provider.ArtifactDigest,
		)
	}

	expectedResolutionFingerprint := capabilityResolutionFingerprint(
		lock.Scope,
		lock.Contract,
		lock.ContractFingerprint,
		lock.Provider,
	)
	if expectedResolutionFingerprint != lock.ResolutionFingerprint {
		return fmt.Errorf("capability lock resolution fingerprint is stale")
	}
	return nil
}

func verifyCapabilityResolutionLockEvidence(
	lock CapabilityResolutionLock,
	input ResolveInput,
) (string, error) {
	if err := verifyCapabilityResolutionLockReplay(lock, input); err != nil {
		return "", err
	}
	if err := validateResolveInput(input); err != nil {
		return "", err
	}
	evidence, ok := matchingEvidence(
		input.Evidence,
		lock.Provider,
		lock.ContractFingerprint,
		lock.Contract.ConformanceProfile,
	)
	if !ok {
		return "", fmt.Errorf(
			"capability lock has no unique current Conformance Evidence for %s@%s/%s",
			lock.Provider.PluginID,
			lock.Provider.PluginVersion,
			lock.Provider.ImplementationID,
		)
	}
	return fingerprint(struct {
		ResolutionFingerprint string `json:"resolutionFingerprint"`
		EvidenceFingerprint   string `json:"evidenceFingerprint"`
	}{
		ResolutionFingerprint: lock.ResolutionFingerprint,
		EvidenceFingerprint:   evidence.EvidenceFingerprint,
	}), nil
}

func validateCapabilityResolutionLock(lock CapabilityResolutionLock) error {
	if lock.Schema != capabilityResolutionLockSchema {
		return fmt.Errorf("capability lock schema %q is not supported", lock.Schema)
	}
	if lock.Canonicalization != capabilityResolutionCanonicalization {
		return fmt.Errorf(
			"capability lock canonicalization %q is not supported",
			lock.Canonicalization,
		)
	}
	if lock.ResolutionPolicy != capabilityResolutionPolicy {
		return fmt.Errorf(
			"capability lock resolution policy %q is not supported",
			lock.ResolutionPolicy,
		)
	}
	if !lock.Scope.Valid() ||
		lock.Requirement.ContractID == "" ||
		lock.Requirement.CompatibleMajor < 0 {
		return fmt.Errorf("capability lock has invalid scope or requirement")
	}
	if lock.Wiring != nil {
		if lock.Wiring.ContractID != lock.Requirement.ContractID ||
			lock.Wiring.PluginID == "" ||
			lock.Wiring.PluginVersion == "" ||
			lock.Wiring.ImplementationID == "" {
			return fmt.Errorf("capability lock has invalid explicit wiring")
		}
		if _, err := parseVersion(lock.Wiring.PluginVersion); err != nil {
			return fmt.Errorf("capability lock wiring plugin version: %w", err)
		}
	}
	if lock.Wiring != nil && !providerMatchesWiring(lock.Provider, *lock.Wiring) {
		return fmt.Errorf("capability lock wiring does not select locked provider")
	}
	if lock.Contract.ID != lock.Requirement.ContractID ||
		lock.Provider.Scope != lock.Scope ||
		lock.Provider.ContractID != lock.Contract.ID ||
		lock.Provider.ContractVersion != lock.Contract.Version ||
		lock.Provider.ContractFingerprint != lock.ContractFingerprint ||
		lock.Provider.ConformanceProfile != lock.Contract.ConformanceProfile {
		return fmt.Errorf("capability lock declarations are internally inconsistent")
	}
	version, err := parseVersion(lock.Contract.Version)
	if err != nil || version[0] != lock.Requirement.CompatibleMajor {
		return fmt.Errorf("capability lock contract is incompatible with requirement")
	}
	if !validSHA256Fingerprint(lock.ContractFingerprint) ||
		lock.ContractFingerprint != fingerprint(lock.Contract) ||
		!validSHA256Fingerprint(lock.ResolutionFingerprint) ||
		!validSHA256Fingerprint(lock.LockFingerprint) {
		return fmt.Errorf("capability lock fingerprints are invalid")
	}
	if !validSHA256Fingerprint(lock.Provider.ArtifactDigest) {
		return fmt.Errorf("capability lock provider requires exact artifact digest")
	}
	if lock.Provider.PluginID == "" ||
		lock.Provider.PluginVersion == "" ||
		lock.Provider.ImplementationID == "" {
		return fmt.Errorf("capability lock provider identity is incomplete")
	}
	if _, err := parseVersion(lock.Provider.PluginVersion); err != nil {
		return fmt.Errorf("capability lock provider plugin version: %w", err)
	}
	if lock.Contract.RequestSemantics == "" ||
		lock.Contract.ResultSemantics == "" ||
		lock.Contract.FailureSemantics == "" ||
		lock.Contract.ConformanceProfile == "" {
		return fmt.Errorf("capability lock contract is incomplete")
	}
	expectedResolutionFingerprint := capabilityResolutionFingerprint(
		lock.Scope,
		lock.Contract,
		lock.ContractFingerprint,
		lock.Provider,
	)
	if expectedResolutionFingerprint != lock.ResolutionFingerprint {
		return fmt.Errorf("capability lock resolution fingerprint is stale")
	}
	if capabilityResolutionLockFingerprint(lock) != lock.LockFingerprint {
		return fmt.Errorf("capability lock fingerprint is stale")
	}
	return nil
}

func capabilityResolutionLockFingerprint(lock CapabilityResolutionLock) string {
	lock.LockFingerprint = ""
	return fingerprint(lock)
}

func validSHA256Fingerprint(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func matchingEvidence(all []ConformanceEvidence, provider Provider, contractFingerprint, profile string) (ConformanceEvidence, bool) {
	var matches []ConformanceEvidence
	for _, evidence := range all {
		if evidence.PluginID == provider.PluginID &&
			evidence.PluginVersion == provider.PluginVersion &&
			evidence.ImplementationID == provider.ImplementationID &&
			evidence.ArtifactDigest == provider.ArtifactDigest &&
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

func providerIdentityKey(provider Provider) string {
	return strings.Join(
		[]string{provider.PluginID, provider.PluginVersion, provider.ImplementationID},
		"\x00",
	)
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

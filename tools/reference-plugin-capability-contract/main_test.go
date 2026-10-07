package main

import (
	"strings"
	"testing"
)

func sourceAdapterContract() Contract {
	return Contract{
		ID:                 "authoring.source-adapter.xead",
		Version:            "1.0.0",
		RequestSemantics:   "exact-source-bytes",
		ResultSemantics:    "source-faithful-inventory-with-unresolved",
		FailureSemantics:   "unsupported-or-invalid-is-not-success",
		ConformanceProfile: "tei.reference.source-adapter/v0",
	}
}

func sourceAdapterProvider(contract Contract, scope AuthorityScope, pluginID, implementation string) Provider {
	return Provider{
		PluginID:            pluginID,
		PluginVersion:       "1.0.0",
		ImplementationID:    implementation,
		ArtifactDigest:      "sha256:" + strings.Repeat("a", 64),
		Scope:               scope,
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile:  contract.ConformanceProfile,
	}
}

func evidenceFor(provider Provider, contract Contract, marker string) ConformanceEvidence {
	return ConformanceEvidence{
		PluginID:                     provider.PluginID,
		PluginVersion:                provider.PluginVersion,
		ImplementationID:             provider.ImplementationID,
		ArtifactDigest:               provider.ArtifactDigest,
		SourceDeclarationFingerprint: provider.SourceDeclarationFingerprint,
		ContractFingerprint:          fingerprint(contract),
		ConformanceProfile:           contract.ConformanceProfile,
		EvidenceFingerprint:          fingerprint(marker),
	}
}

func TestSourceAdapterContractDoesNotRequireCoreCapabilityKind(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	resolution, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-a")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Contract.ID != "authoring.source-adapter.xead" {
		t.Fatalf("contract=%q", resolution.Contract.ID)
	}
	if resolution.Provider.ImplementationID != "xead.reader.v0" {
		t.Fatalf("provider=%+v", resolution.Provider)
	}
}

func TestProviderSelfClaimWithoutConformanceEvidenceFailsClosed(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("expected missing conformance evidence, got %v", err)
	}
}

func TestRuntimeProviderCannotSatisfyAuthoringToolRequirement(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeApplicationRuntime, "runtime.reference", "runtime.reader")
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-runtime")},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("wrong authority scope was accepted: %v", err)
	}
}

func TestAmbiguousProvidersRequireExplicitWiring(t *testing.T) {
	contract := sourceAdapterContract()
	left := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.left", "xead.reader")
	right := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.right", "xead.reader")
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{right, left},
		Evidence: []ConformanceEvidence{
			evidenceFor(left, contract, "evidence-left"),
			evidenceFor(right, contract, "evidence-right"),
		},
	}
	if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous providers were silently selected: %v", err)
	}
	input.Wirings = []Wiring{{
		ContractID:       contract.ID,
		PluginID:         right.PluginID,
		PluginVersion:    right.PluginVersion,
		ImplementationID: right.ImplementationID,
	}}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.PluginID != right.PluginID {
		t.Fatalf("provider=%+v", resolution.Provider)
	}
}

func TestHigherCompatibleContractWithoutProviderFallsBackToCompleteSolution(t *testing.T) {
	v1 := sourceAdapterContract()
	v11 := sourceAdapterContract()
	v11.Version = "1.1.0"
	provider := sourceAdapterProvider(v1, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	resolution, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1},
		Contracts:   []Contract{v1, v11},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, v1, "evidence-v1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Contract.Version != "1.0.0" {
		t.Fatalf("contract=%s want=1.0.0", resolution.Contract.Version)
	}
}

func TestCatalogOrderingDoesNotChangeResolutionIdentity(t *testing.T) {
	v1 := sourceAdapterContract()
	v11 := sourceAdapterContract()
	v11.Version = "1.1.0"
	provider := sourceAdapterProvider(v11, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v1")
	evidence := evidenceFor(provider, v11, "evidence-current")
	left, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1},
		Contracts:   []Contract{v1, v11},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidence},
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1},
		Contracts:   []Contract{v11, v1},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidence},
	})
	if err != nil {
		t.Fatal(err)
	}
	if left.Contract.Version != "1.1.0" || right.Contract.Version != "1.1.0" {
		t.Fatalf("highest complete compatible contract was not selected")
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("catalog ordering changed resolution identity")
	}
}

func TestEvidenceRefreshDoesNotChangeResolutionIdentity(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-old")},
	}
	left, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "evidence-new")}
	right, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("evidence refresh changed resolution identity")
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("evidence refresh was not observable")
	}
}

func TestContractFingerprintMismatchFailsClosed(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	provider.ContractFingerprint = "sha256:" + strings.Repeat("0", 64)
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-a")},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("contract drift was accepted: %v", err)
	}
}


func TestCapabilityResolutionLockPinsExactProviderAcrossCatalogGrowth(t *testing.T) {
	v1 := sourceAdapterContract()
	providerV1 := sourceAdapterProvider(
		v1,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v1",
	)
	baseInput := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1},
		Contracts:   []Contract{v1},
		Providers:   []Provider{providerV1},
		Evidence:    []ConformanceEvidence{evidenceFor(providerV1, v1, "evidence-v1")},
	}
	resolvedV1, err := resolveCapability(baseInput)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(baseInput, resolvedV1)
	if err != nil {
		t.Fatal(err)
	}
	if !validSHA256Fingerprint(lock.LockFingerprint) {
		t.Fatalf("lock fingerprint=%q", lock.LockFingerprint)
	}

	v11 := sourceAdapterContract()
	v11.Version = "1.1.0"
	providerV11 := sourceAdapterProvider(
		v11,
		ScopeAuthoringTool,
		"reference.compat.xead.source.next",
		"xead.reader.v11",
	)
	grown := baseInput
	grown.Contracts = []Contract{v11, v1}
	grown.Providers = []Provider{providerV11, providerV1}
	grown.Evidence = []ConformanceEvidence{
		evidenceFor(providerV11, v11, "evidence-v11"),
		evidenceFor(providerV1, v1, "evidence-v1"),
	}

	fresh, err := resolveCapability(grown)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Contract.Version != "1.1.0" {
		t.Fatalf("fresh resolution contract=%s want=1.1.0", fresh.Contract.Version)
	}
	if fresh.ResolutionFingerprint == lock.ResolutionFingerprint {
		t.Fatal("fresh resolution unexpectedly matched locked resolution")
	}
	if err := verifyCapabilityResolutionLockReplay(lock, grown); err != nil {
		t.Fatalf("exact lock replay failed after catalog growth: %v", err)
	}
	if _, err := verifyCapabilityResolutionLockEvidence(lock, grown); err != nil {
		t.Fatalf("locked provider current Evidence was not accepted: %v", err)
	}
}

func TestCapabilityResolutionLockRejectsContractAndArtifactDrift(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-current")},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	driftedContract := contract
	driftedContract.ResultSemantics = "lossy-inventory"
	contractDrift := input
	contractDrift.Contracts = []Contract{driftedContract}
	if err := verifyCapabilityResolutionLockReplay(lock, contractDrift); err == nil {
		t.Fatal("same-version contract drift was accepted by lock replay")
	}

	driftedProvider := provider
	driftedProvider.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	artifactDrift := input
	artifactDrift.Providers = []Provider{driftedProvider}
	if err := verifyCapabilityResolutionLockReplay(lock, artifactDrift); err == nil {
		t.Fatal("provider artifact drift was accepted by lock replay")
	}
}

func TestCapabilityResolutionLockSeparatesReplayFromEvidenceRefresh(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-old")},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	refreshed := input
	refreshed.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "evidence-new")}
	if err := verifyCapabilityResolutionLockReplay(lock, refreshed); err != nil {
		t.Fatalf("Evidence refresh changed exact declaration replay: %v", err)
	}
	currentEvidenceFingerprint, err := verifyCapabilityResolutionLockEvidence(lock, refreshed)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := resolveCapability(refreshed)
	if err != nil {
		t.Fatal(err)
	}
	if currentEvidenceFingerprint != fresh.EvidenceFingerprint {
		t.Fatalf(
			"lock Evidence fingerprint=%s fresh=%s",
			currentEvidenceFingerprint,
			fresh.EvidenceFingerprint,
		)
	}
	if fresh.ResolutionFingerprint != lock.ResolutionFingerprint {
		t.Fatal("Evidence refresh changed resolution identity")
	}

	withoutEvidence := refreshed
	withoutEvidence.Evidence = nil
	if err := verifyCapabilityResolutionLockReplay(lock, withoutEvidence); err != nil {
		t.Fatalf("semantic declaration replay unexpectedly required current Evidence: %v", err)
	}
	if _, err := verifyCapabilityResolutionLockEvidence(lock, withoutEvidence); err == nil {
		t.Fatal("current Evidence verification succeeded without Evidence")
	}
}

func TestDuplicateLogicalProviderWithDifferentArtifactsFailsClosed(t *testing.T) {
	contract := sourceAdapterContract()
	left := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	right := left
	right.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)

	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{left, right},
		Evidence:    []ConformanceEvidence{evidenceFor(left, contract, "evidence-current")},
		Wirings: []Wiring{{
			ContractID:       contract.ID,
			PluginID:         left.PluginID,
			PluginVersion:    left.PluginVersion,
			ImplementationID: left.ImplementationID,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "logical provider") {
		t.Fatalf("duplicate logical provider artifacts were not rejected: %v", err)
	}
}

func TestMalformedConformanceEvidenceFingerprintFailsClosed(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	evidence := evidenceFor(provider, contract, "evidence-current")
	evidence.EvidenceFingerprint = "not-a-fingerprint"

	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidence},
	})
	if err == nil || !strings.Contains(err.Error(), "conformance evidence") {
		t.Fatalf("malformed Conformance Evidence was not rejected: %v", err)
	}
}


func TestConformanceEvidenceDoesNotSurviveArtifactReplacement(t *testing.T) {
	contract := sourceAdapterContract()
	original := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	evidence := evidenceFor(original, contract, "evidence-original")

	replaced := original
	replaced.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{replaced},
		Evidence:    []ConformanceEvidence{evidence},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("old Conformance Evidence survived artifact replacement: %v", err)
	}
}


func TestCapabilityResolutionLockRejectsWiringDrift(t *testing.T) {
	contract := sourceAdapterContract()
	left := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.left",
		"xead.reader.left",
	)
	right := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.right",
		"xead.reader.right",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{left, right},
		Evidence: []ConformanceEvidence{
			evidenceFor(left, contract, "evidence-left"),
			evidenceFor(right, contract, "evidence-right"),
		},
		Wirings: []Wiring{{
			ContractID:       contract.ID,
			PluginID:         left.PluginID,
			PluginVersion:    left.PluginVersion,
			ImplementationID: left.ImplementationID,
		}},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Wiring == nil || lock.Wiring.PluginID != left.PluginID {
		t.Fatalf("lock wiring=%+v", lock.Wiring)
	}

	rewired := input
	rewired.Wirings = []Wiring{{
		ContractID:       contract.ID,
		PluginID:         right.PluginID,
		PluginVersion:    right.PluginVersion,
		ImplementationID: right.ImplementationID,
	}}
	if err := verifyCapabilityResolutionLockReplay(lock, rewired); err == nil {
		t.Fatal("changed explicit wiring was accepted by lock replay")
	}

	unwired := input
	unwired.Wirings = nil
	if err := verifyCapabilityResolutionLockReplay(lock, unwired); err == nil {
		t.Fatal("removed explicit wiring was accepted by lock replay")
	}
}


func TestCapabilityResolutionLockRejectsInternallyContradictoryWiring(t *testing.T) {
	contract := sourceAdapterContract()
	left := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.left",
		"xead.reader.left",
	)
	right := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.right",
		"xead.reader.right",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{left, right},
		Evidence: []ConformanceEvidence{
			evidenceFor(left, contract, "evidence-left"),
			evidenceFor(right, contract, "evidence-right"),
		},
		Wirings: []Wiring{{
			ContractID:       contract.ID,
			PluginID:         left.PluginID,
			PluginVersion:    left.PluginVersion,
			ImplementationID: left.ImplementationID,
		}},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	lock.Wiring = &Wiring{
		ContractID:       contract.ID,
		PluginID:         right.PluginID,
		PluginVersion:    right.PluginVersion,
		ImplementationID: right.ImplementationID,
	}
	lock.LockFingerprint = capabilityResolutionLockFingerprint(lock)
	if err := validateCapabilityResolutionLock(lock); err == nil ||
		!strings.Contains(err.Error(), "does not select locked provider") {
		t.Fatalf("contradictory lock wiring was accepted: %v", err)
	}
}


func TestCapabilityResolutionReplayIgnoresMalformedCurrentEvidence(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-valid")},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	malformed := input
	malformed.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "evidence-current")}
	malformed.Evidence[0].EvidenceFingerprint = "malformed"
	if err := verifyCapabilityResolutionLockReplay(lock, malformed); err != nil {
		t.Fatalf("declaration replay was coupled to current Evidence: %v", err)
	}
	if _, err := verifyCapabilityResolutionLockEvidence(lock, malformed); err == nil {
		t.Fatal("current Evidence verification accepted malformed Evidence")
	}
}


func TestIncompleteProviderInOtherScopeDoesNotBlockValidResolution(t *testing.T) {
	contract := sourceAdapterContract()
	valid := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.valid",
	)
	incomplete := sourceAdapterProvider(
		contract,
		ScopeApplicationRuntime,
		"legacy.runtime.reference",
		"legacy.reader",
	)
	incomplete.ArtifactDigest = ""

	resolution, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{incomplete, valid},
		Evidence:    []ConformanceEvidence{evidenceFor(valid, contract, "valid-evidence")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.PluginID != valid.PluginID {
		t.Fatalf("resolved provider=%+v", resolution.Provider)
	}
}

func TestRelevantProviderWithoutArtifactRemainsIneligibleNotGloballyInvalid(t *testing.T) {
	contract := sourceAdapterContract()
	incomplete := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.incomplete",
	)
	incomplete.ArtifactDigest = ""

	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{incomplete},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("incomplete provider was not treated as ineligible candidate: %v", err)
	}
}

func TestIncompleteAndExactDuplicateLogicalProviderStillFailsClosed(t *testing.T) {
	contract := sourceAdapterContract()
	incomplete := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.same",
	)
	incomplete.ArtifactDigest = ""
	exact := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.same",
	)

	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{incomplete, exact},
		Evidence:    []ConformanceEvidence{evidenceFor(exact, contract, "exact-evidence")},
	})
	if err == nil || !strings.Contains(err.Error(), "logical provider") {
		t.Fatalf("duplicate migration/provider authorities were not rejected: %v", err)
	}
}

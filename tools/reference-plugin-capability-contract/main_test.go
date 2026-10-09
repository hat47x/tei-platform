package main

import (
	"strings"
	"testing"
)

func TestNonInvocationContractFingerprintDoesNotGainEmptyShapeIssuer(t *testing.T) {
	contract := Contract{
		ID:                 "reference.no-invocation",
		Version:            "1.0.0",
		RequestSemantics:   "none",
		ResultSemantics:    "none",
		FailureSemantics:   "none",
		ConformanceProfile: "tei.reference.no-invocation/v0",
	}
	legacy := struct {
		ID                      string `json:"id"`
		Version                 string `json:"version"`
		RequestSemantics        string `json:"requestSemantics"`
		ResultSemantics         string `json:"resultSemantics"`
		FailureSemantics        string `json:"failureSemantics"`
		RequestShape            string `json:"requestShape,omitempty"`
		RequestShapeFingerprint string `json:"requestShapeFingerprint,omitempty"`
		ResultShape             string `json:"resultShape,omitempty"`
		ResultShapeFingerprint  string `json:"resultShapeFingerprint,omitempty"`
		PayloadCodec            string `json:"payloadCodec,omitempty"`
		ConformanceProfile      string `json:"conformanceProfile"`
	}{
		ID:                 contract.ID,
		Version:            contract.Version,
		RequestSemantics:   contract.RequestSemantics,
		ResultSemantics:    contract.ResultSemantics,
		FailureSemantics:   contract.FailureSemantics,
		ConformanceProfile: contract.ConformanceProfile,
	}
	if got, want := fingerprint(contract), fingerprint(legacy); got != want {
		t.Fatalf("optional shape issuer changed non-invocation contract identity: got=%s want=%s", got, want)
	}
}

func sourceAdapterContract() Contract {
	return Contract{
		ID:                 "authoring.source-adapter.xead",
		Version:            "1.0.0",
		RequestSemantics:   "exact-source-bytes",
		ResultSemantics:    "source-faithful-inventory-with-unresolved",
		FailureSemantics:   "unsupported-or-invalid-is-not-success",
		RequestShape:            sourceAdapterRequestShape,
		RequestShapeFingerprint: sourceAdapterRequestShapeFingerprint,
		ResultShape:             sourceAdapterResultShape,
		ResultShapeFingerprint:  sourceAdapterResultShapeFingerprint,
		PayloadCodec:            referenceJSONPayloadCodec,
		ShapeIssuer:             referenceInvocationShapeIssuerPointer(),
		InvocationEffect:        invocationEffectReadOnly,
		ConformanceProfile:      "tei.reference.source-adapter/v0",
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
		PluginID:            provider.PluginID,
		PluginVersion:       provider.PluginVersion,
		ImplementationID:    provider.ImplementationID,
		ArtifactDigest:      provider.ArtifactDigest,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile:  contract.ConformanceProfile,
		EvidenceFingerprint: marker,
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

func runtimeContract() Contract {
	return Contract{
		ID:                 contractQueryCapabilityContractID,
		Version:            "1.0.0",
		RequestSemantics:   "contract-query-with-bound-parameters",
		ResultSemantics:    "rows-or-explicit-unresolved",
		FailureSemantics:   "execution-error-is-not-empty-result",
		RequestShape:            contractQueryRequestShape,
		RequestShapeFingerprint: contractQueryRequestShapeFingerprint,
		ResultShape:             contractQueryResultShape,
		ResultShapeFingerprint:  contractQueryResultShapeFingerprint,
		PayloadCodec:            referenceJSONPayloadCodec,
		ShapeIssuer:             referenceInvocationShapeIssuerPointer(),
		InvocationEffect:        invocationEffectReadOnly,
		ConformanceProfile:      "tei.reference.contract-query-executor/v0",
	}
}

func TestAuthoringToolProviderCannotSatisfyRuntimeRequirement(t *testing.T) {
	contract := runtimeContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "evidence-authoring")},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("authoring-tool provider was used as a runtime fallback: %v", err)
	}
}

func TestOneLogicalContractCanHaveDistinctPhysicalArtifacts(t *testing.T) {
	contract := sourceAdapterContract()
	builtin := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	wasm := builtin
	wasm.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{wasm, builtin},
		Evidence: []ConformanceEvidence{
			evidenceFor(builtin, contract, "evidence-builtin"),
			evidenceFor(wasm, contract, "evidence-wasm"),
		},
	}
	// Both artifacts conform to the same contract with their own Evidence.
	// Choosing between them must not depend on digest ordering.
	if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("two physical artifacts were silently selected: %v", err)
	}
	input.Wirings = []Wiring{{
		ContractID:       contract.ID,
		PluginID:         builtin.PluginID,
		PluginVersion:    builtin.PluginVersion,
		ImplementationID: builtin.ImplementationID,
	}}
	if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("wiring without an artifact digest picked a physical artifact by ordering: %v", err)
	}
	input.Wirings[0].ArtifactDigest = wasm.ArtifactDigest
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Provider.ArtifactDigest != wasm.ArtifactDigest {
		t.Fatalf("provider=%+v", resolution.Provider)
	}
	other, err := resolveCapability(ResolveInput{
		Scope:       input.Scope,
		Requirement: input.Requirement,
		Contracts:   input.Contracts,
		Providers:   []Provider{builtin},
		Evidence:    input.Evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	if other.ContractFingerprint != resolution.ContractFingerprint {
		t.Fatal("the logical contract fingerprint must not depend on the physical artifact")
	}
	if other.ResolutionFingerprint == resolution.ResolutionFingerprint {
		t.Fatal("the resolution identity must record which physical artifact was selected")
	}
}

func TestConformanceEvidenceOfOneArtifactIsNotInheritedByAnother(t *testing.T) {
	contract := sourceAdapterContract()
	builtin := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	wasm := builtin
	wasm.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	_, err := resolveCapability(ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{wasm},
		Evidence:    []ConformanceEvidence{evidenceFor(builtin, contract, "evidence-builtin")},
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("a different artifact inherited Conformance Evidence: %v", err)
	}
}

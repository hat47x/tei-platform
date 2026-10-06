package main

import (
    "strings"
    "testing"
)

func sourceAdapterContract() Contract {
    return Contract{
        ID: "authoring.source-adapter.xead",
        Version: "1.0.0",
        RequestSemantics: "exact-source-bytes",
        ResultSemantics: "source-faithful-inventory-with-unresolved",
        FailureSemantics: "unsupported-or-invalid-is-not-success",
        ConformanceProfile: "tei.reference.source-adapter/v0",
    }
}

func sourceAdapterProvider(contract Contract, scope AuthorityScope, implementation string) Provider {
    return Provider{
        PluginID: "reference.compat.xead.source",
        PluginVersion: "1.0.0",
        ImplementationID: implementation,
        ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
        Scope: scope,
        ContractID: contract.ID,
        ContractVersion: contract.Version,
        ContractFingerprint: fingerprint(contract),
        ConformanceProfile: contract.ConformanceProfile,
    }
}

func evidenceFor(provider Provider, contract Contract, marker string) ConformanceEvidence {
    return ConformanceEvidence{
        PluginID: provider.PluginID,
        PluginVersion: provider.PluginVersion,
        ImplementationID: provider.ImplementationID,
        ContractFingerprint: fingerprint(contract),
        ConformanceProfile: contract.ConformanceProfile,
        EvidenceFingerprint: marker,
    }
}

func TestSourceAdapterContractDoesNotRequireCoreCapabilityKind(t *testing.T) {
    contract := sourceAdapterContract()
    provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.v0")
    resolution, err := resolveCapability(ResolveInput{
        Scope: ScopeAuthoringTool,
        Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
        Contracts: []Contract{contract},
        Providers: []Provider{provider},
        Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "evidence-a")},
    })
    if err != nil { t.Fatal(err) }
    if resolution.Contract.ID != "authoring.source-adapter.xead" { t.Fatalf("contract=%q", resolution.Contract.ID) }
    if resolution.Provider.ImplementationID != "xead.reader.v0" { t.Fatalf("provider=%+v", resolution.Provider) }
}

func TestProviderSelfClaimWithoutConformanceEvidenceFailsClosed(t *testing.T) {
    contract := sourceAdapterContract()
    provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.v0")
    _, err := resolveCapability(ResolveInput{
        Scope: ScopeAuthoringTool,
        Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
        Contracts: []Contract{contract},
        Providers: []Provider{provider},
    })
    if err == nil || !strings.Contains(err.Error(), "no conformant provider") { t.Fatalf("expected missing conformance evidence, got %v", err) }
}

func TestRuntimeProviderCannotSatisfyAuthoringToolRequirement(t *testing.T) {
    contract := sourceAdapterContract()
    provider := sourceAdapterProvider(contract, ScopeApplicationRuntime, "runtime.reader")
    _, err := resolveCapability(ResolveInput{
        Scope: ScopeAuthoringTool,
        Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
        Contracts: []Contract{contract},
        Providers: []Provider{provider},
        Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "evidence-runtime")},
    })
    if err == nil || !strings.Contains(err.Error(), "no conformant provider") { t.Fatalf("wrong authority scope was accepted: %v", err) }
}

func TestAmbiguousProvidersRequireExplicitWiring(t *testing.T) {
    contract := sourceAdapterContract()
    left := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.left")
    right := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.right")
    input := ResolveInput{
        Scope: ScopeAuthoringTool,
        Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
        Contracts: []Contract{contract},
        Providers: []Provider{right, left},
        Evidence: []ConformanceEvidence{evidenceFor(left, contract, "evidence-left"), evidenceFor(right, contract, "evidence-right")},
    }
    if _, err := resolveCapability(input); err == nil || !strings.Contains(err.Error(), "ambiguous") { t.Fatalf("ambiguous providers were silently selected: %v", err) }
    input.Wirings = []Wiring{{ContractID: contract.ID, ImplementationID: right.ImplementationID}}
    resolution, err := resolveCapability(input)
    if err != nil { t.Fatal(err) }
    if resolution.Provider.ImplementationID != right.ImplementationID { t.Fatalf("provider=%+v", resolution.Provider) }
}

func TestCatalogOrderingDoesNotChangeResolutionIdentity(t *testing.T) {
    v1 := sourceAdapterContract()
    v11 := sourceAdapterContract()
    v11.Version = "1.1.0"
    provider := sourceAdapterProvider(v11, ScopeAuthoringTool, "xead.reader.v1")
    evidence := evidenceFor(provider, v11, "evidence-current")
    left, err := resolveCapability(ResolveInput{Scope: ScopeAuthoringTool, Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1}, Contracts: []Contract{v1, v11}, Providers: []Provider{provider}, Evidence: []ConformanceEvidence{evidence}})
    if err != nil { t.Fatal(err) }
    right, err := resolveCapability(ResolveInput{Scope: ScopeAuthoringTool, Requirement: Requirement{ContractID: v1.ID, CompatibleMajor: 1}, Contracts: []Contract{v11, v1}, Providers: []Provider{provider}, Evidence: []ConformanceEvidence{evidence}})
    if err != nil { t.Fatal(err) }
    if left.Contract.Version != "1.1.0" || right.Contract.Version != "1.1.0" { t.Fatalf("highest compatible contract was not selected") }
    if left.ResolutionFingerprint != right.ResolutionFingerprint { t.Fatalf("catalog ordering changed resolution identity") }
}

func TestEvidenceRefreshDoesNotChangeResolutionIdentity(t *testing.T) {
    contract := sourceAdapterContract()
    provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.v0")
    input := ResolveInput{Scope: ScopeAuthoringTool, Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1}, Contracts: []Contract{contract}, Providers: []Provider{provider}, Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "evidence-old")}}
    left, err := resolveCapability(input)
    if err != nil { t.Fatal(err) }
    input.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "evidence-new")}
    right, err := resolveCapability(input)
    if err != nil { t.Fatal(err) }
    if left.ResolutionFingerprint != right.ResolutionFingerprint { t.Fatalf("evidence refresh changed resolution identity") }
    if left.EvidenceFingerprint == right.EvidenceFingerprint { t.Fatalf("evidence refresh was not observable") }
}

func TestContractFingerprintMismatchFailsClosed(t *testing.T) {
    contract := sourceAdapterContract()
    provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "xead.reader.v0")
    provider.ContractFingerprint = "sha256:" + strings.Repeat("0", 64)
    _, err := resolveCapability(ResolveInput{Scope: ScopeAuthoringTool, Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1}, Contracts: []Contract{contract}, Providers: []Provider{provider}, Evidence: []ConformanceEvidence{evidenceFor(provider, contract, "evidence-a")}})
    if err == nil || !strings.Contains(err.Error(), "no conformant provider") { t.Fatalf("contract drift was accepted: %v", err) }
}
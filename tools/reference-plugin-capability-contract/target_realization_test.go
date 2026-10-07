package main

import (
	"strings"
	"testing"
)

func portableRuntimeContract() Contract {
	return Contract{
		ID:                 "application-runtime.reference-portable",
		Version:            "1.0.0",
		RequestSemantics:   "execute-reference-operation",
		ResultSemantics:    "return-reference-result",
		FailureSemantics:   "unsupported-target-or-execution-failure-is-not-success",
		ConformanceProfile: "tei.reference.portable-runtime/v0",
	}
}

func targetRuntimeProvider(contract Contract, target, digestMarker string) Provider {
	return Provider{
		PluginID:          "tei.reference.portable",
		PluginVersion:     "1.0.0",
		ImplementationID:  "portable.runtime.v1",
		Target:            target,
		ArtifactDigest:    "sha256:" + strings.Repeat(digestMarker, 64),
		Scope:             ScopeApplicationRuntime,
		ContractID:        contract.ID,
		ContractVersion:   contract.Version,
		ContractFingerprint: fingerprint(contract),
		ConformanceProfile: contract.ConformanceProfile,
	}
}

func targetResolveInput(
	contract Contract,
	target string,
	providers ...Provider,
) ResolveInput {
	evidence := make([]ConformanceEvidence, 0, len(providers))
	for i, provider := range providers {
		evidence = append(
			evidence,
			evidenceFor(provider, contract, "target-evidence-"+string(rune('a'+i))),
		)
	}
	return ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Target:      target,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   providers,
		Evidence:    evidence,
	}
}

func TestSameLogicalProviderCanUseDifferentArtifactsForDifferentTargets(t *testing.T) {
	contract := portableRuntimeContract()
	desktop := targetRuntimeProvider(contract, "desktop", "1")
	web := targetRuntimeProvider(contract, "web-browser", "2")
	input := targetResolveInput(contract, "web-browser", desktop, web)

	webResolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if webResolution.Target != "web-browser" ||
		webResolution.Provider.Target != "web-browser" ||
		webResolution.Provider.ArtifactDigest != web.ArtifactDigest {
		t.Fatalf("web resolution=%+v", webResolution)
	}

	input.Target = "desktop"
	desktopResolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if desktopResolution.Target != "desktop" ||
		desktopResolution.Provider.Target != "desktop" ||
		desktopResolution.Provider.ArtifactDigest != desktop.ArtifactDigest {
		t.Fatalf("desktop resolution=%+v", desktopResolution)
	}

	if webResolution.ContractFingerprint != desktopResolution.ContractFingerprint {
		t.Fatal("Target packaging changed capability Contract identity")
	}
	if webResolution.Provider.PluginID != desktopResolution.Provider.PluginID ||
		webResolution.Provider.PluginVersion != desktopResolution.Provider.PluginVersion ||
		webResolution.Provider.ImplementationID != desktopResolution.Provider.ImplementationID {
		t.Fatal("Target packaging changed logical Provider identity")
	}
	if webResolution.ResolutionFingerprint == desktopResolution.ResolutionFingerprint {
		t.Fatal("different Target realizations collapsed into one resolution identity")
	}
}

func TestTargetSpecificProvidersAreNotGuessedWithoutTargetContext(t *testing.T) {
	contract := portableRuntimeContract()
	desktop := targetRuntimeProvider(contract, "desktop", "1")
	web := targetRuntimeProvider(contract, "web-browser", "2")
	input := targetResolveInput(contract, "", desktop, web)

	_, err := resolveCapability(input)
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("Target-specific Provider was guessed without Target context: %v", err)
	}
}

func TestTargetSpecificProviderDoesNotFallbackAcrossTargets(t *testing.T) {
	contract := portableRuntimeContract()
	desktop := targetRuntimeProvider(contract, "desktop", "1")
	input := targetResolveInput(contract, "web-browser", desktop)

	_, err := resolveCapability(input)
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("desktop Provider was used as Web fallback: %v", err)
	}
}

func TestTargetSpecificProviderRequiresEvidenceForExactTarget(t *testing.T) {
	contract := portableRuntimeContract()
	web := targetRuntimeProvider(contract, "web-browser", "2")
	desktopEvidenceProvider := web
	desktopEvidenceProvider.Target = "desktop"

	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Target:      "web-browser",
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{web},
		Evidence: []ConformanceEvidence{
			evidenceFor(
				desktopEvidenceProvider,
				contract,
				"desktop-only-conformance",
			),
		},
	}
	_, err := resolveCapability(input)
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("Conformance Evidence crossed Target boundary: %v", err)
	}
}

func TestCapabilityResolutionLockRejectsTargetDrift(t *testing.T) {
	contract := portableRuntimeContract()
	desktop := targetRuntimeProvider(contract, "desktop", "1")
	web := targetRuntimeProvider(contract, "web-browser", "2")
	webInput := targetResolveInput(contract, "web-browser", desktop, web)

	resolution, err := resolveCapability(webInput)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(webInput, resolution)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Target != "web-browser" {
		t.Fatalf("lock target=%q", lock.Target)
	}

	desktopInput := targetResolveInput(contract, "desktop", desktop, web)
	if err := verifyCapabilityResolutionLockReplay(lock, desktopInput); err == nil ||
		!strings.Contains(err.Error(), "target") {
		t.Fatalf("Target drift was accepted by Capability Resolution Lock: %v", err)
	}
}

func TestSameLogicalProviderDuplicateWithinOneTargetFailsClosed(t *testing.T) {
	contract := portableRuntimeContract()
	left := targetRuntimeProvider(contract, "web-browser", "1")
	right := left
	right.ArtifactDigest = "sha256:" + strings.Repeat("2", 64)

	input := targetResolveInput(contract, "web-browser", left, right)
	_, err := resolveCapability(input)
	if err == nil || !strings.Contains(err.Error(), "logical provider") {
		t.Fatalf("same-Target duplicate realization was not rejected: %v", err)
	}
}

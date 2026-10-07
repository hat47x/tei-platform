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
		RealizationTarget:            target,
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
		RealizationTarget:      target,
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
	if webResolution.RealizationTarget != "web-browser" ||
		webResolution.Provider.RealizationTarget != "web-browser" ||
		webResolution.Provider.ArtifactDigest != web.ArtifactDigest {
		t.Fatalf("web resolution=%+v", webResolution)
	}

	input.RealizationTarget = "desktop"
	desktopResolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if desktopResolution.RealizationTarget != "desktop" ||
		desktopResolution.Provider.RealizationTarget != "desktop" ||
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
	desktopEvidenceProvider.RealizationTarget = "desktop"

	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		RealizationTarget:      "web-browser",
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
	if lock.RealizationTarget != "web-browser" {
		t.Fatalf("lock target=%q", lock.RealizationTarget)
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


func TestCapabilityResolutionLockRejectsNonCanonicalTargetText(t *testing.T) {
	contract := portableRuntimeContract()
	web := targetRuntimeProvider(contract, "web-browser", "3")
	input := targetResolveInput(contract, "web-browser", web)

	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}

	lock.RealizationTarget = " web-browser "
	lock.Provider.RealizationTarget = lock.RealizationTarget
	lock.ResolutionFingerprint = capabilityResolutionFingerprint(
		lock.Scope,
		lock.Contract,
		lock.ContractFingerprint,
		lock.Provider,
	)
	lock.LockFingerprint = capabilityResolutionLockFingerprint(lock)

	if err := validateCapabilityResolutionLock(lock); err == nil ||
		!strings.Contains(err.Error(), "surrounding whitespace") {
		t.Fatalf("non-canonical Target text was accepted in lock: %v", err)
	}
}

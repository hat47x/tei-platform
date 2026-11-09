package main

import (
	"fmt"

	"tei/internal/composition"
	handoff "tei/reference/composition-evidence-handoff"
)

// CompositionResolutionExpectation is an explicit, design-time crosswalk.
// The caller owns this mapping. It is not inferred from a similar name or
// from the Composition contract's Shape, which is NOT the #951 fingerprint.
type CompositionResolutionExpectation struct {
	Target   composition.VerificationTarget
	Provider Provider
}

// resolveCompositionEvidenceReference delegates provider/conformance resolution
// to the EXISTING #951 resolver. It only checks that its exact result matches
// the intended Composition subject and explicitly pinned implementation.
// Neither this reference adapter nor Reconcile attests publisher authority,
// conformance issuer trust, artifact provenance or Host permission.
func resolveCompositionEvidenceReference(
	input ResolveInput,
	expected CompositionResolutionExpectation,
) (handoff.EvidenceReference, error) {
	target := expected.Target
	pinned := expected.Provider
	if target.ProviderID == "" || target.Scope == "" || target.Epoch == 0 ||
		target.Contract.ID == "" || target.Contract.Version == "" || target.Contract.Shape == "" {
		return handoff.EvidenceReference{}, fmt.Errorf("composition target is incomplete")
	}
	if !input.Scope.Valid() || !pinned.Scope.Valid() || input.Scope != pinned.Scope ||
		input.Requirement.ContractID != target.Contract.ID ||
		pinned.ContractID != target.Contract.ID ||
		pinned.ContractVersion != target.Contract.Version ||
		pinned.PluginID == "" || pinned.PluginVersion == "" ||
		pinned.ImplementationID == "" || pinned.ArtifactDigest == "" ||
		pinned.ContractFingerprint == "" || pinned.ConformanceProfile == "" {
		return handoff.EvidenceReference{}, fmt.Errorf("composition-to-capability crosswalk does not pin one complete contract and provider")
	}
	resolved, err := resolveCapability(input)
	if err != nil {
		return handoff.EvidenceReference{}, fmt.Errorf("capability resolver rejected proposal subject: %w", err)
	}
	if resolved.Scope != pinned.Scope ||
		resolved.Contract.ID != target.Contract.ID ||
		resolved.Contract.Version != target.Contract.Version ||
		resolved.ContractFingerprint != pinned.ContractFingerprint ||
		resolved.Provider != pinned ||
		resolved.EvidenceFingerprint == "" || resolved.ResolutionFingerprint == "" {
		return handoff.EvidenceReference{}, fmt.Errorf("capability resolver result differs from explicitly pinned composition binding")
	}
	return handoff.EvidenceReference{
		Subject:               target,
		AuthorityScope:        string(resolved.Scope),
		PluginID:              resolved.Provider.PluginID,
		PluginVersion:         resolved.Provider.PluginVersion,
		ImplementationID:      resolved.Provider.ImplementationID,
		ArtifactDigest:        resolved.Provider.ArtifactDigest,
		ContractFingerprint:   resolved.ContractFingerprint,
		ConformanceProfile:    resolved.Provider.ConformanceProfile,
		EvidenceFingerprint:   resolved.EvidenceFingerprint,
		ResolutionFingerprint: resolved.ResolutionFingerprint,
	}, nil
}

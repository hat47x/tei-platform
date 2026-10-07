package main

import (
	"strings"
	"testing"
)

func capabilityAuthorityFixture(
	t *testing.T,
) (ResolveInput, CapabilityResolutionLock, ConformanceEvidence) {
	t.Helper()
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"xead.reader.v0",
	)
	evidence := evidenceFor(provider, contract, "source-adapter-conformance")
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidence},
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildCapabilityResolutionLock(input, resolution)
	if err != nil {
		t.Fatal(err)
	}
	return input, lock, evidence
}

func authorityGrantFor(
	lock CapabilityResolutionLock,
	issuer string,
	marker string,
) CapabilityConformanceAuthorityGrant {
	return CapabilityConformanceAuthorityGrant{
		ContractFingerprint:          lock.ContractFingerprint,
		ConformanceProfile:           lock.Contract.ConformanceProfile,
		Issuer:                       issuer,
		AuthorityEvidenceFingerprint: fingerprint(marker),
	}
}

func attestationFor(
	evidence ConformanceEvidence,
	grant CapabilityConformanceAuthorityGrant,
) CapabilityConformanceAttestation {
	return CapabilityConformanceAttestation{
		EvidenceFingerprint:          evidence.EvidenceFingerprint,
		Issuer:                       grant.Issuer,
		AuthorityEvidenceFingerprint: grant.AuthorityEvidenceFingerprint,
	}
}

func TestCapabilityConformanceRequiresAcceptedIssuerAuthority(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	grant := authorityGrantFor(lock, "tei.conformance.reference", "issuer-authority")
	attestation := attestationFor(evidence, grant)

	binding, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{grant},
		[]CapabilityConformanceAttestation{attestation},
	)
	if err != nil {
		t.Fatal(err)
	}
	if binding.CapabilityLockFingerprint != lock.LockFingerprint ||
		binding.ConformanceEvidenceFingerprint != evidence.EvidenceFingerprint ||
		binding.Issuer != grant.Issuer ||
		binding.AuthorityEvidenceFingerprint != grant.AuthorityEvidenceFingerprint ||
		!validSHA256Fingerprint(binding.BindingFingerprint) {
		t.Fatalf("authority binding=%+v", binding)
	}
}

func TestCapabilityConformanceEvidenceCannotSelfAuthorize(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	selfIssuer := lock.Provider.PluginID
	selfGrant := authorityGrantFor(lock, selfIssuer, "self-authority")
	selfAttestation := attestationFor(evidence, selfGrant)

	_, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		nil,
		[]CapabilityConformanceAttestation{selfAttestation},
	)
	if err == nil || !strings.Contains(err.Error(), "is not accepted") {
		t.Fatalf("self-issued Conformance Evidence became authority without grant: %v", err)
	}

	if _, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{selfGrant},
		[]CapabilityConformanceAttestation{selfAttestation},
	); err != nil {
		t.Fatalf("explicitly accepted self-issuer should be representable: %v", err)
	}
}

func TestCapabilityConformanceRejectsMissingIssuerAttestation(t *testing.T) {
	input, lock, _ := capabilityAuthorityFixture(t)
	grant := authorityGrantFor(lock, "tei.conformance.reference", "issuer-authority")

	_, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{grant},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "no accepted issuer attestation") {
		t.Fatalf("unattested Conformance Evidence was accepted: %v", err)
	}
}

func TestCapabilityConformanceRejectsStaleAuthorityEvidence(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	grant := authorityGrantFor(lock, "tei.conformance.reference", "issuer-authority")
	stale := attestationFor(evidence, grant)
	stale.AuthorityEvidenceFingerprint = fingerprint("old-authority")

	_, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{grant},
		[]CapabilityConformanceAttestation{stale},
	)
	if err == nil || !strings.Contains(err.Error(), "stale Authority Evidence") {
		t.Fatalf("stale issuer authority was accepted: %v", err)
	}
}

func TestCapabilityAuthorityRefreshDoesNotChangeResolutionIdentity(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	beforeLockFingerprint := lock.LockFingerprint

	leftGrant := authorityGrantFor(lock, "tei.conformance.reference", "authority-left")
	leftBinding, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{leftGrant},
		[]CapabilityConformanceAttestation{attestationFor(evidence, leftGrant)},
	)
	if err != nil {
		t.Fatal(err)
	}

	rightGrant := authorityGrantFor(lock, "tei.conformance.reference", "authority-right")
	rightBinding, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{rightGrant},
		[]CapabilityConformanceAttestation{attestationFor(evidence, rightGrant)},
	)
	if err != nil {
		t.Fatal(err)
	}

	if lock.LockFingerprint != beforeLockFingerprint {
		t.Fatal("Authority Evidence refresh changed Capability Lock identity")
	}
	if leftBinding.BindingFingerprint == rightBinding.BindingFingerprint {
		t.Fatal("Authority Evidence refresh was invisible in authority binding")
	}
}

func TestCapabilityConformanceRejectsWrongContractAuthority(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	grant := authorityGrantFor(lock, "tei.conformance.reference", "issuer-authority")
	wrong := grant
	wrong.ContractFingerprint = fingerprint("different-contract")
	attestation := attestationFor(evidence, wrong)

	_, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{wrong},
		[]CapabilityConformanceAttestation{attestation},
	)
	if err == nil || !strings.Contains(err.Error(), "is not accepted") {
		t.Fatalf("authority for another Contract was accepted: %v", err)
	}
}

func TestCapabilityConformanceRejectsConflictingIssuerAttestations(t *testing.T) {
	input, lock, evidence := capabilityAuthorityFixture(t)
	left := authorityGrantFor(lock, "tei.conformance.left", "authority-left")
	right := authorityGrantFor(lock, "tei.conformance.right", "authority-right")

	_, err := verifyCapabilityResolutionAcceptedConformance(
		lock,
		input,
		[]CapabilityConformanceAuthorityGrant{left, right},
		[]CapabilityConformanceAttestation{
			attestationFor(evidence, left),
			attestationFor(evidence, right),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "conflicting issuer attestations") {
		t.Fatalf("conflicting issuer attestations were accepted: %v", err)
	}
}

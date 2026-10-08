package main

import (
	"fmt"
	"sort"
	"strings"
)

const capabilityConformanceAuthorityBindingSchema =
	"tei.reference.capability-conformance-authority-binding/v0"

type CapabilityConformanceAuthorityGrant struct {
	ContractFingerprint      string `json:"contractFingerprint"`
	ConformanceProfile       string `json:"conformanceProfile"`
	Issuer                    string `json:"issuer"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
}

type CapabilityConformanceAttestation struct {
	EvidenceFingerprint      string `json:"evidenceFingerprint"`
	Issuer                   string `json:"issuer"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
}

type CapabilityConformanceAuthorityBinding struct {
	Schema                       string `json:"schema"`
	CapabilityLockFingerprint    string `json:"capabilityLockFingerprint"`
	ConformanceEvidenceFingerprint string `json:"conformanceEvidenceFingerprint"`
	Issuer                       string `json:"issuer"`
	AuthorityEvidenceFingerprint string `json:"authorityEvidenceFingerprint"`
	BindingFingerprint           string `json:"bindingFingerprint"`
}

func verifyCapabilityResolutionAcceptedConformance(
	lock CapabilityResolutionLock,
	input ResolveInput,
	grants []CapabilityConformanceAuthorityGrant,
	attestations []CapabilityConformanceAttestation,
) (CapabilityConformanceAuthorityBinding, error) {
	if err := verifyCapabilityResolutionLockReplay(lock, input); err != nil {
		return CapabilityConformanceAuthorityBinding{}, err
	}
	if err := validateResolveInput(input); err != nil {
		return CapabilityConformanceAuthorityBinding{}, err
	}

	evidence, ok := matchingEvidence(
		input.Evidence,
		lock.Provider,
		lock.ContractFingerprint,
		lock.Contract.ConformanceProfile,
	)
	if !ok {
		return CapabilityConformanceAuthorityBinding{}, fmt.Errorf(
			"capability lock has no unique current Conformance Evidence for %s@%s/%s",
			lock.Provider.PluginID,
			lock.Provider.PluginVersion,
			lock.Provider.ImplementationID,
		)
	}

	normalizedGrants, err := normalizeCapabilityConformanceAuthorityGrants(grants)
	if err != nil {
		return CapabilityConformanceAuthorityBinding{}, err
	}
	normalizedAttestations, err := normalizeCapabilityConformanceAttestations(attestations)
	if err != nil {
		return CapabilityConformanceAuthorityBinding{}, err
	}

	attestation, ok := normalizedAttestations[evidence.EvidenceFingerprint]
	if !ok {
		return CapabilityConformanceAuthorityBinding{}, fmt.Errorf(
			"Conformance Evidence %s has no accepted issuer attestation",
			evidence.EvidenceFingerprint,
		)
	}

	grantKey := capabilityConformanceAuthorityGrantKey(
		lock.ContractFingerprint,
		lock.Contract.ConformanceProfile,
		attestation.Issuer,
	)
	grant, ok := normalizedGrants[grantKey]
	if !ok {
		return CapabilityConformanceAuthorityBinding{}, fmt.Errorf(
			"issuer %q is not accepted for contract %s profile %s",
			attestation.Issuer,
			lock.ContractFingerprint,
			lock.Contract.ConformanceProfile,
		)
	}
	if attestation.AuthorityEvidenceFingerprint !=
		grant.AuthorityEvidenceFingerprint {
		return CapabilityConformanceAuthorityBinding{}, fmt.Errorf(
			"Conformance Evidence issuer attestation references stale Authority Evidence",
		)
	}

	binding := CapabilityConformanceAuthorityBinding{
		Schema:                         capabilityConformanceAuthorityBindingSchema,
		CapabilityLockFingerprint:      lock.LockFingerprint,
		ConformanceEvidenceFingerprint: evidence.EvidenceFingerprint,
		Issuer:                         attestation.Issuer,
		AuthorityEvidenceFingerprint:   grant.AuthorityEvidenceFingerprint,
	}
	binding.BindingFingerprint =
		capabilityConformanceAuthorityBindingFingerprint(binding)
	return binding, nil
}

func normalizeCapabilityConformanceAuthorityGrants(
	values []CapabilityConformanceAuthorityGrant,
) (map[string]CapabilityConformanceAuthorityGrant, error) {
	out := map[string]CapabilityConformanceAuthorityGrant{}
	for i, value := range values {
		value.ContractFingerprint = strings.TrimSpace(value.ContractFingerprint)
		value.ConformanceProfile = strings.TrimSpace(value.ConformanceProfile)
		value.Issuer = strings.TrimSpace(value.Issuer)
		value.AuthorityEvidenceFingerprint =
			strings.TrimSpace(value.AuthorityEvidenceFingerprint)
		if !validSHA256Fingerprint(value.ContractFingerprint) ||
			value.ConformanceProfile == "" ||
			value.Issuer == "" ||
			!validSHA256Fingerprint(value.AuthorityEvidenceFingerprint) {
			return nil, fmt.Errorf(
				"capability conformance authority grant[%d] is incomplete or invalid",
				i,
			)
		}
		key := capabilityConformanceAuthorityGrantKey(
			value.ContractFingerprint,
			value.ConformanceProfile,
			value.Issuer,
		)
		if existing, ok := out[key]; ok && existing != value {
			return nil, fmt.Errorf(
				"capability conformance authority grant %s is conflicting",
				key,
			)
		}
		out[key] = value
	}
	return out, nil
}

func normalizeCapabilityConformanceAttestations(
	values []CapabilityConformanceAttestation,
) (map[string]CapabilityConformanceAttestation, error) {
	out := map[string]CapabilityConformanceAttestation{}
	for i, value := range values {
		value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)
		value.Issuer = strings.TrimSpace(value.Issuer)
		value.AuthorityEvidenceFingerprint =
			strings.TrimSpace(value.AuthorityEvidenceFingerprint)
		if !validSHA256Fingerprint(value.EvidenceFingerprint) ||
			value.Issuer == "" ||
			!validSHA256Fingerprint(value.AuthorityEvidenceFingerprint) {
			return nil, fmt.Errorf(
				"capability conformance attestation[%d] is incomplete or invalid",
				i,
			)
		}
		if existing, ok := out[value.EvidenceFingerprint]; ok &&
			existing != value {
			return nil, fmt.Errorf(
				"Conformance Evidence %s has conflicting issuer attestations",
				value.EvidenceFingerprint,
			)
		}
		out[value.EvidenceFingerprint] = value
	}
	return out, nil
}

func capabilityConformanceAuthorityGrantKey(
	contractFingerprint string,
	profile string,
	issuer string,
) string {
	return strings.Join(
		[]string{contractFingerprint, profile, issuer},
		"\x00",
	)
}

func capabilityConformanceAuthorityBindingFingerprint(
	binding CapabilityConformanceAuthorityBinding,
) string {
	binding.BindingFingerprint = ""
	return fingerprint(binding)
}

func sortedCapabilityConformanceAuthorityGrants(
	values map[string]CapabilityConformanceAuthorityGrant,
) []CapabilityConformanceAuthorityGrant {
	out := make([]CapabilityConformanceAuthorityGrant, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ContractFingerprint != out[j].ContractFingerprint {
			return out[i].ContractFingerprint < out[j].ContractFingerprint
		}
		if out[i].ConformanceProfile != out[j].ConformanceProfile {
			return out[i].ConformanceProfile < out[j].ConformanceProfile
		}
		return out[i].Issuer < out[j].Issuer
	})
	return out
}

package main

import (
	"fmt"
	"sort"
)

// referenceIssuerSelection reports a policy decision; it is not Runtime
// Acceptance or cryptographic authentication. The fake signature profile
// used by this Reference never proves the issuer controls a signing key.
type referenceIssuerSelection struct {
	Issuer                          string `json:"issuer"`
	PolicyFingerprint               string `json:"policyFingerprint"`
	SelectedAttestationFingerprint string `json:"selectedAttestationFingerprint,omitempty"`
	Matched                         bool   `json:"matched"`
	Reason                          string `json:"reason"`
}

type referenceAttestationSelection struct {
	Satisfied bool                       `json:"satisfied"`
	Issuers   []referenceIssuerSelection `json:"issuers"`
}

// selectReferenceAttestations illustrates an explicit policy requirement over
// independent immutable attestations. Each required issuer must have at least
// one currently policy-matching attestation for the same exact Build result.
// It deliberately does not infer requirements from the signatures present.
func selectReferenceAttestations(
	plan BuildExecutionPlan,
	set ArtifactSet,
	attestations []BuildAttestation,
	requiredPolicies []AttestationTrustPolicy,
) (referenceAttestationSelection, error) {
	if len(requiredPolicies) == 0 {
		return referenceAttestationSelection{}, fmt.Errorf("reference attestation selection requires an explicit issuer policy")
	}
	policies := append([]AttestationTrustPolicy(nil), requiredPolicies...)
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Issuer < policies[j].Issuer
	})
	for i, policy := range policies {
		if err := policy.Validate(); err != nil {
			return referenceAttestationSelection{}, err
		}
		if i > 0 && policies[i-1].Issuer == policy.Issuer {
			return referenceAttestationSelection{}, fmt.Errorf("reference attestation issuer %q is required more than once", policy.Issuer)
		}
	}

	type verifiedAttestation struct {
		attestation  BuildAttestation
		verification BuildAttestationVerification
	}
	byIssuer := make(map[string][]verifiedAttestation)
	for _, attestation := range attestations {
		verified, err := verifyBuildAttestation(attestation, plan, set)
		if err != nil {
			return referenceAttestationSelection{}, fmt.Errorf("reference attestation has invalid subject or binding: %w", err)
		}
		byIssuer[verified.Issuer] = append(byIssuer[verified.Issuer], verifiedAttestation{
			attestation:  attestation,
			verification: verified,
		})
	}
	for issuer := range byIssuer {
		sort.Slice(byIssuer[issuer], func(i, j int) bool {
			return byIssuer[issuer][i].verification.AttestationFingerprint <
				byIssuer[issuer][j].verification.AttestationFingerprint
		})
	}

	result := referenceAttestationSelection{
		Satisfied: true,
		Issuers:   make([]referenceIssuerSelection, 0, len(policies)),
	}
	for _, policy := range policies {
		policyFingerprint, err := attestationTrustPolicyFingerprint(policy)
		if err != nil {
			return referenceAttestationSelection{}, err
		}
		selection := referenceIssuerSelection{
			Issuer:            policy.Issuer,
			PolicyFingerprint: policyFingerprint,
			Reason:            "required-issuer-attestation-missing",
		}
		for _, verified := range byIssuer[policy.Issuer] {
			decision, err := evaluateAttestationTrust(verified.attestation, verified.verification, policy)
			if err != nil {
				return referenceAttestationSelection{}, err
			}
			selection.SelectedAttestationFingerprint = decision.AttestationFingerprint
			selection.Reason = decision.Reason
			if decision.Trusted {
				selection.Matched = true
				break
			}
		}
		if !selection.Matched {
			result.Satisfied = false
		}
		result.Issuers = append(result.Issuers, selection)
	}
	return result, nil
}

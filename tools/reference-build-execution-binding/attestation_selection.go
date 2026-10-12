package main

import (
	"fmt"
	"sort"
	"strings"
)

// referenceIssuerSelection reports a policy decision; it is not Runtime
// Acceptance or cryptographic authentication. The fake signature profile
// used by this Reference never proves the issuer controls a signing key.
type referenceIssuerSelection struct {
	Issuer                         string `json:"issuer"`
	PolicyFingerprint              string `json:"policyFingerprint"`
	SelectedAttestationFingerprint string `json:"selectedAttestationFingerprint,omitempty"`
	Matched                        bool   `json:"matched"`
	Reason                         string `json:"reason"`
}

type referenceAttestationSelection struct {
	RequiredSubject AttestationSubject         `json:"requiredSubject"`
	Satisfied       bool                       `json:"satisfied"`
	Issuers         []referenceIssuerSelection `json:"issuers"`
}

// referenceAttestationScope is an explicit request about the subject of
// the proof. A member attestation cannot satisfy a whole-set requirement.
type referenceAttestationScope struct {
	Kind AttestationSubjectKind `json:"kind"`
	Role string                 `json:"role,omitempty"`
}

func referenceRequiredAttestationSubject(set ArtifactSet, scope referenceAttestationScope) (AttestationSubject, error) {
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		return AttestationSubject{}, err
	}
	subject := AttestationSubject{
		Kind:                   scope.Kind,
		ArtifactSetFingerprint: setFingerprint,
	}
	switch scope.Kind {
	case attestationSubjectArtifactSet:
		if scope.Role != "" {
			return AttestationSubject{}, fmt.Errorf("artifact-set requirement must not specify a member role")
		}
	case attestationSubjectArtifactMember:
		if strings.TrimSpace(scope.Role) == "" {
			return AttestationSubject{}, fmt.Errorf("artifact-member requirement needs a member role")
		}
		member, ok := artifactMemberByRole(set, scope.Role)
		if !ok {
			return AttestationSubject{}, fmt.Errorf("required artifact member role %q is absent", scope.Role)
		}
		subject.Role = scope.Role
		subject.ArtifactDigest = member.Digest
	default:
		return AttestationSubject{}, fmt.Errorf("unsupported required attestation subject kind %q", scope.Kind)
	}
	if err := subject.Validate(); err != nil {
		return AttestationSubject{}, err
	}
	return subject, nil
}

// selectReferenceAttestations requires attestations covering the whole set.
// Member-level requirements must use selectReferenceAttestationsForSubject.
func selectReferenceAttestations(
	plan BuildExecutionPlan,
	set ArtifactSet,
	attestations []BuildAttestation,
	requiredPolicies []AttestationTrustPolicy,
) (referenceAttestationSelection, error) {
	return selectReferenceAttestationsForSubject(plan, set, attestations, requiredPolicies,
		referenceAttestationScope{Kind: attestationSubjectArtifactSet})
}

// selectReferenceAttestationsForSubject requires explicit scope and issuer
// policy. This is Reference policy evaluation, not Runtime Acceptance.
func selectReferenceAttestationsForSubject(
	plan BuildExecutionPlan,
	set ArtifactSet,
	attestations []BuildAttestation,
	requiredPolicies []AttestationTrustPolicy,
	scope referenceAttestationScope,
) (referenceAttestationSelection, error) {
	subject, err := referenceRequiredAttestationSubject(set, scope)
	if err != nil {
		return referenceAttestationSelection{}, err
	}
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
		RequiredSubject: subject,
		Satisfied:       true,
		Issuers:         make([]referenceIssuerSelection, 0, len(policies)),
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
		candidates := byIssuer[policy.Issuer]
		matchingSubject := false
		for _, verified := range candidates {
			if normalizeAttestationSubject(verified.attestation.Subject) != normalizeAttestationSubject(subject) {
				continue
			}
			matchingSubject = true
			decision, err := evaluateAttestationTrust(verified.attestation, verified.verification, policy)
			if err != nil {
				return referenceAttestationSelection{}, err
			}
			selection.Reason = decision.Reason
			if decision.Trusted {
				selection.SelectedAttestationFingerprint = decision.AttestationFingerprint
				selection.Matched = true
				break
			}
		}
		if !matchingSubject && len(candidates) > 0 {
			selection.Reason = "required-subject-attestation-missing"
		}
		if !selection.Matched {
			result.Satisfied = false
		}
		result.Issuers = append(result.Issuers, selection)
	}
	return result, nil
}

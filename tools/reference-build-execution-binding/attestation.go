package main

import (
	"fmt"
	"sort"
	"strings"
)

const (
	buildAttestationFormatVersion       = 1
	attestationTrustPolicyFormatVersion = 1
	referenceFakeSignatureProfile       = "tei.reference.fake-signature/v1"
)

type AttestationSubjectKind string

const (
	attestationSubjectArtifactSet    AttestationSubjectKind = "artifact-set"
	attestationSubjectArtifactMember AttestationSubjectKind = "artifact-member"
)

type AttestationSubject struct {
	Kind                   AttestationSubjectKind `json:"kind"`
	ArtifactSetFingerprint string                 `json:"artifactSetFingerprint"`
	Role                   string                 `json:"role,omitempty"`
	ArtifactDigest         string                 `json:"artifactDigest,omitempty"`
}

type BuildAttestation struct {
	FormatVersion                   int                `json:"formatVersion"`
	Predicate                       string             `json:"predicate"`
	BuildExecutionResultFingerprint string             `json:"buildExecutionResultFingerprint"`
	Subject                         AttestationSubject `json:"subject"`
	Issuer                          string             `json:"issuer"`
	SignatureProfile                string             `json:"signatureProfile"`
	Signature                       string             `json:"signature"`
}

type attestationUnsignedPayload struct {
	FormatVersion                   int                `json:"formatVersion"`
	Predicate                       string             `json:"predicate"`
	BuildExecutionResultFingerprint string             `json:"buildExecutionResultFingerprint"`
	Subject                         AttestationSubject `json:"subject"`
	Issuer                          string             `json:"issuer"`
	SignatureProfile                string             `json:"signatureProfile"`
}

type AttestationTrustPolicy struct {
	FormatVersion    int      `json:"formatVersion"`
	Issuer           string   `json:"issuer"`
	SignatureProfile string   `json:"signatureProfile"`
	Predicates       []string `json:"predicates"`
	Revoked          bool     `json:"revoked"`
	Revision         string   `json:"revision"`
}

type AttestationTrustDecision struct {
	AttestationFingerprint string `json:"attestationFingerprint"`
	PolicyFingerprint      string `json:"policyFingerprint"`
	Trusted                bool   `json:"trusted"`
	Reason                 string `json:"reason"`
}

type BuildAttestationVerification struct {
	AttestationFingerprint          string `json:"attestationFingerprint"`
	BuildExecutionResultFingerprint string `json:"buildExecutionResultFingerprint"`
	ArtifactSetFingerprint          string `json:"artifactSetFingerprint"`
	Issuer                          string `json:"issuer"`
	SignatureProfile                string `json:"signatureProfile"`
	Predicate                       string `json:"predicate"`
}

func (s AttestationSubject) Validate() error {
	if err := validateFingerprint("attestation artifact set fingerprint", s.ArtifactSetFingerprint); err != nil {
		return err
	}
	switch s.Kind {
	case attestationSubjectArtifactSet:
		if s.Role != "" || s.ArtifactDigest != "" {
			return fmt.Errorf("artifact-set attestation subject must not declare member role or digest")
		}
	case attestationSubjectArtifactMember:
		if strings.TrimSpace(s.Role) == "" {
			return fmt.Errorf("artifact-member attestation subject role is required")
		}
		if err := validateDigest("artifact-member attestation digest", s.ArtifactDigest); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported attestation subject kind %q", s.Kind)
	}
	return nil
}

func (a BuildAttestation) Validate() error {
	if a.FormatVersion != buildAttestationFormatVersion {
		return fmt.Errorf("unsupported build attestation format %d", a.FormatVersion)
	}
	if strings.TrimSpace(a.Predicate) == "" {
		return fmt.Errorf("build attestation predicate is required")
	}
	if err := validateFingerprint("build attestation result fingerprint", a.BuildExecutionResultFingerprint); err != nil {
		return err
	}
	if err := a.Subject.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Issuer) == "" {
		return fmt.Errorf("build attestation issuer is required")
	}
	if strings.TrimSpace(a.SignatureProfile) == "" {
		return fmt.Errorf("build attestation signature profile is required")
	}
	if err := validateDigest("build attestation signature", a.Signature); err != nil {
		return err
	}
	return nil
}

func normalizeAttestationSubject(subject AttestationSubject) AttestationSubject {
	out := subject
	out.ArtifactSetFingerprint = normalizeFingerprint(subject.ArtifactSetFingerprint)
	out.ArtifactDigest = normalizeDigest(subject.ArtifactDigest)
	return out
}

func unsignedAttestationPayload(a BuildAttestation) attestationUnsignedPayload {
	return attestationUnsignedPayload{
		FormatVersion:                   a.FormatVersion,
		Predicate:                       a.Predicate,
		BuildExecutionResultFingerprint: normalizeFingerprint(a.BuildExecutionResultFingerprint),
		Subject:                         normalizeAttestationSubject(a.Subject),
		Issuer:                          a.Issuer,
		SignatureProfile:                a.SignatureProfile,
	}
}

func referenceFakeSignature(a BuildAttestation) (string, error) {
	if a.SignatureProfile != referenceFakeSignatureProfile {
		return "", fmt.Errorf("unsupported Reference fake signature profile %q", a.SignatureProfile)
	}
	payloadFingerprint, err := fingerprintJSON(unsignedAttestationPayload(a))
	if err != nil {
		return "", err
	}
	signatureFingerprint, err := fingerprintJSON(struct {
		Profile            string `json:"profile"`
		Issuer             string `json:"issuer"`
		PayloadFingerprint string `json:"payloadFingerprint"`
	}{
		Profile:            a.SignatureProfile,
		Issuer:             a.Issuer,
		PayloadFingerprint: payloadFingerprint,
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + signatureFingerprint, nil
}

func signReferenceAttestation(a BuildAttestation) (BuildAttestation, error) {
	a.SignatureProfile = referenceFakeSignatureProfile
	signature, err := referenceFakeSignature(a)
	if err != nil {
		return BuildAttestation{}, err
	}
	a.Signature = signature
	if err := a.Validate(); err != nil {
		return BuildAttestation{}, err
	}
	return a, nil
}

func buildAttestationFingerprint(a BuildAttestation) (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	normalized := a
	normalized.BuildExecutionResultFingerprint = normalizeFingerprint(a.BuildExecutionResultFingerprint)
	normalized.Subject = normalizeAttestationSubject(a.Subject)
	normalized.Signature = normalizeDigest(a.Signature)
	return fingerprintJSON(normalized)
}

func verifyBuildAttestation(
	a BuildAttestation,
	plan BuildExecutionPlan,
	set ArtifactSet,
) (BuildAttestationVerification, error) {
	if err := a.Validate(); err != nil {
		return BuildAttestationVerification{}, err
	}
	executionFingerprint, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		return BuildAttestationVerification{}, err
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		return BuildAttestationVerification{}, err
	}
	resultFingerprint, err := buildExecutionResultFingerprint(BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: executionFingerprint,
		ArtifactSetFingerprint:   setFingerprint,
	})
	if err != nil {
		return BuildAttestationVerification{}, err
	}
	if normalizeFingerprint(a.BuildExecutionResultFingerprint) != resultFingerprint {
		return BuildAttestationVerification{}, fmt.Errorf("attestation build execution result fingerprint does not match")
	}
	if normalizeFingerprint(a.Subject.ArtifactSetFingerprint) != setFingerprint {
		return BuildAttestationVerification{}, fmt.Errorf("attestation artifact set fingerprint does not match")
	}
	if a.Subject.Kind == attestationSubjectArtifactMember {
		member, ok := artifactMemberByRole(set, a.Subject.Role)
		if !ok {
			return BuildAttestationVerification{}, fmt.Errorf("attestation member role %s is not present in Artifact Set", a.Subject.Role)
		}
		if normalizeDigest(member.Digest) != normalizeDigest(a.Subject.ArtifactDigest) {
			return BuildAttestationVerification{}, fmt.Errorf("attestation member digest does not match Artifact Set")
		}
	}
	expectedSignature, err := referenceFakeSignature(a)
	if err != nil {
		return BuildAttestationVerification{}, err
	}
	if normalizeDigest(a.Signature) != normalizeDigest(expectedSignature) {
		return BuildAttestationVerification{}, fmt.Errorf("Reference fake attestation signature does not match immutable payload")
	}
	attestationFingerprint, err := buildAttestationFingerprint(a)
	if err != nil {
		return BuildAttestationVerification{}, err
	}
	return BuildAttestationVerification{
		AttestationFingerprint:          attestationFingerprint,
		BuildExecutionResultFingerprint: resultFingerprint,
		ArtifactSetFingerprint:          setFingerprint,
		Issuer:                          a.Issuer,
		SignatureProfile:                a.SignatureProfile,
		Predicate:                       a.Predicate,
	}, nil
}

func verifyBuildAttestationBinding(
	a BuildAttestation,
	plan BuildExecutionPlan,
	set ArtifactSet,
) error {
	_, err := verifyBuildAttestation(a, plan, set)
	return err
}

func (p AttestationTrustPolicy) Validate() error {
	if p.FormatVersion != attestationTrustPolicyFormatVersion {
		return fmt.Errorf("unsupported attestation trust policy format %d", p.FormatVersion)
	}
	if strings.TrimSpace(p.Issuer) == "" {
		return fmt.Errorf("attestation trust policy issuer is required")
	}
	if strings.TrimSpace(p.SignatureProfile) == "" {
		return fmt.Errorf("attestation trust policy signature profile is required")
	}
	if len(p.Predicates) == 0 {
		return fmt.Errorf("attestation trust policy requires at least one predicate")
	}
	seen := map[string]struct{}{}
	for _, predicate := range p.Predicates {
		if strings.TrimSpace(predicate) == "" {
			return fmt.Errorf("attestation trust policy predicate is required")
		}
		if _, duplicate := seen[predicate]; duplicate {
			return fmt.Errorf("attestation trust policy predicate %s is declared more than once", predicate)
		}
		seen[predicate] = struct{}{}
	}
	if strings.TrimSpace(p.Revision) == "" {
		return fmt.Errorf("attestation trust policy revision is required")
	}
	return nil
}

func attestationTrustPolicyFingerprint(policy AttestationTrustPolicy) (string, error) {
	if err := policy.Validate(); err != nil {
		return "", err
	}
	normalized := policy
	normalized.Predicates = append([]string(nil), policy.Predicates...)
	sort.Strings(normalized.Predicates)
	return fingerprintJSON(normalized)
}

func evaluateAttestationTrust(
	a BuildAttestation,
	verification BuildAttestationVerification,
	policy AttestationTrustPolicy,
) (AttestationTrustDecision, error) {
	attestationFingerprint, err := buildAttestationFingerprint(a)
	if err != nil {
		return AttestationTrustDecision{}, err
	}
	if normalizeFingerprint(verification.AttestationFingerprint) != attestationFingerprint ||
		normalizeFingerprint(verification.BuildExecutionResultFingerprint) != normalizeFingerprint(a.BuildExecutionResultFingerprint) ||
		normalizeFingerprint(verification.ArtifactSetFingerprint) != normalizeFingerprint(a.Subject.ArtifactSetFingerprint) ||
		verification.Issuer != a.Issuer ||
		verification.SignatureProfile != a.SignatureProfile ||
		verification.Predicate != a.Predicate {
		return AttestationTrustDecision{}, fmt.Errorf("attestation trust evaluation requires matching verified attestation evidence")
	}
	policyFingerprint, err := attestationTrustPolicyFingerprint(policy)
	if err != nil {
		return AttestationTrustDecision{}, err
	}
	decision := AttestationTrustDecision{
		AttestationFingerprint: attestationFingerprint,
		PolicyFingerprint:      policyFingerprint,
	}
	if policy.Revoked {
		decision.Reason = "issuer-revoked"
		return decision, nil
	}
	if a.Issuer != policy.Issuer {
		decision.Reason = "issuer-not-trusted"
		return decision, nil
	}
	if a.SignatureProfile != policy.SignatureProfile {
		decision.Reason = "signature-profile-not-trusted"
		return decision, nil
	}
	for _, predicate := range policy.Predicates {
		if a.Predicate == predicate {
			decision.Trusted = true
			decision.Reason = "trusted-by-current-policy"
			return decision, nil
		}
	}
	decision.Reason = "predicate-not-trusted"
	return decision, nil
}

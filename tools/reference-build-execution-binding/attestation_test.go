package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func attestationFixture(t *testing.T) (BuildExecutionPlan, ArtifactSet, string) {
	t.Helper()
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: "artifact", Locator: "source://artifact", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.attestation_reference"),
			Version:         "1.0.0",
			ContractVersion: "tei.reference.build/v1",
			ArtifactDigest:  digestB,
		},
		Recipe:         plugin.BuildRecipe{Kind: "attestation-reference", Entrypoint: "artifact"},
		OutputContract: "artifact-set",
		Target:         "host/reference",
	}
	executionFingerprint, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "primary", Format: "application/octet-stream", Digest: digestC},
			{Role: "manifest", Format: "application/json", Digest: digestD},
		},
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	resultFingerprint, err := buildExecutionResultFingerprint(BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: executionFingerprint,
		ArtifactSetFingerprint:   setFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan, set, resultFingerprint
}

func signedSetAttestation(t *testing.T, plan BuildExecutionPlan, set ArtifactSet, resultFingerprint string) BuildAttestation {
	t.Helper()
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	attestation, err := signReferenceAttestation(BuildAttestation{
		FormatVersion:                   buildAttestationFormatVersion,
		Predicate:                       "tei.reference.build-result/v1",
		BuildExecutionResultFingerprint: resultFingerprint,
		Subject: AttestationSubject{
			Kind:                   attestationSubjectArtifactSet,
			ArtifactSetFingerprint: setFingerprint,
		},
		Issuer: "tei.reference.issuer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBuildAttestationBinding(attestation, plan, set); err != nil {
		t.Fatal(err)
	}
	return attestation
}

func TestBuildAttestationBindsExactArtifactSetAndBuildResult(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)

	changed := set
	changed.Members = append([]ArtifactMember(nil), set.Members...)
	changed.Members[0].Digest = digestA
	if err := verifyBuildAttestationBinding(attestation, plan, changed); err == nil {
		t.Fatal("attestation accepted changed Artifact Set bytes")
	}

	attestation.BuildExecutionResultFingerprint = strings.Repeat("f", 64)
	if err := verifyBuildAttestationBinding(attestation, plan, set); err == nil {
		t.Fatal("attestation accepted a different BuildExecutionResult")
	}
}

func TestArtifactMemberAttestationDoesNotImplicitlyCoverOtherMember(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	setFingerprint, _ := artifactSetFingerprint(set)
	attestation, err := signReferenceAttestation(BuildAttestation{
		FormatVersion:                   buildAttestationFormatVersion,
		Predicate:                       "tei.reference.member/v1",
		BuildExecutionResultFingerprint: resultFingerprint,
		Subject: AttestationSubject{
			Kind:                   attestationSubjectArtifactMember,
			ArtifactSetFingerprint: setFingerprint,
			Role:                   "primary",
			ArtifactDigest:         digestC,
		},
		Issuer: "tei.reference.issuer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBuildAttestationBinding(attestation, plan, set); err != nil {
		t.Fatal(err)
	}

	other := attestation
	other.Subject.Role = "manifest"
	other.Subject.ArtifactDigest = digestD
	if err := verifyBuildAttestationBinding(other, plan, set); err == nil {
		t.Fatal("member attestation mutation was accepted without a new signature")
	}
}

func TestTrustRevocationChangesDecisionNotImmutableAttestation(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	attestationFingerprintBefore, err := buildAttestationFingerprint(attestation)
	if err != nil {
		t.Fatal(err)
	}

	policy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           "tei.reference.issuer",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates:       []string{"tei.reference.build-result/v1"},
		Revision:         "policy-1",
	}
	verification, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted.Trusted {
		t.Fatalf("trusted policy rejected attestation: %#v", accepted)
	}

	policy.Revoked = true
	policy.Revision = "policy-2"
	revoked, err := evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Trusted || revoked.Reason != "issuer-revoked" {
		t.Fatalf("revoked policy trusted attestation: %#v", revoked)
	}
	attestationFingerprintAfter, err := buildAttestationFingerprint(attestation)
	if err != nil {
		t.Fatal(err)
	}
	if attestationFingerprintAfter != attestationFingerprintBefore {
		t.Fatal("trust policy rotation changed immutable attestation identity")
	}
	if accepted.PolicyFingerprint == revoked.PolicyFingerprint {
		t.Fatal("trust policy revision/revocation did not change fresh verification identity")
	}
}

func TestWrongIssuerOrPredicateDoesNotBecomeTrusted(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	policy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           "tei.reference.other_issuer",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates:       []string{"tei.reference.build-result/v1"},
		Revision:         "policy-1",
	}
	verification, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Trusted || decision.Reason != "issuer-not-trusted" {
		t.Fatalf("wrong issuer was trusted: %#v", decision)
	}

	policy.Issuer = attestation.Issuer
	policy.Predicates = []string{"tei.reference.other-predicate/v1"}
	decision, err = evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Trusted || decision.Reason != "predicate-not-trusted" {
		t.Fatalf("wrong predicate was trusted: %#v", decision)
	}
}

func TestFakeSignatureProfileProvidesNoCryptoAuthorityShortcut(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	if attestation.SignatureProfile != referenceFakeSignatureProfile {
		t.Fatalf("unexpected fake signature profile %q", attestation.SignatureProfile)
	}
	if err := verifyBuildAttestationBinding(attestation, plan, set); err != nil {
		t.Fatal(err)
	}

	// Presence and deterministic verification of the Reference fake signature
	// deliberately says nothing about current trust. An explicit policy is
	// still required.
	policy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           "tei.reference.untrusted",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates:       []string{attestation.Predicate},
		Revision:         "policy-1",
	}
	verification, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Trusted {
		t.Fatal("fake signature presence granted trust without issuer policy")
	}
}

func TestTrustEvaluationRejectsMismatchedVerificationEvidence(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	verification, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	policy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           attestation.Issuer,
		SignatureProfile: attestation.SignatureProfile,
		Predicates:       []string{attestation.Predicate},
		Revision:         "policy-1",
	}

	tampered := verification
	tampered.BuildExecutionResultFingerprint = strings.Repeat("f", 64)
	if _, err := evaluateAttestationTrust(attestation, tampered, policy); err == nil {
		t.Fatal("trust evaluation accepted verification Evidence for a different build result")
	}

	tampered = verification
	tampered.ArtifactSetFingerprint = strings.Repeat("e", 64)
	if _, err := evaluateAttestationTrust(attestation, tampered, policy); err == nil {
		t.Fatal("trust evaluation accepted verification Evidence for a different Artifact Set")
	}

	tampered = verification
	tampered.Issuer = "tei.reference.other_issuer"
	if _, err := evaluateAttestationTrust(attestation, tampered, policy); err == nil {
		t.Fatal("trust evaluation accepted verification Evidence for a different issuer")
	}
}

func TestTrustEvaluationCanonicalizesVerificationFingerprintHexCase(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	verification, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	verification.AttestationFingerprint = strings.ToUpper(verification.AttestationFingerprint)
	verification.BuildExecutionResultFingerprint = strings.ToUpper(verification.BuildExecutionResultFingerprint)
	verification.ArtifactSetFingerprint = strings.ToUpper(verification.ArtifactSetFingerprint)

	policy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           attestation.Issuer,
		SignatureProfile: attestation.SignatureProfile,
		Predicates:       []string{attestation.Predicate},
		Revision:         "policy-1",
	}
	decision, err := evaluateAttestationTrust(attestation, verification, policy)
	if err != nil {
		t.Fatalf("valid fingerprint hex case was rejected: %v", err)
	}
	if !decision.Trusted {
		t.Fatalf("valid fingerprint hex case changed Trust decision: %#v", decision)
	}
}

func TestIndependentAttestationsForSameArtifactSet(t *testing.T) {
	plan, set, resultFingerprint := attestationFixture(t)
	first := signedSetAttestation(t, plan, set, resultFingerprint)
	second := first
	second.Issuer = "tei.reference.independent_issuer"
	var err error
	second, err = signReferenceAttestation(second)
	if err != nil {
		t.Fatal(err)
	}

	// The same physical artifact may have multiple immutable attestations.
	// Issuer identity belongs to the signed payload, not to the artifact.
	attestations := []BuildAttestation{first, second}
	if len(attestations) != 2 {
		t.Fatal("one attestation was discarded")
	}
	firstFingerprint, err := buildAttestationFingerprint(first)
	if err != nil {
		t.Fatal(err)
	}
	secondFingerprint, err := buildAttestationFingerprint(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstFingerprint == secondFingerprint {
		t.Fatal("different issuers were collapsed into one attestation identity")
	}

	firstVerification, err := verifyBuildAttestation(first, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	secondVerification, err := verifyBuildAttestation(second, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	policyA := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           first.Issuer,
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates:       []string{first.Predicate},
		Revision:         "issuer-a-1",
	}
	policyB := policyA
	policyB.Issuer = second.Issuer
	policyB.Revision = "issuer-b-1"

	for _, tc := range []struct {
		name         string
		attestation  BuildAttestation
		verification BuildAttestationVerification
		policy       AttestationTrustPolicy
		wantTrusted  bool
	}{
		{name: "issuer A own policy", attestation: first, verification: firstVerification, policy: policyA, wantTrusted: true},
		{name: "issuer B own policy", attestation: second, verification: secondVerification, policy: policyB, wantTrusted: true},
		{name: "issuer A under B policy", attestation: first, verification: firstVerification, policy: policyB},
		{name: "issuer B under A policy", attestation: second, verification: secondVerification, policy: policyA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := evaluateAttestationTrust(tc.attestation, tc.verification, tc.policy)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Trusted != tc.wantTrusted {
				t.Fatalf("issuer-specific trust leaked: trusted=%v, want %v", decision.Trusted, tc.wantTrusted)
			}
		})
	}

	// Rotation/revocation is a fresh policy decision for one issuer.
	// It neither mutates its attestation nor revokes an unrelated issuer.
	policyA.Revoked = true
	policyA.Revision = "issuer-a-2"
	revokedA, err := evaluateAttestationTrust(first, firstVerification, policyA)
	if err != nil {
		t.Fatal(err)
	}
	independentB, err := evaluateAttestationTrust(second, secondVerification, policyB)
	if err != nil {
		t.Fatal(err)
	}
	if revokedA.Trusted || revokedA.Reason != "issuer-revoked" || !independentB.Trusted {
		t.Fatalf("revocation affected another issuer: A=%#v B=%#v", revokedA, independentB)
	}
	after, err := buildAttestationFingerprint(first)
	if err != nil {
		t.Fatal(err)
	}
	if after != firstFingerprint {
		t.Fatal("issuer revocation modified an immutable attestation")
	}

	// Swapping issuer metadata without a new signature is rejected at the
	// binding-verification step. This is only a fake-signature Reference.
	tampered := first
	tampered.Issuer = second.Issuer
	if _, err := verifyBuildAttestation(tampered, plan, set); err == nil {
		t.Fatal("issuer substitution reused a signature for a different payload")
	}
}

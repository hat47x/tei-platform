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
			ID: core.MustCanonicalID("tei.builder.attestation_reference"),
			Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1",
			ArtifactDigest: digestB,
		},
		Recipe: plugin.BuildRecipe{Kind: "attestation-reference", Entrypoint: "artifact"},
		OutputContract: "artifact-set",
		Target: "host/reference",
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
		FormatVersion: attestationTrustPolicyFormatVersion,
		Issuer: "tei.reference.issuer",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates: []string{"tei.reference.build-result/v1"},
		Revision: "policy-1",
	}
	accepted, err := evaluateAttestationTrust(attestation, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !accepted.Trusted {
		t.Fatalf("trusted policy rejected attestation: %#v", accepted)
	}

	policy.Revoked = true
	policy.Revision = "policy-2"
	revoked, err := evaluateAttestationTrust(attestation, policy)
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
		FormatVersion: attestationTrustPolicyFormatVersion,
		Issuer: "tei.reference.other_issuer",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates: []string{"tei.reference.build-result/v1"},
		Revision: "policy-1",
	}
	decision, err := evaluateAttestationTrust(attestation, policy)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Trusted || decision.Reason != "issuer-not-trusted" {
		t.Fatalf("wrong issuer was trusted: %#v", decision)
	}

	policy.Issuer = attestation.Issuer
	policy.Predicates = []string{"tei.reference.other-predicate/v1"}
	decision, err = evaluateAttestationTrust(attestation, policy)
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
		FormatVersion: attestationTrustPolicyFormatVersion,
		Issuer: "tei.reference.untrusted",
		SignatureProfile: referenceFakeSignatureProfile,
		Predicates: []string{attestation.Predicate},
		Revision: "policy-1",
	}
	decision, err := evaluateAttestationTrust(attestation, policy)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Trusted {
		t.Fatal("fake signature presence granted trust without issuer policy")
	}
}

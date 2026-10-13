package main

import (
	"strings"
	"testing"
)

func signedReferenceMember(t *testing.T, set ArtifactSet, original BuildAttestation, role string) BuildAttestation {
	t.Helper()
	member, ok := artifactMemberByRole(set, role)
	if !ok {
		t.Fatalf("fixture member %q does not exist", role)
	}
	attestation := original
	attestation.Subject.Kind = attestationSubjectArtifactMember
	attestation.Subject.Role = role
	attestation.Subject.ArtifactDigest = member.Digest
	resigned, err := signReferenceAttestation(attestation)
	if err != nil {
		t.Fatal(err)
	}
	return resigned
}

func TestReferenceSetRequirementRejectsMemberOnlyAttestation(t *testing.T) {
	plan, set, wholeSet, _, policy, _ := referenceSelectionFixture(t)
	member := signedReferenceMember(t, set, wholeSet, "primary")
	memberFingerprint, err := buildAttestationFingerprint(member)
	if err != nil {
		t.Fatal(err)
	}

	memberOnly, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{member},
		[]AttestationTrustPolicy{policy},
	)
	if err != nil {
		t.Fatal(err)
	}
	if memberOnly.Satisfied || len(memberOnly.Issuers) != 1 ||
		memberOnly.Issuers[0].Reason != "required-subject-attestation-missing" ||
		memberOnly.Issuers[0].SelectedAttestationFingerprint != "" {
		t.Fatalf("member proof was promoted to whole-set acceptance: %#v", memberOnly)
	}
	if memberOnly.RequiredSubject.Kind != attestationSubjectArtifactSet {
		t.Fatalf("implicit whole-set requirement lost: %#v", memberOnly.RequiredSubject)
	}

	both, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{member, wholeSet},
		[]AttestationTrustPolicy{policy},
	)
	if err != nil || !both.Satisfied {
		t.Fatalf("valid whole-set attestation was not selected: %#v err=%v", both, err)
	}
	if both.Issuers[0].SelectedAttestationFingerprint == memberFingerprint {
		t.Fatal("whole-set decision selected a member fingerprint")
	}
}

func TestReferenceMemberRequirementDoesNotInheritWholeSetOrOtherRole(t *testing.T) {
	plan, set, wholeSet, _, policy, _ := referenceSelectionFixture(t)
	primary := signedReferenceMember(t, set, wholeSet, "primary")
	manifest := signedReferenceMember(t, set, wholeSet, "manifest")
	scope := referenceAttestationScope{Kind: attestationSubjectArtifactMember, Role: "primary"}

	for _, tc := range []struct {
		name  string
		proof BuildAttestation
	}{
		{name: "whole-set proof", proof: wholeSet},
		{name: "other role proof", proof: manifest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := selectReferenceAttestationsForSubject(plan, set,
				[]BuildAttestation{tc.proof},
				[]AttestationTrustPolicy{policy},
				scope,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Satisfied ||
				result.Issuers[0].Reason != "required-subject-attestation-missing" ||
				result.Issuers[0].SelectedAttestationFingerprint != "" {
				t.Fatalf("%s satisfied primary member: %#v", tc.name, result)
			}
		})
	}

	selected, err := selectReferenceAttestationsForSubject(plan, set,
		[]BuildAttestation{manifest, wholeSet, primary},
		[]AttestationTrustPolicy{policy},
		scope,
	)
	if err != nil || !selected.Satisfied || !selected.Issuers[0].Matched {
		t.Fatalf("actual primary member proof not selected: %#v err=%v", selected, err)
	}
	digest := selected.RequiredSubject.ArtifactDigest
	if selected.RequiredSubject.Kind != attestationSubjectArtifactMember ||
		selected.RequiredSubject.Role != "primary" || digest != digestC {
		t.Fatalf("required member was not tied to the actual artifact digest: %#v", selected.RequiredSubject)
	}
	fingerprint, err := buildAttestationFingerprint(primary)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Issuers[0].SelectedAttestationFingerprint != fingerprint {
		t.Fatal("selected attestation did not have exact required member subject")
	}
}

func TestReferenceSelectionRejectsInvalidSubjectRequirements(t *testing.T) {
	plan, set, proof, _, policy, _ := referenceSelectionFixture(t)
	for _, tc := range []struct {
		name  string
		scope referenceAttestationScope
		want  string
	}{
		{name: "no subject kind", want: "unsupported required"},
		{name: "unknown subject kind", scope: referenceAttestationScope{Kind: "unknown"}, want: "unsupported required"},
		{name: "set with member role", scope: referenceAttestationScope{Kind: attestationSubjectArtifactSet, Role: "primary"}, want: "must not specify"},
		{name: "member with no role", scope: referenceAttestationScope{Kind: attestationSubjectArtifactMember}, want: "needs a member role"},
		{name: "member with unknown role", scope: referenceAttestationScope{Kind: attestationSubjectArtifactMember, Role: "missing"}, want: "absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := selectReferenceAttestationsForSubject(plan, set,
				[]BuildAttestation{proof},
				[]AttestationTrustPolicy{policy},
				tc.scope,
			)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid required subject accepted: %v", err)
			}
		})
	}
}

func TestReferenceSelectionNeverLabelsRejectedAttestationAsSelected(t *testing.T) {
	plan, set, proof, _, policy, _ := referenceSelectionFixture(t)
	policy.Predicates = []string{"tei.reference.untrusted-predicate/v1"}
	decision, err := selectReferenceAttestations(plan, set, []BuildAttestation{proof},
		[]AttestationTrustPolicy{policy})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Satisfied || decision.Issuers[0].Matched ||
		decision.Issuers[0].Reason != "predicate-not-trusted" ||
		decision.Issuers[0].SelectedAttestationFingerprint != "" {
		t.Fatalf("untrusted candidate was labeled selected: %#v", decision)
	}
}

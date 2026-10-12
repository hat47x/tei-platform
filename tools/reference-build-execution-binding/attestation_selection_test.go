package main

import (
	"reflect"
	"strings"
	"testing"
)

func referenceSelectionFixture(t *testing.T) (BuildExecutionPlan, ArtifactSet, BuildAttestation, BuildAttestation, AttestationTrustPolicy, AttestationTrustPolicy) {
	t.Helper()
	plan, set, resultFingerprint := attestationFixture(t)
	first := signedSetAttestation(t, plan, set, resultFingerprint)
	second := first
	second.Issuer = "tei.reference.zzz_issuer"
	var err error
	second, err = signReferenceAttestation(second)
	if err != nil {
		t.Fatal(err)
	}
	firstPolicy := AttestationTrustPolicy{
		FormatVersion:    attestationTrustPolicyFormatVersion,
		Issuer:           first.Issuer,
		SignatureProfile: first.SignatureProfile,
		Predicates:       []string{first.Predicate},
		Revision:         "issuer-a-1",
	}
	secondPolicy := firstPolicy
	secondPolicy.Issuer = second.Issuer
	secondPolicy.Revision = "issuer-b-1"
	return plan, set, first, second, firstPolicy, secondPolicy
}

func TestReferenceRequiredIssuerSelectionIsIndependentOfInputOrder(t *testing.T) {
	plan, set, first, second, policyA, policyB := referenceSelectionFixture(t)
	forward, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, second},
		[]AttestationTrustPolicy{policyB, policyA},
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{second, first, first},
		[]AttestationTrustPolicy{policyA, policyB},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !forward.Satisfied || !reversed.Satisfied || !reflect.DeepEqual(forward, reversed) {
		t.Fatalf("same valid issuer set produced different decisions: forward=%#v reversed=%#v", forward, reversed)
	}
	if len(forward.Issuers) != 2 ||
		!forward.Issuers[0].Matched ||
		!forward.Issuers[1].Matched {
		t.Fatalf("each independent required issuer must match: %#v", forward)
	}
}

func TestReferenceSelectionDoesNotInferRequiredIssuersFromSignatures(t *testing.T) {
	plan, set, first, second, policyA, policyB := referenceSelectionFixture(t)
	onlyA, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first},
		[]AttestationTrustPolicy{policyA},
	)
	if err != nil || !onlyA.Satisfied {
		t.Fatalf("only explicitly required issuer A should suffice: selection=%#v err=%v", onlyA, err)
	}
	bothRequired, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, first},
		[]AttestationTrustPolicy{policyA, policyB},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bothRequired.Satisfied || bothRequired.Issuers[1].Reason != "required-issuer-attestation-missing" {
		t.Fatalf("duplicate issuer A counted as missing B: %#v", bothRequired)
	}
	wrongPolicy := policyB
	wrongPolicy.Predicates = []string{"tei.reference.unrelated-predicate/v1"}
	wrongPolicy.Revision = "issuer-b-2"
	untrusted, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, second},
		[]AttestationTrustPolicy{policyA, wrongPolicy},
	)
	if err != nil {
		t.Fatal(err)
	}
	if untrusted.Satisfied || untrusted.Issuers[1].Reason != "predicate-not-trusted" {
		t.Fatalf("a present but untrusted attestation was accepted: %#v", untrusted)
	}
}

func TestReferenceSelectionRevocationChangesPolicyNotImmutableAttestation(t *testing.T) {
	plan, set, first, second, policyA, policyB := referenceSelectionFixture(t)
	prior, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, second},
		[]AttestationTrustPolicy{policyA, policyB},
	)
	if err != nil || !prior.Satisfied {
		t.Fatalf("baseline failed: decision=%#v err=%v", prior, err)
	}
	policyA.Revoked = true
	policyA.Revision = "issuer-a-2"
	revoked, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, second},
		[]AttestationTrustPolicy{policyA, policyB},
	)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Satisfied || revoked.Issuers[0].Reason != "issuer-revoked" ||
		!revoked.Issuers[1].Matched {
		t.Fatalf("revocation was not confined to issuer A: %#v", revoked)
	}
	if revoked.Issuers[0].SelectedAttestationFingerprint != prior.Issuers[0].SelectedAttestationFingerprint ||
		revoked.Issuers[1].SelectedAttestationFingerprint != prior.Issuers[1].SelectedAttestationFingerprint {
		t.Fatal("fresh trust decision modified immutable attestation identities")
	}
	if revoked.Issuers[0].PolicyFingerprint == prior.Issuers[0].PolicyFingerprint {
		t.Fatal("changed policy revision/revocation did not change policy identity")
	}
}

func TestReferenceSelectionRejectsAmbiguousPolicyAndInvalidBinding(t *testing.T) {
	plan, set, first, second, policyA, policyB := referenceSelectionFixture(t)
	for _, tc := range []struct {
		name string
		policies []AttestationTrustPolicy
		want string
	}{
		{name: "no explicit policy", want: "explicit issuer policy"},
		{name: "duplicate issuer", policies: []AttestationTrustPolicy{policyA, policyA}, want: "more than once"},
		{name: "invalid policy", policies: []AttestationTrustPolicy{{Issuer: policyA.Issuer}}, want: "format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := selectReferenceAttestations(plan, set, []BuildAttestation{first, second}, tc.policies)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid explicit policy was accepted: %v", err)
			}
		})
	}
	tampered := second
	tampered.Issuer = first.Issuer
	if _, err := selectReferenceAttestations(plan, set,
		[]BuildAttestation{first, tampered},
		[]AttestationTrustPolicy{policyA, policyB},
	); err == nil {
		t.Fatal("a forged attestation must not count as a required issuer")
	}
	wrongSet := set
	wrongSet.Members = append([]ArtifactMember(nil), set.Members...)
	wrongSet.Members[0].Digest = digestA
	if _, err := selectReferenceAttestations(plan, wrongSet,
		[]BuildAttestation{first, second},
		[]AttestationTrustPolicy{policyA, policyB},
	); err == nil {
		t.Fatal("attestation from a different Artifact Set was accepted")
	}
}

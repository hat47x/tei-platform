package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func externalProvenanceFixture(t *testing.T) (BuildExecutionPlan, ArtifactSet, BuildAttestation, externalProvenanceClaim) {
	t.Helper()
	plan, set, resultFingerprint := attestationFixture(t)
	attestation := signedSetAttestation(t, plan, set, resultFingerprint)
	verified, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	claim := externalProvenanceClaim{
		Profile:                         "tei.reference.external-provenance/v0",
		AttestationFingerprint:          verified.AttestationFingerprint,
		BuildExecutionResultFingerprint: verified.BuildExecutionResultFingerprint,
		ArtifactSetFingerprint:          verified.ArtifactSetFingerprint,
		Issuer:                          verified.Issuer,
		SignatureProfile:                verified.SignatureProfile,
		Predicate:                       verified.Predicate,
	}
	return plan, set, attestation, claim
}

func externalDocumentAndBinding(t *testing.T, claim externalProvenanceClaim) ([]byte, externalProvenanceBinding) {
	t.Helper()
	document, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	return document, externalBindingForBytes(claim.Profile, document)
}

func externalBindingForBytes(profile string, document []byte) externalProvenanceBinding {
	sum := sha256.Sum256(document)
	return externalProvenanceBinding{
		Profile:       profile,
		ContentDigest: "sha256:" + hex.EncodeToString(sum[:]),
	}
}

func TestExternalProvenanceContentAddressBindsVerifiedSubject(t *testing.T) {
	plan, set, attestation, claim := externalProvenanceFixture(t)
	document, binding := externalDocumentAndBinding(t, claim)
	evidence, err := verifyExternalProvenanceDocument(document, binding, attestation, plan, set)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.ContentDigest != binding.ContentDigest ||
		evidence.Profile != claim.Profile ||
		evidence.AttestationFingerprint != claim.AttestationFingerprint {
		t.Fatalf("external provenance binding evidence changed: %#v", evidence)
	}
	// A locator or a document carrying plausible issuer fields never
	// substitutes for the exact bytes addressed by the content digest.
	mutated := append(append([]byte(nil), document...), '\n')
	if _, err := verifyExternalProvenanceDocument(mutated, binding, attestation, plan, set); err == nil ||
		!strings.Contains(err.Error(), "content digest") {
		t.Fatalf("modified document bytes were accepted: %v", err)
	}
}

func TestExternalProvenanceRejectsSubjectAndIssuerSubstitution(t *testing.T) {
	plan, set, attestation, claim := externalProvenanceFixture(t)
	for _, tc := range []struct {
		name   string
		mutate func(*externalProvenanceClaim)
	}{
		{name: "wrong issuer", mutate: func(c *externalProvenanceClaim) { c.Issuer = "tei.reference.other_issuer" }},
		{name: "wrong predicate", mutate: func(c *externalProvenanceClaim) { c.Predicate = "tei.reference.other-predicate/v1" }},
		{name: "wrong signature profile", mutate: func(c *externalProvenanceClaim) { c.SignatureProfile = "tei.reference.other-signature/v1" }},
		{name: "wrong attestation", mutate: func(c *externalProvenanceClaim) { c.AttestationFingerprint = strings.Repeat("f", 64) }},
		{name: "wrong build result", mutate: func(c *externalProvenanceClaim) { c.BuildExecutionResultFingerprint = strings.Repeat("f", 64) }},
		{name: "wrong artifact set", mutate: func(c *externalProvenanceClaim) { c.ArtifactSetFingerprint = strings.Repeat("f", 64) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := claim
			tc.mutate(&changed)
			document, binding := externalDocumentAndBinding(t, changed)
			// Even a correctly recomputed content digest cannot authorize
			// an external claim about some other subject or issuer.
			_, err := verifyExternalProvenanceDocument(document, binding, attestation, plan, set)
			if err == nil || !strings.Contains(err.Error(), "does not match") {
				t.Fatalf("substituted provenance claim was accepted: %v", err)
			}
		})
	}

	document, binding := externalDocumentAndBinding(t, claim)
	binding.Profile = "tei.reference.unrelated-profile/v0"
	if _, err := verifyExternalProvenanceDocument(document, binding, attestation, plan, set); err == nil {
		t.Fatal("external provenance profile mismatch was accepted")
	}

	alteredSet := set
	alteredSet.Members = append([]ArtifactMember(nil), set.Members...)
	alteredSet.Members[0].Digest = digestA
	binding.Profile = claim.Profile
	if _, err := verifyExternalProvenanceDocument(document, binding, attestation, plan, alteredSet); err == nil {
		t.Fatal("external provenance accepted a different actual Artifact Set")
	}
}

func TestExternalProvenanceStrictDocumentBoundary(t *testing.T) {
	plan, set, attestation, claim := externalProvenanceFixture(t)
	document, _ := externalDocumentAndBinding(t, claim)
	extraField := strings.Replace(string(document), `"issuer":`, `"notInProfile":"unexpected","issuer":`, 1)
	duplicateField := strings.Replace(string(document), `"issuer":`, `"issuer":"tei.reference.forged","issuer":`, 1)

	for _, tc := range []struct {
		name      string
		document  []byte
		wantError string
	}{
		{name: "duplicate JSON key", document: []byte(duplicateField), wantError: "duplicate field"},
		{name: "unknown JSON key", document: []byte(extraField), wantError: "unknown field"},
		{name: "nonobject JSON", document: []byte("[]"), wantError: "JSON object"},
		{name: "trailing document", document: append(append([]byte(nil), document...), []byte(" []")...), wantError: "trailing content"},
		{name: "oversized document", document: []byte(strings.Repeat(" ", maxExternalProvenanceDocumentBytes+1)), wantError: "at most"},
		{name: "missing field", document: []byte(`{"profile":"tei.reference.external-provenance/v0"}`), wantError: "does not match"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := externalBindingForBytes(claim.Profile, tc.document)
			_, err := verifyExternalProvenanceDocument(tc.document, binding, attestation, plan, set)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected %q for invalid provenance document, got %v", tc.wantError, err)
			}
		})
	}
}

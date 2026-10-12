package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const maxExternalProvenanceDocumentBytes = 64 << 10

// externalProvenanceClaim is a deliberately small, flat Reference projection.
// Its contents are not proof of an issuer's identity or signing authority.
type externalProvenanceClaim struct {
	Profile                         string `json:"profile"`
	AttestationFingerprint          string `json:"attestationFingerprint"`
	BuildExecutionResultFingerprint string `json:"buildExecutionResultFingerprint"`
	ArtifactSetFingerprint          string `json:"artifactSetFingerprint"`
	Issuer                          string `json:"issuer"`
	SignatureProfile                string `json:"signatureProfile"`
	Predicate                       string `json:"predicate"`
}

type externalProvenanceBinding struct {
	Profile       string `json:"profile"`
	ContentDigest string `json:"contentDigest"`
}

// externalProvenanceEvidence is binding evidence only. It MUST NOT be used
// as current Trust Acceptance, signature authenticity, or Runtime Authority.
type externalProvenanceEvidence struct {
	Profile                string `json:"profile"`
	ContentDigest          string `json:"contentDigest"`
	AttestationFingerprint string `json:"attestationFingerprint"`
}

func decodeExternalProvenanceDocument(document []byte) (externalProvenanceClaim, error) {
	if len(document) == 0 || len(document) > maxExternalProvenanceDocumentBytes {
		return externalProvenanceClaim{}, fmt.Errorf("external provenance document must be nonempty and at most %d bytes", maxExternalProvenanceDocumentBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(document))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return externalProvenanceClaim{}, fmt.Errorf("external provenance document must be a JSON object")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return externalProvenanceClaim{}, fmt.Errorf("external provenance field name: %w", err)
		}
		name, ok := token.(string)
		if !ok {
			return externalProvenanceClaim{}, fmt.Errorf("external provenance field name must be a string")
		}
		if _, duplicate := fields[name]; duplicate {
			return externalProvenanceClaim{}, fmt.Errorf("external provenance duplicate field %q", name)
		}
		switch name {
		case "profile", "attestationFingerprint", "buildExecutionResultFingerprint", "artifactSetFingerprint", "issuer", "signatureProfile", "predicate":
		default:
			return externalProvenanceClaim{}, fmt.Errorf("external provenance unknown field %q", name)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return externalProvenanceClaim{}, fmt.Errorf("external provenance field %q: %w", name, err)
		}
		fields[name] = value
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return externalProvenanceClaim{}, fmt.Errorf("external provenance object is not closed")
	}
	var tail json.RawMessage
	if err := decoder.Decode(&tail); err != io.EOF {
		return externalProvenanceClaim{}, fmt.Errorf("external provenance document has trailing content")
	}
	normalized, err := json.Marshal(fields)
	if err != nil {
		return externalProvenanceClaim{}, err
	}
	var claim externalProvenanceClaim
	if err := json.Unmarshal(normalized, &claim); err != nil {
		return externalProvenanceClaim{}, fmt.Errorf("external provenance claim: %w", err)
	}
	return claim, nil
}

func verifyExternalProvenanceDocument(
	document []byte,
	binding externalProvenanceBinding,
	attestation BuildAttestation,
	plan BuildExecutionPlan,
	set ArtifactSet,
) (externalProvenanceEvidence, error) {
	if strings.TrimSpace(binding.Profile) == "" {
		return externalProvenanceEvidence{}, fmt.Errorf("external provenance profile is required")
	}
	if err := validateDigest("external provenance content digest", binding.ContentDigest); err != nil {
		return externalProvenanceEvidence{}, err
	}
	sum := sha256.Sum256(document)
	actualDigest := "sha256:" + hex.EncodeToString(sum[:])
	if normalizeDigest(binding.ContentDigest) != actualDigest {
		return externalProvenanceEvidence{}, fmt.Errorf("external provenance content digest does not match document bytes")
	}
	claim, err := decodeExternalProvenanceDocument(document)
	if err != nil {
		return externalProvenanceEvidence{}, err
	}
	verified, err := verifyBuildAttestation(attestation, plan, set)
	if err != nil {
		return externalProvenanceEvidence{}, fmt.Errorf("external provenance requires verified attestation binding: %w", err)
	}
	if claim.Profile != binding.Profile ||
		normalizeFingerprint(claim.AttestationFingerprint) != verified.AttestationFingerprint ||
		normalizeFingerprint(claim.BuildExecutionResultFingerprint) != verified.BuildExecutionResultFingerprint ||
		normalizeFingerprint(claim.ArtifactSetFingerprint) != verified.ArtifactSetFingerprint ||
		claim.Issuer != verified.Issuer ||
		claim.SignatureProfile != verified.SignatureProfile ||
		claim.Predicate != verified.Predicate {
		return externalProvenanceEvidence{}, fmt.Errorf("external provenance subject, issuer, or profile does not match verified attestation")
	}
	return externalProvenanceEvidence{
		Profile:                binding.Profile,
		ContentDigest:          actualDigest,
		AttestationFingerprint: verified.AttestationFingerprint,
	}, nil
}

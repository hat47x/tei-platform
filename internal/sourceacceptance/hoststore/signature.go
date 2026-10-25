package hoststore

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// The signature scheme below is a PROVISIONAL Host policy (see
// plan/workstreams/TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md): an
// issuer signs a Decision with an Ed25519 private key it keeps outside the
// repository and the Host, and the Host trusts issuers listed in a trust
// file. Holding a signature is evidence that a key holder issued the Decision;
// it is not, by itself, evidence of who the human was.

const (
	signedPayloadSchema = "tei.host-acceptance-decision/v0"
	trustSchema         = "tei.host-acceptance-trust/v0"
	algorithmEd25519    = "ed25519"
)

// signatureEvidence is the Decision.Evidence shape produced by Sign.
type signatureEvidence struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

// SigningPayload returns the exact bytes an issuer signs: a domain-separated
// JSON document of the issuer and the Record with members sorted by path, so
// the same Decision always yields the same bytes.
func SigningPayload(d Decision) ([]byte, error) {
	record := cloneRecord(d.Record)
	sort.Slice(record.Artifacts, func(i, j int) bool { return record.Artifacts[i].Path < record.Artifacts[j].Path })
	return json.Marshal(struct {
		Schema string                  `json:"schema"`
		Issuer string                  `json:"issuer"`
		Record sourceacceptance.Record `json:"record"`
	}{signedPayloadSchema, d.Issuer, record})
}

// Sign attaches an Ed25519 signature as the Decision's Evidence. It is used by
// an issuer, never by the Host or the Store.
func Sign(d Decision, keyID string, key ed25519.PrivateKey) (Decision, error) {
	if strings.TrimSpace(keyID) == "" || len(key) != ed25519.PrivateKeySize {
		return Decision{}, errors.New("HOST_STORE_SIGNING_KEY_INVALID")
	}
	payload, err := SigningPayload(d)
	if err != nil {
		return Decision{}, err
	}
	evidence, err := json.Marshal(signatureEvidence{
		Algorithm: algorithmEd25519, KeyID: keyID,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, payload)),
	})
	if err != nil {
		return Decision{}, err
	}
	d.Evidence = evidence
	return d, nil
}

// TrustedIssuer is one entry of the Host's trust file.
type TrustedIssuer struct {
	Issuer       string   `json:"issuer"`
	KeyID        string   `json:"keyId"`
	PublicKey    string   `json:"publicKey"`
	Applications []string `json:"applications"`
}

// TrustPolicy is the Host's list of issuers whose signed Decisions it accepts.
type TrustPolicy struct {
	Schema  string          `json:"schema"`
	Issuers []TrustedIssuer `json:"issuers"`
}

// LoadTrustPolicy reads and validates a trust file. Errors do not include the
// file's contents.
func LoadTrustPolicy(path string) (TrustPolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TrustPolicy{}, errors.New("HOST_STORE_TRUST_UNREADABLE")
	}
	return ParseTrustPolicy(raw)
}

// ParseTrustPolicy validates a trust document: known schema, at least one
// issuer, a 32-byte public key, a non-empty key ID and Application list, and
// no duplicate issuer/key pair.
func ParseTrustPolicy(raw []byte) (TrustPolicy, error) {
	var policy TrustPolicy
	if err := json.Unmarshal(raw, &policy); err != nil || policy.Schema != trustSchema || len(policy.Issuers) == 0 {
		return TrustPolicy{}, errors.New("HOST_STORE_TRUST_INVALID")
	}
	seen := map[string]bool{}
	for _, issuer := range policy.Issuers {
		key, err := base64.StdEncoding.DecodeString(issuer.PublicKey)
		id := issuer.Issuer + "\x00" + issuer.KeyID
		if err != nil || len(key) != ed25519.PublicKeySize || strings.TrimSpace(issuer.Issuer) == "" ||
			strings.TrimSpace(issuer.KeyID) == "" || len(issuer.Applications) == 0 || seen[id] {
			return TrustPolicy{}, errors.New("HOST_STORE_TRUST_INVALID")
		}
		seen[id] = true
	}
	return policy, nil
}

// Verifier returns a DecisionVerifier that accepts a Decision only when a
// listed issuer, with the key ID named in the Evidence, signed exactly this
// Decision for an Application it is trusted for.
func (p TrustPolicy) Verifier() DecisionVerifier {
	issuers := append([]TrustedIssuer(nil), p.Issuers...)
	return func(_ context.Context, d Decision) error {
		var evidence signatureEvidence
		if err := json.Unmarshal(d.Evidence, &evidence); err != nil || evidence.Algorithm != algorithmEd25519 {
			return errors.New("HOST_STORE_SIGNATURE_MISSING")
		}
		signature, err := base64.StdEncoding.DecodeString(evidence.Signature)
		if err != nil || len(signature) != ed25519.SignatureSize {
			return errors.New("HOST_STORE_SIGNATURE_MALFORMED")
		}
		payload, err := SigningPayload(d)
		if err != nil {
			return err
		}
		for _, issuer := range issuers {
			if issuer.Issuer != d.Issuer || issuer.KeyID != evidence.KeyID || !contains(issuer.Applications, d.Record.ApplicationID) {
				continue
			}
			key, _ := base64.StdEncoding.DecodeString(issuer.PublicKey)
			if ed25519.Verify(ed25519.PublicKey(key), payload, signature) {
				return nil
			}
			return errors.New("HOST_STORE_SIGNATURE_INVALID")
		}
		return fmt.Errorf("HOST_STORE_ISSUER_NOT_TRUSTED")
	}
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// RecordForIndex builds the Record an issuer signs for a loaded Source Index.
func RecordForIndex(applicationID, decisionRef, authorityRevision string, index sourceindex.Index) sourceacceptance.Record {
	artifacts := append([]sourceindex.Artifact(nil), index.Artifacts...)
	return sourceacceptance.Record{
		ApplicationID: applicationID, DecisionRef: decisionRef, AuthorityRevision: authorityRevision,
		ModelRevision: sourceindex.ModelRevision(sourceindex.Index{Artifacts: artifacts}), Artifacts: artifacts,
	}
}

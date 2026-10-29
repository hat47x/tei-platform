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
	"time"

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
//
// NotBefore, NotAfter and Revoked are optional and PROVISIONAL (decisions
// record D7). NotBefore/NotAfter are RFC3339 UTC timestamps ending in a capital
// "Z"; a key is usable when NotBefore <= now < NotAfter (NotBefore inclusive,
// NotAfter exclusive). Omitted (or JSON null) fields mean no restriction, and
// an entry carrying none of the three never consults the clock. Limits that
// matter to operators:
//   - The window is judged at READ time against the Host's own clock, not at
//     signing time (the signed payload carries no timestamp), so every Decision
//     from a retired key lapses together and must be re-issued under a new key.
//   - A running Web Target loads the trust file once, so adding Revoked or
//     editing a window reaches it only after a restart or reload. Only the
//     passage of time across NotAfter/NotBefore self-enforces.
//   - Binaries that predate these fields ignore them and keep trusting the key.
//   - A skewed or manipulated clock is not defended against; only a zero,
//     out-of-range or panicking clock fails closed.
type TrustedIssuer struct {
	Issuer       string   `json:"issuer"`
	KeyID        string   `json:"keyId"`
	PublicKey    string   `json:"publicKey"`
	Applications []string `json:"applications"`
	NotBefore    *string  `json:"notBefore,omitempty"`
	NotAfter     *string  `json:"notAfter,omitempty"`
	Revoked      bool     `json:"revoked,omitempty"`
}

// TrustPolicy is the Host's list of issuers whose signed Decisions it accepts.
type TrustPolicy struct {
	Schema  string          `json:"schema"`
	Issuers []TrustedIssuer `json:"issuers"`

	now func() time.Time // not serialized; nil means time.Now
}

// WithClock returns a copy of the policy whose verifiers read the time from
// now (nil means time.Now). Call it before Verifier(). It exists for tests and
// for Hosts that inject a clock; it does not change the trust file.
func (p TrustPolicy) WithClock(now func() time.Time) TrustPolicy {
	p.now = now
	return p
}

// keyWindow is a parsed validity window; a nil bound is unrestricted.
type keyWindow struct {
	notBefore, notAfter *time.Time
}

// parseKeyTime accepts only RFC3339 with a capital "Z" (no offsets).
func parseKeyTime(s string) (time.Time, error) {
	if !strings.HasSuffix(s, "Z") {
		return time.Time{}, errors.New("not UTC")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// parseKeyWindow is the single rule shared by ParseTrustPolicy (and the CLI via
// it) and the verifier, so parse-time and verify-time checks cannot drift.
func parseKeyWindow(i TrustedIssuer) (keyWindow, error) {
	var w keyWindow
	if i.NotBefore != nil {
		t, err := parseKeyTime(*i.NotBefore)
		if err != nil {
			return keyWindow{}, err
		}
		w.notBefore = &t
	}
	if i.NotAfter != nil {
		t, err := parseKeyTime(*i.NotAfter)
		if err != nil {
			return keyWindow{}, err
		}
		w.notAfter = &t
	}
	if w.notBefore != nil && w.notAfter != nil && !w.notAfter.After(*w.notBefore) {
		return keyWindow{}, errors.New("empty window")
	}
	return w, nil
}

func (i TrustedIssuer) restricted() bool {
	return i.Revoked || i.NotBefore != nil || i.NotAfter != nil
}

// readClock calls the clock and fails closed on a panic, a zero time or a year
// outside 1..9999.
func readClock(clock func() time.Time) (now time.Time, err error) {
	defer func() {
		if recover() != nil {
			now, err = time.Time{}, errors.New("HOST_STORE_CLOCK_INVALID")
		}
	}()
	now = clock()
	if now.IsZero() || now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, errors.New("HOST_STORE_CLOCK_INVALID")
	}
	return now, nil
}

// checkKeyState runs only after the signature verified. It never touches the
// clock for an unrestricted entry.
func checkKeyState(i TrustedIssuer, clock func() time.Time) error {
	if !i.restricted() {
		return nil
	}
	if i.Revoked {
		return errors.New("HOST_STORE_KEY_REVOKED")
	}
	w, err := parseKeyWindow(i)
	if err != nil {
		return errors.New("HOST_STORE_KEY_WINDOW_INVALID")
	}
	if clock == nil {
		clock = time.Now
	}
	now, err := readClock(clock)
	if err != nil {
		return err
	}
	if w.notBefore != nil && now.Before(*w.notBefore) {
		return errors.New("HOST_STORE_KEY_NOT_YET_VALID")
	}
	if w.notAfter != nil && !now.Before(*w.notAfter) {
		return errors.New("HOST_STORE_KEY_EXPIRED")
	}
	return nil
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
// no duplicate issuer/key pair, and a well-formed optional validity window.
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
		if _, err := parseKeyWindow(issuer); err != nil {
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
	clock := p.now
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
			if !ed25519.Verify(ed25519.PublicKey(key), payload, signature) {
				return errors.New("HOST_STORE_SIGNATURE_INVALID")
			}
			// State is judged only for a genuine signature, so a forgery on an
			// expired key still reports SIGNATURE_INVALID.
			return checkKeyState(issuer, clock)
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

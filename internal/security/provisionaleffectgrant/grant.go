// Package provisionaleffectgrant is a DEVELOPMENT- AND VERIFICATION-ONLY
// reference issuer for effect Grants (provisional owner decision D8, see
// plan/workstreams/TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md).
//
// It is NOT a production authority. A signed provisional Grant says only that
// the holder of a private key kept outside the repository and the Store signed
// this exact (Application, Capability, Subject, Resource, Epochs, period)
// tuple. It never shows who the human was, never reads the real Authority
// sources, and every result carries the provisional mark so that Evidence built
// from it (effectcommit.ProvenanceProvisionalGrant) cannot be counted for G2.
//
// Design (mirrors internal/sourceacceptance/hoststore):
//   - Ed25519 over a domain-separated canonical JSON payload.
//   - Trust is a separate trust file (its own schema, so an acceptance key is
//     never an effect-grant key). Key revocation and key window reuse the
//     hoststore implementation through TrustPolicy.VerifyDetached.
//   - notBefore and notAfter are mandatory, UTC, and at most MaxValidity apart.
//   - Verification runs on every read; FileSource re-reads the Grant file and
//     the trust file each time, so revocation and expiry apply immediately.
//   - Errors carry fixed codes only: never the payload, signature or file bytes.
package provisionaleffectgrant

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"tei/internal/security/legacyauthorization"
	"tei/internal/sourceacceptance/hoststore"
)

const (
	// Schema is the signed document schema. It contains "provisional" on purpose.
	Schema = "tei.host-effect-grant-provisional/v0"
	// TrustSchema is the trust file schema; it is not the Source Acceptance one.
	TrustSchema = "tei.host-effect-grant-provisional-trust/v0"
	// MaxValidity is the longest allowed notBefore..notAfter span.
	MaxValidity = 24 * time.Hour
	// MinValidity is the shortest allowed span, so a Grant is not a zero-length
	// or clock-skew-sized sliver that could only be an accident.
	MinValidity = time.Minute
	// maxFieldBytes bounds every text field of a Grant.
	maxFieldBytes = 256
	// MarkProvisional is the Provenance mark. It equals
	// effectcommit.ProvenanceProvisionalGrant; effectcommit is not imported here
	// so this reference issuer stays independent of the effect Runner.
	MarkProvisional = "provisional-grant"

	maxDocumentBytes = 64 << 10
)

// Epochs is the wire form of legacyauthorization.AuthorityEpochs.
type Epochs struct {
	Source      string `json:"source"`
	SubjectLink uint64 `json:"subjectLink"`
	Membership  uint64 `json:"membership"`
	Policy      uint64 `json:"policy"`
}

// Grant is the signed document. Provisional is fixed true.
type Grant struct {
	Schema        string `json:"schema"`
	Provisional   bool   `json:"provisional"`
	Issuer        string `json:"issuer"`
	KeyID         string `json:"keyId"`
	ApplicationID string `json:"applicationId"`
	Capability    string `json:"capability"`
	SubjectID     string `json:"subjectId"`
	Resource      string `json:"resource"`
	Epochs        Epochs `json:"epochs"`
	NotBefore     string `json:"notBefore"`
	NotAfter      string `json:"notAfter"`
	Signature     string `json:"signature,omitempty"`
}

// Query is what the Host is about to allow. Every field must match exactly.
type Query struct {
	ApplicationID string
	Capability    string
	SubjectID     string
	Resource      string
}

// Verified is the result of a successful verification.
type Verified struct {
	// Grant is the value an effect Runner consumes.
	Grant legacyauthorization.AuthorizationGrant
	// GrantRef identifies the signed payload (digest only, no secrets).
	GrantRef string
	// Provenance is always MarkProvisional.
	Provenance string
	NotAfter   time.Time
}

type signedPayload struct {
	Domain        string `json:"domain"`
	Provisional   bool   `json:"provisional"`
	Issuer        string `json:"issuer"`
	KeyID         string `json:"keyId"`
	ApplicationID string `json:"applicationId"`
	Capability    string `json:"capability"`
	SubjectID     string `json:"subjectId"`
	Resource      string `json:"resource"`
	Epochs        Epochs `json:"epochs"`
	NotBefore     string `json:"notBefore"`
	NotAfter      string `json:"notAfter"`
}

// SigningPayload returns the exact bytes an issuer signs.
func SigningPayload(g Grant) ([]byte, error) {
	return json.Marshal(signedPayload{
		Domain: Schema, Provisional: g.Provisional, Issuer: g.Issuer, KeyID: g.KeyID,
		ApplicationID: g.ApplicationID, Capability: g.Capability, SubjectID: g.SubjectID,
		Resource: g.Resource, Epochs: g.Epochs, NotBefore: g.NotBefore, NotAfter: g.NotAfter,
	})
}

// plainField accepts a non-empty, bounded, valid UTF-8 text with no control or
// format character (tab, ESC, bidi overrides such as U+202E, zero-width
// characters, ...) and no whitespace other than inner ASCII spaces.
func plainField(s string) bool {
	if s == "" || len(s) > maxFieldBytes || !utf8.ValidString(s) || s != strings.TrimSpace(s) {
		return false
	}
	for _, r := range s {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) ||
			unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r) || (unicode.IsSpace(r) && r != ' ') {
			return false
		}
	}
	return true
}

func parseUTC(s string) (time.Time, error) {
	if !strings.HasSuffix(s, "Z") {
		return time.Time{}, errors.New("not UTC")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// window validates the structure and returns the parsed period.
func (g Grant) window() (time.Time, time.Time, error) {
	if g.Schema != Schema || !g.Provisional ||
		!plainField(g.Issuer) || !plainField(g.KeyID) || !plainField(g.ApplicationID) ||
		!plainField(g.Capability) || !plainField(g.SubjectID) || !plainField(g.Resource) ||
		!plainField(g.Epochs.Source) || g.Epochs.SubjectLink == 0 || g.Epochs.Membership == 0 || g.Epochs.Policy == 0 {
		return time.Time{}, time.Time{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	nb, err1 := parseUTC(g.NotBefore)
	na, err2 := parseUTC(g.NotAfter)
	if err1 != nil || err2 != nil || !na.After(nb) {
		return time.Time{}, time.Time{}, errors.New("PROVISIONAL_GRANT_PERIOD_INVALID")
	}
	if na.Sub(nb) < MinValidity {
		return time.Time{}, time.Time{}, errors.New("PROVISIONAL_GRANT_PERIOD_TOO_SHORT")
	}
	if na.Sub(nb) > MaxValidity {
		return time.Time{}, time.Time{}, errors.New("PROVISIONAL_GRANT_PERIOD_TOO_LONG")
	}
	return nb, na, nil
}

// Sign validates g (Schema and Provisional are set by Sign) and attaches the
// signature. It is used by an issuer, never by the Host.
func Sign(g Grant, key ed25519.PrivateKey) (Grant, error) {
	if len(key) != ed25519.PrivateKeySize {
		return Grant{}, errors.New("PROVISIONAL_GRANT_SIGNING_KEY_INVALID")
	}
	g.Schema, g.Provisional, g.Signature = Schema, true, ""
	if _, _, err := g.window(); err != nil {
		return Grant{}, err
	}
	payload, err := SigningPayload(g)
	if err != nil {
		return Grant{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	g.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, payload))
	return g, nil
}

// Parse decodes a Grant document strictly: size bound, no unknown member, no
// trailing data.
func Parse(raw []byte) (Grant, error) {
	if len(raw) > maxDocumentBytes {
		return Grant{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	if err := checkKeys(raw); err != nil {
		return Grant{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	var g Grant
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		return Grant{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	if _, err := dec.Token(); err != io.EOF {
		return Grant{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	return g, nil
}

// Verify accepts g only when a listed, currently usable issuer key signed
// exactly this document and every bound field equals q at the policy's clock.
// The signature is judged first so that a tampered field reports a signature
// error, not a binding one. now (nil means time.Now) is used for the Grant
// period; the trust file's own key window uses the policy's clock.
func Verify(g Grant, q Query, policy hoststore.TrustPolicy, now func() time.Time) (Verified, error) {
	return verify(g, q, policy, nil, now)
}

// VerifyScoped is Verify plus the per-key trust scope of trust: a key that
// carries capabilities or resourcePatterns cannot sign a Grant outside them
// (PROVISIONAL_GRANT_OUT_OF_TRUST_SCOPE). A key without scope behaves as in
// Verify. The scope is judged after the signature and key state, so a forgery
// still reports a signature error.
func VerifyScoped(g Grant, q Query, trust Trust, now func() time.Time) (Verified, error) {
	return verify(g, q, trust.Policy, &trust, now)
}

func verify(g Grant, q Query, policy hoststore.TrustPolicy, trust *Trust, now func() time.Time) (Verified, error) {
	nb, na, err := g.window()
	if err != nil {
		return Verified{}, err
	}
	signature, err := base64.StdEncoding.DecodeString(g.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return Verified{}, errors.New("PROVISIONAL_GRANT_SIGNATURE_MALFORMED")
	}
	payload, err := SigningPayload(g)
	if err != nil {
		return Verified{}, errors.New("PROVISIONAL_GRANT_INVALID")
	}
	if err := policy.VerifyDetached(g.Issuer, g.KeyID, g.ApplicationID, payload, signature); err != nil {
		return Verified{}, err
	}
	if trust != nil {
		if scope, ok := trust.ScopeFor(g.Issuer, g.KeyID); ok && !scope.Allows(g.Capability, g.Resource) {
			return Verified{}, errors.New(errOutOfTrustScope)
		}
	}
	if now == nil {
		now = time.Now
	}
	t, err := readClock(now)
	if err != nil {
		return Verified{}, err
	}
	if t.Before(nb) {
		return Verified{}, errors.New("PROVISIONAL_GRANT_NOT_YET_VALID")
	}
	if !t.Before(na) {
		return Verified{}, errors.New("PROVISIONAL_GRANT_EXPIRED")
	}
	if g.ApplicationID != q.ApplicationID || g.Capability != q.Capability ||
		g.SubjectID != q.SubjectID || g.Resource != q.Resource {
		return Verified{}, errors.New("PROVISIONAL_GRANT_BINDING_MISMATCH")
	}
	digest := sha256.Sum256(payload)
	return Verified{
		Grant: legacyauthorization.AuthorizationGrant{
			SubjectID: g.SubjectID, Action: g.Capability, Resource: g.Resource,
			Epochs: &legacyauthorization.AuthorityEpochs{
				Source: g.Epochs.Source, SubjectLink: g.Epochs.SubjectLink,
				Membership: g.Epochs.Membership, Policy: g.Epochs.Policy,
			},
		},
		GrantRef:   MarkProvisional + ":sha256:" + hex.EncodeToString(digest[:]),
		Provenance: MarkProvisional,
		NotAfter:   na,
	}, nil
}

var (
	grantKeys = map[string]bool{
		"schema": true, "provisional": true, "issuer": true, "keyId": true, "applicationId": true,
		"capability": true, "subjectId": true, "resource": true, "epochs": true,
		"notBefore": true, "notAfter": true, "signature": true,
	}
	epochKeys = map[string]bool{"source": true, "subjectLink": true, "membership": true, "policy": true}
)

// checkKeys refuses duplicate keys and any key whose spelling differs from the
// schema (encoding/json would otherwise match "SubjectID" to "subjectId" and
// silently keep the last of two duplicates). Values are validated by Decode.
func checkKeys(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var object func(allowed map[string]bool, top bool) error
	object = func(allowed map[string]bool, top bool) error {
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return errors.New("not an object")
		}
		seen := map[string]bool{}
		for dec.More() {
			tok, err := dec.Token()
			key, ok := tok.(string)
			if err != nil || !ok || !allowed[key] || seen[key] {
				return errors.New("bad key")
			}
			seen[key] = true
			if top && key == "epochs" {
				if err := object(epochKeys, false); err != nil {
					return err
				}
				continue
			}
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		return err
	}
	return object(grantKeys, true)
}

func readClock(clock func() time.Time) (now time.Time, err error) {
	defer func() {
		if recover() != nil {
			now, err = time.Time{}, errors.New("PROVISIONAL_GRANT_CLOCK_INVALID")
		}
	}()
	now = clock().UTC()
	if now.IsZero() || now.Year() < 1 || now.Year() > 9999 {
		return time.Time{}, errors.New("PROVISIONAL_GRANT_CLOCK_INVALID")
	}
	return now, nil
}

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

func plainField(s string) bool {
	return strings.TrimSpace(s) != "" && s == strings.TrimSpace(s) && !strings.ContainsAny(s, "\x00\r\n")
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

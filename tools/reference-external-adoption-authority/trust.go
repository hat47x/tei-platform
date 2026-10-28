package externaladoptionauthority

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
)

const (
	// OwnerAuthorityClass is the authority class the extension catalog
	// verifies namespace authority statements against.
	OwnerAuthorityClass = "semantic-namespace-owner"
	// SignerAuthorityClass is the class this sample uses for distribution
	// signers, judged by the same authoritytrust.Verify as the owner.
	SignerAuthorityClass = "semantic-publication-signer"
)

// BaseTime is the logical origin of the run. Trust points and publication
// expiry are expressed against it; the wall clock is never read, because
// the WSL2 clock can jump.
var BaseTime = time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)

// PublicationValidUntil and HostNotAfter are the signed expiry and the Host
// ceiling of the sample's publication.
var (
	PublicationValidUntil = BaseTime.Add(24 * time.Hour)
	HostNotAfter          = BaseTime.Add(48 * time.Hour)
)

// SignerKey is one generation of a signer's key. Keys are derived from a
// fixed seed so every run signs the same bytes.
type SignerKey struct {
	Ref     string
	Version string
	Epoch   uint64
	Public  ed25519.PublicKey
	private ed25519.PrivateKey
}

// NewSignerKey derives a deterministic key for ref@version.
func NewSignerKey(ref, version string, epoch uint64) SignerKey {
	seed := sha256.Sum256([]byte("tei-eaa-signer:" + ref + "@" + version))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return SignerKey{Ref: ref, Version: version, Epoch: epoch,
		Public: priv.Public().(ed25519.PublicKey), private: priv}
}

// Fingerprint identifies the public key.
func (k SignerKey) Fingerprint() string { return digestBytes(k.Public) }

// TrustPoint is the Host's trust state at one logical point: the current
// owner grants, signer grants and signer keys. A judgment names the point it
// used; nothing reads an earlier point's result as the current answer.
type TrustPoint struct {
	Seq          int                    `json:"seq"`
	Label        string                 `json:"label"`
	Now          time.Time              `json:"now"`
	OwnerGrants  []authoritytrust.Grant `json:"ownerGrants"`
	SignerGrants []authoritytrust.Grant `json:"signerGrants"`
	keys         map[string]SignerKey
}

// TrustHistory is append-only.
type TrustHistory struct {
	Points []TrustPoint `json:"points"`
}

// Add appends a point and returns it with its sequence number.
func (h *TrustHistory) Add(p TrustPoint) TrustPoint {
	p.Seq = len(h.Points) + 1
	if p.Now.IsZero() {
		p.Now = BaseTime.Add(time.Duration(p.Seq) * time.Hour)
	}
	h.Points = append(h.Points, p)
	return p
}

// Current is the last point.
func (h *TrustHistory) Current() TrustPoint { return h.Points[len(h.Points)-1] }

// Key returns the signer key the point holds for ref.
func (p TrustPoint) Key(ref string) (SignerKey, bool) {
	k, ok := p.keys[ref]
	return k, ok
}

// WithKeys returns a copy of the point holding the given keys.
func (p TrustPoint) WithKeys(keys ...SignerKey) TrustPoint {
	p.keys = map[string]SignerKey{}
	for _, k := range keys {
		p.keys[k.Ref] = k
	}
	return p
}

// OwnerScope is the catalog's trust scope for a namespace.
func OwnerScope(namespace string) string { return "namespace:" + namespace }

// NewGrant wraps authoritytrust.NewGrant with a policy fingerprint derived
// from a label, so the sample never writes a fingerprint by hand.
func NewGrant(class, scope string, issuer authoritytrust.Issuer, policy string, active bool) (authoritytrust.Grant, error) {
	return authoritytrust.NewGrant(class, scope, issuer, digestBytes([]byte(policy)), active)
}

// SignerStatement is what the sample asks authoritytrust.Verify about a
// signer: the same function the catalog uses for a namespace owner.
func SignerStatement(namespace string, key SignerKey) authoritytrust.Statement {
	return authoritytrust.Statement{
		AuthorityClass:       SignerAuthorityClass,
		Scope:                OwnerScope(namespace),
		Issuer:               authoritytrust.Issuer{ID: key.Ref, Version: key.Version},
		StatementFingerprint: key.Fingerprint(),
	}
}

// SignerLookup is the Host lookup handed to InspectPublished. It answers
// from this point only: the key the point holds for the signer reference,
// and only if authoritytrust.Verify finds exactly one active grant for that
// key generation.
func (p TrustPoint) SignerLookup() semanticcompositionpreflight.PublisherTrustLookup {
	return func(s semanticcompositionpreflight.PublisherSubject) (semanticcompositionpreflight.PublisherTrust, error) {
		key, ok := p.keys[s.SignerRef]
		if !ok {
			return semanticcompositionpreflight.PublisherTrust{}, fmt.Errorf("no current key for signer %s", s.SignerRef)
		}
		if _, err := authoritytrust.Verify(SignerStatement(s.NamespaceID, key), p.SignerGrants); err != nil {
			return semanticcompositionpreflight.PublisherTrust{}, err
		}
		return semanticcompositionpreflight.PublisherTrust{
			PublicKey: append(ed25519.PublicKey(nil), key.Public...),
			NotAfter:  HostNotAfter,
			Epoch:     key.Epoch,
		}, nil
	}
}

// writeOwnerGrants writes one file per grant, as the catalog CLI reads them.
func writeOwnerGrants(dir string, grants []authoritytrust.Grant) ([]string, error) {
	var paths []string
	for i, g := range grants {
		raw, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, fmt.Sprintf("owner-grant-%d.json", i+1))
		if err := writeFile(path, append(raw, '\n')); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func issuerOf(k SignerKey) authoritytrust.Issuer {
	return authoritytrust.Issuer{ID: k.Ref, Version: k.Version}
}

func ownerList(grants ...authoritytrust.Grant) []authoritytrust.Grant {
	return append([]authoritytrust.Grant(nil), grants...)
}

package semanticcompositionpreflight

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tei/internal/composition"
	"tei/internal/semantic"
)

// PublicationProof is untrusted, signed publisher data. It carries no public
// key and cannot establish that a signer is trusted or that a Host grants
// permission to fetch, accept, or execute a semantic asset.
type PublicationProof struct {
	PublisherID string
	SignerRef   string
	ValidUntil  time.Time
	Signature   []byte
}

// PublisherSubject is the exact trusted lookup subject for one already
// explicitly selected retrieval service. The Host resolves it using its own
// policy; a distribution envelope cannot supply the trust policy or key.
type PublisherSubject struct {
	Scope          string
	ProviderID     string
	Contract       composition.Contract
	PublisherID    string
	SignerRef      string
	PackageID      string
	PackageVersion string
	NamespaceID    string
}

// PublisherTrust is a Host-controlled, point-in-time public-key allowance.
// NotAfter is a Host ceiling separate from the publisher's signed ValidUntil.
// The Host must reject revoked keys itself on every use; this record is not
// a durable permission or trusted Builder provenance.
type PublisherTrust struct {
	PublicKey ed25519.PublicKey
	NotAfter  time.Time
	Epoch     uint64
}

// PublisherTrustLookup must come from the Host policy, not from the signed
// payload, a plugin, or a caller-controlled manifest.
type PublisherTrustLookup func(PublisherSubject) (PublisherTrust, error)

// PublishedSelection combines an explicit service/build selection with an
// untrusted detached proof. No new Artifact Lock or provider resolver exists.
type PublishedSelection struct {
	Asset SelectedAsset
	Proof PublicationProof
}

// ErrPublisherTrustChanged reports that the Host's trusted signer binding
// changed while inspecting the selected semantic assets.
var ErrPublisherTrustChanged = errors.New("Host publisher trust changed during inspection")

const semanticPublicationDomain = "tei-reference-semantic-publication-ed25519-v0"

// SemanticPublicationMessage returns deterministic, domain-separated bytes
// covering the selected service, package, namespace, exact artifact and
// semantic content, build-plan fingerprint, output kind, signer and expiry.
// The publisher signs these bytes; they contain no public-key material.
//
// This *reference* signature format must not be mistaken for the production
// Plugin signing/Build Attestation format or the Linux process image signature.
func SemanticPublicationMessage(scope string, asset SelectedAsset, proof PublicationProof) ([]byte, error) {
	if scope == "" || asset.ProviderID == "" ||
		asset.Contract.ID == "" || asset.Contract.Version == "" || asset.Contract.Shape == "" ||
		proof.PublisherID == "" || proof.SignerRef == "" ||
		asset.Expected.Package.PackageID == "" || asset.Expected.Package.Version == "" ||
		asset.Expected.NamespaceID == "" || asset.Expected.ArtifactDigest == "" ||
		asset.Expected.SemanticDigest == "" || asset.Build.PlanFingerprint == "" ||
		proof.ValidUntil.IsZero() {
		return nil, errors.New("semantic publication requires complete exact identities and expiry")
	}
	return json.Marshal(struct {
		Domain          string `json:"domain"`
		Scope           string `json:"scope"`
		ProviderID      string `json:"providerId"`
		ContractID      string `json:"contractId"`
		ContractVersion string `json:"contractVersion"`
		ContractShape   string `json:"contractShape"`
		PublisherID     string `json:"publisherId"`
		SignerRef       string `json:"signerRef"`
		PackageID       string `json:"packageId"`
		PackageVersion  string `json:"packageVersion"`
		NamespaceID     string `json:"namespaceId"`
		ArtifactDigest  string `json:"artifactDigest"`
		SemanticDigest  string `json:"semanticDigest"`
		PlanFingerprint string `json:"planFingerprint"`
		OutputKind      string `json:"outputKind"`
		ValidUntil      string `json:"validUntil"`
	}{
		Domain:          semanticPublicationDomain,
		Scope:           scope,
		ProviderID:      asset.ProviderID,
		ContractID:      asset.Contract.ID,
		ContractVersion: asset.Contract.Version,
		ContractShape:   asset.Contract.Shape,
		PublisherID:     proof.PublisherID,
		SignerRef:       proof.SignerRef,
		PackageID:       asset.Expected.Package.PackageID,
		PackageVersion:  asset.Expected.Package.Version,
		NamespaceID:     asset.Expected.NamespaceID,
		ArtifactDigest:  asset.Expected.ArtifactDigest,
		SemanticDigest:  asset.Expected.SemanticDigest,
		PlanFingerprint: asset.Build.PlanFingerprint,
		OutputKind:      string(asset.Build.OutputKind),
		ValidUntil:      proof.ValidUntil.UTC().Format(time.RFC3339Nano),
	})
}

// InspectPublished verifies every publication against a Host-selected key and
// Host expiry ceiling before any asset fetch, then reuses the existing
// Composition/Build Lock/CAS/Semantic read-only preflight.
//
// This is a design-time point-in-time observation. The Host MUST independently
// check current signer trust, artifact provenance, runtime permissions, and
// business meaning acceptance when the result is used. Signing an artifact
// binding cannot confer those authorities.
func InspectPublished(
	registry *composition.Registry,
	scope string,
	local []semantic.Namespace,
	publications []PublishedSelection,
	trusted PublisherTrustLookup,
	now time.Time,
) (Report, error) {
	if trusted == nil || now.IsZero() || len(publications) == 0 {
		return Report{}, errors.New("Host signer lookup, current time and explicit publications are required")
	}
	assets := make([]SelectedAsset, 0, len(publications))
	verified := make([]struct {
		subject PublisherSubject
		trust   PublisherTrust
	}, 0, len(publications))
	for _, selection := range publications {
		asset, proof := selection.Asset, selection.Proof
		if len(proof.Signature) != ed25519.SignatureSize {
			return Report{}, fmt.Errorf("namespace %s: missing or malformed publisher signature", asset.Expected.NamespaceID)
		}
		if proof.ValidUntil.IsZero() || !now.Before(proof.ValidUntil) {
			return Report{}, fmt.Errorf("namespace %s: expired publisher binding", asset.Expected.NamespaceID)
		}
		msg, err := SemanticPublicationMessage(scope, asset, proof)
		if err != nil {
			return Report{}, err
		}
		subject := PublisherSubject{
			Scope: scope, ProviderID: asset.ProviderID, Contract: asset.Contract,
			PublisherID: proof.PublisherID, SignerRef: proof.SignerRef,
			PackageID:      asset.Expected.Package.PackageID,
			PackageVersion: asset.Expected.Package.Version,
			NamespaceID:    asset.Expected.NamespaceID,
		}
		allowance, err := trusted(subject)
		if err != nil {
			return Report{}, fmt.Errorf("namespace %s: Host has not approved publication signer: %w", asset.Expected.NamespaceID, err)
		}
		if len(allowance.PublicKey) != ed25519.PublicKeySize || allowance.Epoch == 0 ||
			allowance.NotAfter.IsZero() || !now.Before(allowance.NotAfter) ||
			proof.ValidUntil.After(allowance.NotAfter) {
			return Report{}, fmt.Errorf("namespace %s: publisher signature or Host allowance rejected", asset.Expected.NamespaceID)
		}
		// Snapshot the Host-provided key before verifying it and before the
		// later epoch comparison; untrusted callers cannot replace this copy.
		allowance.PublicKey = append(ed25519.PublicKey(nil), allowance.PublicKey...)
		if !ed25519.Verify(allowance.PublicKey, msg, proof.Signature) {
			return Report{}, fmt.Errorf("namespace %s: publisher signature or Host allowance rejected", asset.Expected.NamespaceID)
		}
		assets = append(assets, asset)
		verified = append(verified, struct {
			subject PublisherSubject
			trust   PublisherTrust
		}{subject: subject, trust: allowance})
	}
	report, err := Inspect(registry, scope, local, assets)
	if err != nil {
		return report, err
	}
	// The Host must check trust at actual use time as well. This final
	// snapshot check merely detects changes during this one inspection.
	for _, item := range verified {
		current, err := trusted(item.subject)
		if err != nil || current.Epoch != item.trust.Epoch ||
			!bytes.Equal(current.PublicKey, item.trust.PublicKey) ||
			!current.NotAfter.Equal(item.trust.NotAfter) {
			return report, ErrPublisherTrustChanged
		}
	}
	return report, nil
}

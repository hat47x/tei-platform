package semanticcompositionpreflight

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

var publicationTime = time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

func signedPublications(t *testing.T) (
	[]PublishedSelection, PublisherTrustLookup, []byte,
) {
	t.Helper()
	_, _, assets := selectedAssetsFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	approved := map[PublisherSubject]PublisherTrust{}
	selections := make([]PublishedSelection, 0, len(assets))
	for _, asset := range assets {
		proof := PublicationProof{
			PublisherID: "community.publisher",
			SignerRef: "host-approved.semantic-publisher",
			ValidUntil: publicationTime.Add(24 * time.Hour),
		}
		message, err := SemanticPublicationMessage("tenant-a", asset, proof)
		if err != nil {
			t.Fatal(err)
		}
		proof.Signature = ed25519.Sign(private, message)
		selections = append(selections, PublishedSelection{Asset: asset, Proof: proof})
		subject := PublisherSubject{
			Scope: "tenant-a", ProviderID: asset.ProviderID, Contract: asset.Contract,
			PublisherID: proof.PublisherID, SignerRef: proof.SignerRef,
		}
		approved[subject] = PublisherTrust{
			PublicKey: append(ed25519.PublicKey(nil), public...),
			NotAfter: publicationTime.Add(48 * time.Hour),
		}
	}
	lookup := func(subject PublisherSubject) (PublisherTrust, error) {
		trust, exists := approved[subject]
		if !exists {
			return PublisherTrust{}, errors.New("Host publication allowance not found")
		}
		return trust, nil
	}
	return selections, lookup, append([]byte(nil), public...)
}

func TestSignedPublicationsVerifyExactBindingsBeforeSemanticInspection(t *testing.T) {
	registry, local, _ := selectedAssetsFixture(t)
	selections, lookup, _ := signedPublications(t)
	report, err := InspectPublished(registry, "tenant-a", local, selections, lookup, publicationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.References.Dependencies) != 2 ||
		len(report.References.Unmatched) != 0 || len(report.Diagnostics) != 0 {
		t.Fatalf("trusted signature check did not reach canonical semantic validation: %+v", report)
	}
	reordered := []PublishedSelection{selections[1], selections[0]}
	again, err := InspectPublished(registry, "tenant-a", local, reordered, lookup, publicationTime)
	if err != nil || !reflect.DeepEqual(again.References, report.References) {
		t.Fatalf("explicit publication order changed semantic view: %+v %v", again, err)
	}
}

func TestPublicationProofTamperingFailsBeforeAnyAssetFetch(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*[]PublishedSelection, *string, *time.Time)
	}{
		{"wrong-artifact", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Expected.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
		}},
		{"wrong-semantic-content", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Expected.SemanticDigest = "sha256:" + strings.Repeat("f", 64)
		}},
		{"wrong-build-plan", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Build.PlanFingerprint = strings.Repeat("a", 64)
		}},
		{"wrong-package", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Expected.Package.Version = "9.9.9"
		}},
		{"wrong-namespace", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Expected.NamespaceID = "other"
		}},
		{"wrong-service", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.ProviderID = "read-os"
		}},
		{"wrong-contract", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Asset.Contract.Shape = "forged"
		}},
		{"wrong-scope", func(_ *[]PublishedSelection, scope *string, _ *time.Time) {
			*scope = "tenant-b"
		}},
		{"wrong-publisher", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Proof.PublisherID = "other.publisher"
		}},
		{"wrong-signer-reference", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Proof.SignerRef = "unapproved"
		}},
		{"expiry-tampered", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Proof.ValidUntil = publicationTime.Add(36 * time.Hour)
		}},
		{"expired", func(_ *[]PublishedSelection, _ *string, now *time.Time) {
			*now = publicationTime.Add(24 * time.Hour)
		}},
		{"malformed-signature", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Proof.Signature = []byte("unsigned")
		}},
		{"bit-flipped-signature", func(items *[]PublishedSelection, _ *string, _ *time.Time) {
			(*items)[0].Proof.Signature = append([]byte(nil), (*items)[0].Proof.Signature...)
			(*items)[0].Proof.Signature[0] ^= 0xff
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			registry, local, _ := selectedAssetsFixture(t)
			selection, lookup, _ := signedPublications(t)
			scope, now := "tenant-a", publicationTime
			tc.mutate(&selection, &scope, &now)
			calls := 0
			for i := range selection {
				selection[i].Asset.Fetch = func(string) ([]byte, error) {
					calls++
					return nil, errors.New("signature should fail before fetching")
				}
			}
			if _, err := InspectPublished(registry, scope, local, selection, lookup, now); err == nil {
				t.Fatal("tampered, untrusted or expired publication succeeded")
			}
			if calls != 0 {
				t.Fatalf("invalid publication fetched %d asset(s)", calls)
			}
		})
	}
}

func TestHostTrustAndExpiryControlPublicationIndependentOfSignedProof(t *testing.T) {
	cases := []struct {
		name   string
		trust  func(PublisherTrustLookup) PublisherTrustLookup
	}{
		{"host-key-revoked", func(_ PublisherTrustLookup) PublisherTrustLookup {
			return func(PublisherSubject) (PublisherTrust, error) {
				return PublisherTrust{}, errors.New("revoked")
			}
		}},
		{"host-rotated-key", func(base PublisherTrustLookup) PublisherTrustLookup {
			return func(s PublisherSubject) (PublisherTrust, error) {
				p, err := base(s)
				if err == nil {
					p.PublicKey = make(ed25519.PublicKey, ed25519.PublicKeySize)
				}
				return p, err
			}
		}},
		{"host-policy-expired", func(base PublisherTrustLookup) PublisherTrustLookup {
			return func(s PublisherSubject) (PublisherTrust, error) {
				p, err := base(s)
				p.NotAfter = publicationTime
				return p, err
			}
		}},
		{"signed-expiry-exceeds-host-ceiling", func(base PublisherTrustLookup) PublisherTrustLookup {
			return func(s PublisherSubject) (PublisherTrust, error) {
				p, err := base(s)
				p.NotAfter = publicationTime.Add(time.Hour)
				return p, err
			}
		}},
		{"missing-public-key", func(base PublisherTrustLookup) PublisherTrustLookup {
			return func(s PublisherSubject) (PublisherTrust, error) {
				p, err := base(s)
				p.PublicKey = nil
				return p, err
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			registry, local, _ := selectedAssetsFixture(t)
			selections, allow, _ := signedPublications(t)
			calls := 0
			for i := range selections {
				selections[i].Asset.Fetch = func(string) ([]byte, error) {
					calls++
					return nil, errors.New("must not fetch")
				}
			}
			if _, err := InspectPublished(registry, "tenant-a", local, selections, tc.trust(allow), publicationTime); err == nil {
				t.Fatal("Host signer disallowance was ignored")
			}
			if calls != 0 {
				t.Fatalf("disallowed publisher fetched %d assets", calls)
			}
		})
	}
}

func TestPublicationRequiresHostLookupAndNow(t *testing.T) {
	r, local, _ := selectedAssetsFixture(t)
	selections, lookup, _ := signedPublications(t)
	if _, err := InspectPublished(r, "tenant-a", local, selections, nil, publicationTime); err == nil {
		t.Fatal("untrusted publisher self-supplied key was accepted")
	}
	if _, err := InspectPublished(r, "tenant-a", local, selections, lookup, time.Time{}); err == nil {
		t.Fatal("missing trusted Host time was accepted")
	}
	if _, err := InspectPublished(r, "tenant-a", local, nil, lookup, publicationTime); err == nil {
		t.Fatal("no explicit assets should not be treated as complete")
	}
}

func TestPublicationSignatureDomainIsNotAnExecutionImageSignature(t *testing.T) {
	selections, _, _ := signedPublications(t)
	proof := selections[0].Proof
	msg, err := SemanticPublicationMessage("tenant-a", selections[0].Asset, proof)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(msg), "tei-reference-semantic-publication-ed25519-v0") ||
		strings.Contains(string(msg), "tei-linux-pinned-image-ed25519-v1") {
		t.Fatalf("publication proof reused executable image authority: %s", msg)
	}
}

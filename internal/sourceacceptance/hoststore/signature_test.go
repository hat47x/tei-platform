package hoststore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
)

type issuerKey struct {
	issuer, keyID string
	public        ed25519.PublicKey
	private       ed25519.PrivateKey
}

func newIssuer(t *testing.T, issuer, keyID string) issuerKey {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return issuerKey{issuer: issuer, keyID: keyID, public: public, private: private}
}

func trustOf(t *testing.T, apps []string, keys ...issuerKey) TrustPolicy {
	t.Helper()
	doc := TrustPolicy{Schema: trustSchema}
	for _, k := range keys {
		doc.Issuers = append(doc.Issuers, TrustedIssuer{
			Issuer: k.issuer, KeyID: k.keyID, PublicKey: base64.StdEncoding.EncodeToString(k.public), Applications: apps,
		})
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ParseTrustPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func signed(t *testing.T, k issuerKey, ref string, content string) Decision {
	t.Helper()
	d := decision(ref, members(t, content))
	d.Issuer = k.issuer
	out, err := Sign(d, k.keyID, k.private)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSignedDecisionFromATrustedIssuerBecomesCurrent(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	store := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := store.Ingest(context.Background(), signed(t, alice, "d1", "application order_entry {}\n")); err != nil {
		t.Fatal(err)
	}
	if err := verify(store, members(t, "application order_entry {}\n")); err != nil {
		t.Fatalf("signed R1 rejected: %v", err)
	}
}

func TestUnsignedUntrustedOrTamperedDecisionsAreRefused(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	mallory := newIssuer(t, "issuer-m", "k9")
	policy := trustOf(t, []string{"order_entry"}, alice)
	verifier := policy.Verifier()
	good := signed(t, alice, "d1", "application order_entry {}\n")

	cases := map[string]Decision{
		"unsigned":         decision("d1", members(t, "application order_entry {}\n")),
		"untrusted issuer": signed(t, mallory, "d1", "application order_entry {}\n"),
	}
	impostor := signed(t, mallory, "d1", "application order_entry {}\n")
	impostor.Issuer = alice.issuer // claims Alice, signed by Mallory
	cases["impostor"] = impostor
	tampered := good
	tampered.Record = cloneRecord(good.Record)
	tampered.Record.AuthorityRevision = "host-forged"
	cases["tampered record"] = tampered
	wrongKeyID := good
	var ev signatureEvidence
	_ = json.Unmarshal(good.Evidence, &ev)
	ev.KeyID = "k2"
	wrongKeyID.Evidence, _ = json.Marshal(ev)
	cases["unknown key id"] = wrongKeyID
	otherApp := newIssuer(t, "issuer-r", "k1")
	cases["issuer not trusted for application"] = signed(t, otherApp, "d1", "application order_entry {}\n")

	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			if err := verifier(context.Background(), d); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	scoped := trustOf(t, []string{"repair_intake"}, otherApp)
	if err := scoped.Verifier()(context.Background(), cases["issuer not trusted for application"]); err == nil {
		t.Fatal("issuer trusted only for repair_intake signed order_entry")
	}
	if err := verifier(context.Background(), good); err != nil {
		t.Fatalf("control: %v", err)
	}
}

func TestDecisionPlacedInTheDirectoryWithoutIngestIsNotCurrent(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	mallory := newIssuer(t, "issuer-m", "k9")
	store := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := store.Ingest(context.Background(), signed(t, alice, "d1", "application order_entry {}\n")); err != nil {
		t.Fatal(err)
	}
	// Someone who can write the Store directory replaces the current entry
	// with a Decision signed by an untrusted key for different Source.
	raw, err := os.ReadFile(store.path("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	var state file
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	forged := signed(t, mallory, "d1", "application order_entry { forged }\n")
	forged.Issuer = alice.issuer
	state.Entries[0].Decision = forged
	if err := store.save(state); err != nil {
		t.Fatal(err)
	}
	err = verify(store, members(t, "application order_entry { forged }\n"))
	if !errors.Is(err, sourceacceptance.ErrCurrentAcceptanceUnavailable) {
		t.Fatalf("a Decision written outside Ingest was treated as current: %v", err)
	}
}

func TestStoreWithoutVerifierReportsNoCurrentDecision(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	trusted := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := trusted.Ingest(context.Background(), signed(t, alice, "d1", "x")); err != nil {
		t.Fatal(err)
	}
	blind, err := Open(trusted.dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blind.ReadCurrentAcceptedSource(context.Background(), "order_entry"); !errors.Is(err, ErrVerifierRequired) {
		t.Fatalf("no verifier must mean no authority: %v", err)
	}
}

func TestTrustPolicyValidation(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	for name, raw := range map[string]string{
		"not json":       `{`,
		"wrong schema":   `{"schema":"x","issuers":[{"issuer":"a","keyId":"k","publicKey":"` + key + `","applications":["order_entry"]}]}`,
		"no issuers":     `{"schema":"` + trustSchema + `","issuers":[]}`,
		"short key":      `{"schema":"` + trustSchema + `","issuers":[{"issuer":"a","keyId":"k","publicKey":"AAAA","applications":["order_entry"]}]}`,
		"no keyId":       `{"schema":"` + trustSchema + `","issuers":[{"issuer":"a","keyId":"","publicKey":"` + key + `","applications":["order_entry"]}]}`,
		"no application": `{"schema":"` + trustSchema + `","issuers":[{"issuer":"a","keyId":"k","publicKey":"` + key + `","applications":[]}]}`,
		"duplicate":      `{"schema":"` + trustSchema + `","issuers":[{"issuer":"a","keyId":"k","publicKey":"` + key + `","applications":["x"]},{"issuer":"a","keyId":"k","publicKey":"` + key + `","applications":["y"]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseTrustPolicy([]byte(raw))
			if err == nil || strings.Contains(err.Error(), key) {
				t.Fatalf("accepted or leaked: %v", err)
			}
		})
	}
}

func TestSigningPayloadIsOrderIndependent(t *testing.T) {
	d := decision("d1", members(t, "application order_entry {}\n"))
	reversed := d
	reversed.Record = cloneRecord(d.Record)
	reversed.Record.Artifacts[0], reversed.Record.Artifacts[1] = reversed.Record.Artifacts[1], reversed.Record.Artifacts[0]
	a, err := SigningPayload(d)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SigningPayload(reversed)
	if err != nil || string(a) != string(b) {
		t.Fatal("member order changed the signed bytes")
	}
}

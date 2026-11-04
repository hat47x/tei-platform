package hoststore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
)

// D9: the acceptance kind is part of the signed bytes.

func provisionalSigned(t *testing.T, k issuerKey, ref string) Decision {
	t.Helper()
	d := decision(ref, members(t, "application order_entry {}\n"))
	d.Issuer = k.issuer
	d.AcceptanceKind = sourceacceptance.AcceptanceKindProvisional
	out, err := Sign(d, k.keyID, k.private)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProvisionalKindIsReadBackAndReportedByVerifyCurrent(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	store := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := store.Ingest(context.Background(), provisionalSigned(t, alice, "d1")); err != nil {
		t.Fatal(err)
	}
	index := members(t, "application order_entry {}\n")
	record, err := sourceacceptance.VerifyCurrent(context.Background(), store, "order_entry", index)
	if err != nil || record.AcceptanceKind != sourceacceptance.AcceptanceKindProvisional {
		t.Fatalf("kind=%q err=%v", record.AcceptanceKind, err)
	}
}

func TestLegacyDecisionStaysReadableAsUnspecifiedNeverFormal(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	store := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := store.Ingest(context.Background(), signed(t, alice, "d1", "application order_entry {}\n")); err != nil {
		t.Fatal(err)
	}
	record, err := sourceacceptance.VerifyCurrent(context.Background(), store, "order_entry", members(t, "application order_entry {}\n"))
	if err != nil || record.AcceptanceKind != sourceacceptance.AcceptanceKindUnspecified {
		t.Fatalf("kind=%q err=%v", record.AcceptanceKind, err)
	}
}

// A v0 payload is byte-for-byte what it was before D9 (golden shape).
func TestLegacySigningPayloadIsUnchanged(t *testing.T) {
	d := decision("d1", members(t, "application order_entry {}\n"))
	d.Issuer = "iss"
	raw, err := SigningPayload(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), `{"schema":"tei.host-acceptance-decision/v0","issuer":"iss","record":{`) || strings.Contains(string(raw), "acceptanceKind") {
		t.Fatalf("legacy payload changed: %s", raw)
	}
	d.AcceptanceKind = "provisional"
	raw, _ = SigningPayload(d)
	if !strings.HasPrefix(string(raw), `{"schema":"tei.host-acceptance-decision/v1","issuer":"iss","acceptanceKind":"provisional","record":{`) {
		t.Fatalf("v1 payload: %s", raw)
	}
}

func TestAcceptanceKindTamperingBreaksTheSignature(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	verifier := trustOf(t, []string{"order_entry"}, alice).Verifier()
	ctx := context.Background()
	prov := provisionalSigned(t, alice, "d1")
	legacy := signed(t, alice, "d2", "application order_entry {}\n")

	removed := prov
	removed.AcceptanceKind = "" // downgrade to "unspecified" / strip the marker
	added := legacy
	added.AcceptanceKind = "provisional" // add a marker to an old signature
	formalFromProv := prov
	formalFromProv.AcceptanceKind = "formal"
	formalFromLegacy := legacy
	formalFromLegacy.AcceptanceKind = "formal"
	for name, d := range map[string]Decision{
		"removed": removed, "added": added, "formal-from-provisional": formalFromProv, "formal-from-legacy": formalFromLegacy,
		"case": func() Decision { x := prov; x.AcceptanceKind = "Provisional"; return x }(),
	} {
		if err := verifier(ctx, d); err == nil {
			t.Errorf("%s: verifier accepted a tampered kind", name)
		}
	}
	if err := verifier(ctx, prov); err != nil {
		t.Fatalf("genuine provisional Decision refused: %v", err)
	}
	if err := verifier(ctx, legacy); err != nil {
		t.Fatalf("genuine legacy Decision refused: %v", err)
	}
}

func TestFormalCannotBeSignedIngestedOrRead(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	d := decision("d1", members(t, "application order_entry {}\n"))
	d.Issuer, d.AcceptanceKind = alice.issuer, "formal"
	if _, err := Sign(d, alice.keyID, alice.private); err == nil {
		t.Fatal("Sign issued a formal Decision")
	}
	// Even a verifier that approves everything cannot make "formal" current.
	store := openStore(t, func(context.Context, Decision) error { return nil })
	if err := store.Ingest(context.Background(), d); err == nil {
		t.Fatal("Ingest accepted a formal Decision")
	}
	// A store file edited by hand to say "formal" is not current.
	prov := provisionalSigned(t, alice, "d1")
	store = openStore(t, func(context.Context, Decision) error { return nil })
	if err := store.Ingest(context.Background(), prov); err != nil {
		t.Fatal(err)
	}
	path := store.path("order_entry")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(raw), `"acceptanceKind": "provisional"`, `"acceptanceKind": "formal"`, 1)
	if edited == string(raw) {
		t.Fatal("store file did not contain the marker")
	}
	if err := os.WriteFile(filepath.Clean(path), []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCurrentAcceptedSource(context.Background(), "order_entry"); err == nil {
		t.Fatal("a hand-edited formal Decision became current")
	}
	var probe map[string]any
	_ = json.Unmarshal(raw, &probe)
}

func TestVerifyCurrentRefusesAReaderThatReportsFormal(t *testing.T) {
	alice := newIssuer(t, "issuer-a", "k1")
	store := openStore(t, trustOf(t, []string{"order_entry"}, alice).Verifier())
	if err := store.Ingest(context.Background(), provisionalSigned(t, alice, "d1")); err != nil {
		t.Fatal(err)
	}
	index := members(t, "application order_entry {}\n")
	_, err := sourceacceptance.VerifyCurrent(context.Background(), formalReader{store}, "order_entry", index)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_WITNESS_INVALID") {
		t.Fatalf("formal reader result accepted: %v", err)
	}
}

type formalReader struct{ *Store }

func (f formalReader) ReadCurrentAcceptedSource(ctx context.Context, app string) (sourceacceptance.Record, error) {
	r, err := f.Store.ReadCurrentAcceptedSource(ctx, app)
	r.AcceptanceKind = "formal"
	return r, err
}

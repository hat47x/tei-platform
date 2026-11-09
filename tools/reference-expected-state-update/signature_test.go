package expectedstateupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func signedSampleRecord(t *testing.T) Record {
	t.Helper()
	record, err := RecordRun(
		[]UpdateRequest{{
			Key: "order:1", ExpectedPresent: true, Expected: "old", NewValue: "new",
			IdempotencyKey: "op-1", Authorized: true, Valid: true,
		}},
		[]Result{{
			Schema: Schema, Key: "order:1", Outcome: OutcomeApplied,
			Before: "old", NewValue: "new",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestSignedRecordRoundTrip(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignRecord(signedSampleRecord(t), "audit-key-1", private)
	if err != nil {
		t.Fatal(err)
	}
	if signed.Schema != SignedRecordSchema || signed.Algorithm != SignatureAlgorithm || signed.KeyID != "audit-key-1" {
		t.Fatalf("unexpected envelope: %+v", signed)
	}
	if err := VerifySignedRecord(signed, "audit-key-1", public); err != nil {
		t.Fatal(err)
	}
}

func TestSignedRecordRejectsTamperedEntry(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignRecord(signedSampleRecord(t), "audit-key-1", private)
	if err != nil {
		t.Fatal(err)
	}
	// The attacker changes the outcome and recomputes the record digest. The
	// digest is self-consistent again, but the signature no longer matches.
	signed.Record.Entries[0].Outcome = OutcomeConflict
	digest, err := recordDigest(signed.Record.Entries)
	if err != nil {
		t.Fatal(err)
	}
	signed.Record.Digest = digest
	if err := VerifySignedRecord(signed, "audit-key-1", public); err == nil {
		t.Fatal("a tampered record with a recomputed digest was accepted")
	}
}

func TestSignedRecordRejectsWrongKey(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignRecord(signedSampleRecord(t), "audit-key-1", private)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignedRecord(signed, "audit-key-1", otherPublic); err == nil {
		t.Fatal("a signature verified with an unrelated key")
	}
}

func TestSignedRecordRejectsKeyIDMismatch(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignRecord(signedSampleRecord(t), "audit-key-1", private)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignedRecord(signed, "audit-key-2", public); err == nil {
		t.Fatal("a signature verified under a different key id")
	}
	// Re-labelling the envelope without re-signing must also fail.
	signed.KeyID = "audit-key-2"
	if err := VerifySignedRecord(signed, "audit-key-2", public); err == nil {
		t.Fatal("a re-labelled key id verified")
	}
}

func TestSignedRecordRejectsMissingOrUnsupportedEnvelope(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignRecord(signedSampleRecord(t), "audit-key-1", private)
	if err != nil {
		t.Fatal(err)
	}
	missing := signed
	missing.Signature = ""
	if err := VerifySignedRecord(missing, "audit-key-1", public); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing signature error=%v", err)
	}
	unsupported := signed
	unsupported.Algorithm = "hmac-sha256"
	if err := VerifySignedRecord(unsupported, "audit-key-1", public); err == nil {
		t.Fatal("an unsupported algorithm was accepted")
	}
	badSchema := signed
	badSchema.Schema = "tei.reference.expected-state-signed-record/v999"
	if err := VerifySignedRecord(badSchema, "audit-key-1", public); err == nil {
		t.Fatal("an unknown envelope schema was accepted")
	}
}

func TestSignRecordRequiresKeyID(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SignRecord(signedSampleRecord(t), "", private); err == nil {
		t.Fatal("signing without a key id was accepted")
	}
}

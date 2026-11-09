package expectedstateupdate

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func keys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return public, private
}

func TestKeyRingSignsAndVerifiesWithAnActiveKey(t *testing.T) {
	ring := NewKeyRing()
	public, private := keys(t)
	if err := ring.Add("key-1", public, private); err != nil {
		t.Fatal(err)
	}
	signed, err := ring.Sign(signedSampleRecord(t), "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.Verify(signed); err != nil {
		t.Fatal(err)
	}
}

func TestKeyRingRefusesUnknownAndDuplicateKeys(t *testing.T) {
	ring := NewKeyRing()
	public, private := keys(t)
	if err := ring.Add("key-1", public, private); err != nil {
		t.Fatal(err)
	}
	if err := ring.Add("key-1", public, private); err == nil {
		t.Fatal("a duplicate key id was accepted")
	}
	if err := ring.Add("", public, private); err == nil {
		t.Fatal("an empty key id was accepted")
	}
	if _, err := ring.Sign(signedSampleRecord(t), "unknown"); err == nil {
		t.Fatal("signing with an unknown key was accepted")
	}
	signed, err := SignRecord(signedSampleRecord(t), "unknown", private)
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.Verify(signed); err == nil {
		t.Fatal("verification with an unknown key id was accepted")
	}
}

func TestKeyRingRotationRetiresTheOldKey(t *testing.T) {
	ring := NewKeyRing()
	oldPublic, oldPrivate := keys(t)
	newPublic, newPrivate := keys(t)
	if err := ring.Add("key-1", oldPublic, oldPrivate); err != nil {
		t.Fatal(err)
	}
	before, err := ring.Sign(signedSampleRecord(t), "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.Rotate("key-1", "key-2", newPublic, newPrivate); err != nil {
		t.Fatal(err)
	}
	if _, err := ring.Sign(signedSampleRecord(t), "key-1"); err == nil {
		t.Fatal("a retired key was used for signing")
	}
	if err := ring.Verify(before); err == nil {
		t.Fatal("a retired key verified a record through Verify")
	}
	if err := ring.VerifyArchived(before); err != nil {
		t.Fatalf("VerifyArchived refused a pre-rotation signature: %v", err)
	}
	after, err := ring.Sign(signedSampleRecord(t), "key-2")
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.Verify(after); err != nil {
		t.Fatal(err)
	}
	if err := ring.Rotate("key-1", "key-3", newPublic, newPrivate); err == nil {
		t.Fatal("rotating an already retired key was accepted")
	}
}

func TestKeyRingVerifyArchivedStillRefusesUnknownKeys(t *testing.T) {
	ring := NewKeyRing()
	_, private := keys(t)
	signed, err := SignRecord(signedSampleRecord(t), "unknown", private)
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.VerifyArchived(signed); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("VerifyArchived error=%v", err)
	}
	if err := ring.Retire("unknown"); err == nil {
		t.Fatal("retiring an unknown key was accepted")
	}
}

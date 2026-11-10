package hashid

import (
	"crypto/sha3"
	"encoding/hex"
	"io"
	"testing"
)

// TestNISTCSHAKE256Sample3 validates the underlying cSHAKE256 primitive
// against NIST SP 800-185 published samples, independently of TEI framing.
// X=00010203, L=512 bits, N="", S="Email Signature".
// Reference: https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines/example-values
// (cSHAKE_samples.pdf, sample 3).
func TestNISTCSHAKE256Sample3(t *testing.T) {
	xof := sha3.NewCSHAKE256(nil, []byte("Email Signature"))
	if _, err := xof.Write([]byte{0, 1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	actual := make([]byte, 64)
	if _, err := io.ReadFull(xof, actual); err != nil {
		t.Fatal(err)
	}
	const want = "d008828e2b80ac9d2218ffee1d070c48" +
		"b8e4c87bff32c9699d5b6896eee0edd1" +
		"64020e2be0560858d9c00c037e34a969" +
		"37c561a74c412bb4c746469527281c8c"
	if hex.EncodeToString(actual) != want {
		t.Fatalf("NIST cSHAKE256 sample mismatch: got %x", actual)
	}
}

// cSHAKE with empty N and S is defined to reduce to ordinary SHAKE.
// This verifies that our Go SHA-3 provider follows this standard behavior.
func TestCSHAKE256EmptyCustomizationMatchesSHAKE256(t *testing.T) {
	input := []byte("tei.hashid.cshake256.reference")
	cshake := sha3.NewCSHAKE256(nil, nil)
	shake := sha3.NewSHAKE256()
	if _, err := cshake.Write(input); err != nil {
		t.Fatal(err)
	}
	if _, err := shake.Write(input); err != nil {
		t.Fatal(err)
	}
	a := make([]byte, 64)
	b := make([]byte, 64)
	if _, err := io.ReadFull(cshake, a); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(shake, b); err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(a) != hex.EncodeToString(b) {
		t.Fatal("cSHAKE256 with empty customization must equal SHAKE256")
	}
}

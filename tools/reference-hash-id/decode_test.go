package hashid

import (
	"bytes"
	"strings"
	"testing"
)

func TestDigestEncodingRoundTripAndCanonicality(t *testing.T) {
	s := fixtureScheme()
	input := []byte("hello TEI")
	for _, tc := range []struct {
		name     string
		encoding string
		alphabet string
	}{
		{"hex", EncodingHex, ""},
		{"base32", EncodingBase32, ""},
		{"base64url", EncodingBase64URL, ""},
		{"decimal", EncodingRadix, "0123456789"},
		{"base36", EncodingRadix, "0123456789abcdefghijklmnopqrstuvwxyz"},
		{"base62", EncodingRadix, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.Encoding, s.Alphabet = tc.encoding, tc.alphabet
			derived, err := Derive(s, input)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeDigest(s, derived.Text)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(derived.Digest, decoded) {
				t.Fatal("round-trip changed digest")
			}
		})
	}
}

func TestRadix128BitWidthsAndRejects(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.OutputBits, s.Encoding = AlgorithmSHAKE256, 128, EncodingRadix
	for _, tc := range []struct {
		alphabet string
		width    int
	}{
		{"0123456789", 39},
		{"0123456789abcdefghijklmnopqrstuvwxyz", 25},
		{"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 22},
	} {
		s.Alphabet = tc.alphabet
		for _, digest := range [][]byte{make([]byte, 16), bytes.Repeat([]byte{0xff}, 16)} {
			encoded := encodeRadix(tc.alphabet, digest)
			if len(encoded) != tc.width {
				t.Fatalf("radix=%d got width %d, expected %d", len(tc.alphabet), len(encoded), tc.width)
			}
			decoded, err := DecodeDigest(s, encoded)
			if err != nil || !bytes.Equal(decoded, digest) {
				t.Fatalf("radix=%d: %v", len(tc.alphabet), err)
			}
			if _, err := DecodeDigest(s, encoded[1:]); err == nil {
				t.Fatal("short radix accepted")
			}
		}
		if _, err := DecodeDigest(s, strings.Repeat(string(tc.alphabet[len(tc.alphabet)-1]), tc.width)); err == nil {
			t.Fatal("out of range radix accepted")
		}
	}
}

func TestRejectNoncanonicalDigestForms(t *testing.T) {
	s := fixtureScheme()
	derived, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDigest(s, strings.ToUpper(derived.Text)); err == nil {
		t.Fatal("uppercase hex accepted")
	}
	s.Encoding = EncodingBase32
	derived, err = Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDigest(s, strings.ToUpper(derived.Text)); err == nil {
		t.Fatal("uppercase base32 accepted")
	}
	s.Encoding = EncodingBase64URL
	derived, err = Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDigest(s, derived.Text+"="); err == nil {
		t.Fatal("padded base64url accepted")
	}
	if _, err := DecodeDigest(s, strings.Repeat("A", maxOutputBits+1)); err == nil {
		t.Fatal("oversized untrusted digest text accepted")
	}
	s.Mode, s.Encoding, s.Algorithm, s.OutputBits = ModeExactCharacters, "", AlgorithmSHAKE256, 0
	s.Alphabet, s.Length = "0123456789", 30
	if _, err := DecodeDigest(s, strings.Repeat("0", 30)); err == nil {
		t.Fatal("exactCharacters must not decode")
	}
}

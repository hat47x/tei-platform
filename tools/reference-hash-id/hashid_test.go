package hashid

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"strings"
	"testing"
)

func fixtureScheme() Scheme {
	return Scheme{
		SchemeID:         "tei.std.hashId.v1",
		Purpose:          "example.record",
		Scope:            "example.application",
		InputKind:        "bytes",
		Canonicalization: "raw-bytes/v1",
		Algorithm:        AlgorithmSHA256,
		Mode:             ModeEncodedDigest,
		OutputBits:       256,
		Encoding:         EncodingHex,
	}
}

func TestSHA256Golden(t *testing.T) {
	s := fixtureScheme()
	r, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "01e745ee1728403d83ebe8ccd903b375f40d40fc1754fd0bc41ca249142f838d"
	if r.Text != want || hex.EncodeToString(r.Digest) != want {
		t.Fatalf("got text=%q digest=%x, want=%s", r.Text, r.Digest, want)
	}
	sum := sha256.Sum256(frame(s, []byte("hello TEI")))
	if !bytes.Equal(r.Digest, sum[:]) {
		// Deliberately compare against SHA-256 of the framed input, not raw input.
		t.Fatal("digest does not bind the scheme frame")
	}
}

func TestEncodingOnlyChangesPresentation(t *testing.T) {
	s := fixtureScheme()
	first, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, alphabet string
	}{
		{EncodingBase32, ""},
		{EncodingBase64URL, ""},
		{EncodingRadix, "0123456789"},
		{EncodingRadix, "0123456789abcdefghijklmnopqrstuvwxyz"},
		{EncodingRadix, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"},
	} {
		s.Encoding, s.Alphabet = tc.name, tc.alphabet
		r, err := Derive(s, []byte("hello TEI"))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !bytes.Equal(first.Digest, r.Digest) {
			t.Fatalf("%s changed binary digest", tc.name)
		}
		if r.Text == "" {
			t.Fatalf("%s empty text", tc.name)
		}
	}
	s.Encoding, s.Alphabet = EncodingRadix, "0123456789"
	r, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Text) != 78 { // ceil(log10(2^256)) = 78
		t.Fatalf("unexpected 256-bit decimal width: %d", len(r.Text))
	}
}

func TestSHAKE256Golden(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.OutputBits, s.Encoding = AlgorithmSHAKE256, 160, EncodingBase32
	r, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "y5bb6zwgps3bs443yyc73wx767dz2x5j" ||
		hex.EncodeToString(r.Digest) != "c7421f66c67cb619739bc605fddafff7c79d5fa9" {
		t.Fatalf("SHAKE256 framed golden mismatch: %q / %x", r.Text, r.Digest)
	}
	recovered, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(r.Text))
	if err != nil || !bytes.Equal(recovered, r.Digest) {
		t.Fatalf("base32 round-trip failed: %v", err)
	}
}

func TestExactCharactersGoldenAndDomainSeparation(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.Mode = AlgorithmSHAKE256, ModeExactCharacters
	s.OutputBits, s.Encoding = 0, ""
	s.Alphabet, s.Length = "0123456789abcdefghjkmnpqrstvwxyz", 32
	r, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "8t24m4aa03dc15gxscz9tbk7es640wm0" {
		t.Fatalf("exact-character golden mismatch: %q", r.Text)
	}
	if r.Digest != nil {
		t.Fatal("exactCharacters is not a reversible digest representation")
	}
	again, err := Derive(s, []byte("hello TEI"))
	if err != nil || again.Text != r.Text {
		t.Fatalf("nondeterminism: %q / %v", again.Text, err)
	}
	s.Length = 31
	changed, err := Derive(s, []byte("hello TEI"))
	if err != nil || changed.Text == r.Text[:31] {
		t.Fatal("length must be bound to the XOF domain")
	}
	s.Length = 32
	s.Scope = "example.other"
	changed, err = Derive(s, []byte("hello TEI"))
	if err != nil || changed.Text == r.Text {
		t.Fatal("scope must be bound to the XOF domain")
	}
}

func TestRejectionSamplingNonPowerOfTwoAndCSHAKE(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.Mode = AlgorithmCSHAKE256, ModeExactCharacters
	s.OutputBits, s.Encoding = 0, ""
	s.Alphabet, s.Length = "0123456789", 30
	r1, err := Derive(s, []byte("hello TEI"))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Derive(s, []byte("hello TEI"))
	if err != nil || r1.Text != r2.Text || len(r1.Text) != s.Length {
		t.Fatalf("cSHAKE exact output not reproducible: %q %q %v", r1.Text, r2.Text, err)
	}
	for _, b := range []byte(r1.Text) {
		if b < '0' || b > '9' {
			t.Fatalf("character outside decimal alphabet: %q", b)
		}
	}
	s.Algorithm = AlgorithmSHAKE256
	r3, err := Derive(s, []byte("hello TEI"))
	if err != nil || r3.Text == r1.Text {
		t.Fatal("SHAKE and cSHAKE should be distinct")
	}
	// Independent Python hashlib SHAKE256 vector for A=10, L=30.
	// Input byte #19 is 252 and must be rejected (threshold=250).
	if r3.Text != "763327628360903151303303983778" {
		t.Fatalf("rejection-sampling golden mismatch: %q", r3.Text)
	}
}

func TestRejectInvalidSchemes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Scheme)
	}{
		{"empty scheme", func(s *Scheme) { s.SchemeID = "" }},
		{"non-ascii purpose", func(s *Scheme) { s.Purpose = "試験" }},
		{"unsupported normalization", func(s *Scheme) { s.Canonicalization = "json/v1" }},
		{"unsupported algorithm", func(s *Scheme) { s.Algorithm = "blake3" }},
		{"implicit truncation", func(s *Scheme) { s.OutputBits = 160 }},
		{"non-byte output", func(s *Scheme) { s.OutputBits = 159 }},
		{"length with digest", func(s *Scheme) { s.Length = 32 }},
		{"unknown encoding", func(s *Scheme) { s.Encoding = "base1000" }},
		{"alphabet with named encoding", func(s *Scheme) { s.Alphabet = "0123456789" }},
		{"repeated alphabet", func(s *Scheme) { s.Encoding, s.Alphabet = EncodingRadix, "0012345" }},
		{"unicode alphabet", func(s *Scheme) { s.Encoding, s.Alphabet = EncodingRadix, "abβ" }},
		{"exact with SHA256", func(s *Scheme) {
			s.Mode, s.OutputBits, s.Encoding, s.Length, s.Alphabet =
				ModeExactCharacters, 0, "", 16, "0123456789"
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := fixtureScheme()
			tc.mutate(&s)
			if _, err := Derive(s, []byte("source")); err == nil {
				t.Fatal("invalid scheme must fail closed")
			}
		})
	}
}

func TestPopulationCollisionBudget(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.Mode = AlgorithmSHAKE256, ModeExactCharacters
	s.OutputBits, s.Encoding = 0, ""
	s.Alphabet, s.Length = "0123456789", 20
	p, err := CollisionProbability(s, 1_000_000_000)
	if err != nil || p < 0.004 || p > 0.006 {
		t.Fatalf("unexpected birthday probability: %.8f / %v", p, err)
	}
	if err := CheckCollisionBudget(s, 1_000_000_000, 1e-6); err == nil {
		t.Fatal("unsafe decimal length should exceed collision budget")
	}
	s.Alphabet, s.Length = "0123456789abcdefghjkmnpqrstvwxyz", 32
	if err := CheckCollisionBudget(s, 1_000_000_000, 1e-12); err != nil {
		t.Fatal(err)
	}
	if err := CheckCollisionBudget(s, 1, -1); err == nil {
		t.Fatal("invalid budget not rejected")
	}
}

func TestCollisionBudgetConservativeSmallSpaceAndPigeonhole(t *testing.T) {
	s := fixtureScheme()
	s.Algorithm, s.Mode = AlgorithmSHAKE256, ModeExactCharacters
	s.OutputBits, s.Encoding, s.Alphabet, s.Length = 0, "", "01", 1

	estimate, err := CollisionProbability(s, 2)
	if err != nil || estimate >= 0.5 {
		t.Fatalf("birthday approximation must not be confused with exact risk: %.9g, %v", estimate, err)
	}
	upper, err := CollisionRiskUpperBound(s, 2)
	if err != nil || upper < 0.5 || upper > 0.500000000000001 {
		t.Fatalf("union bound for two draws from two values must be 0.5: %.9g, %v", upper, err)
	}
	if err := CheckCollisionBudget(s, 2, 0.4); err == nil {
		t.Fatal("budget between optimistic approximation and union bound was accepted")
	}
	if err := CheckCollisionBudget(s, 2, 0.51); err != nil {
		t.Fatalf("valid small-space upper-bound policy rejected: %v", err)
	}
	if probability, err := CollisionProbability(s, 3); err != nil || probability != 1 {
		t.Fatalf("population exceeding address space must mean certain collision: %.9g, %v", probability, err)
	}
	if err := CheckCollisionBudget(s, 3, 1); err == nil {
		t.Fatal("more subjects than finite IDs cannot be accepted even at ceiling 1")
	}

	// The same certainty must be detected for 8-bit digest mode.
	s.Mode, s.Alphabet, s.Length = ModeEncodedDigest, "", 0
	s.Encoding, s.OutputBits = EncodingHex, 8
	if err := CheckCollisionBudget(s, 257, 1); err == nil {
		t.Fatal("257 allocations in an 8-bit space cannot pass")
	}
	if err := CheckCollisionBudget(s, 2, 0.1); err != nil {
		t.Fatalf("sufficient bound for 8-bit case rejected: %v", err)
	}

	// A 4096-bit address space has positive risk for n=2, even when
	// float64 represents the tiny approximation as zero.
	s.OutputBits = 4096
	if estimate, err := CollisionProbability(s, 2); err != nil || estimate != 0 {
		t.Fatalf("expected float64 underflow in long digest fixture: %.9g %v", estimate, err)
	}
	if upper, err := CollisionRiskUpperBound(s, 2); err != nil || upper <= 0 {
		t.Fatalf("collision upper bound must not report false zero: %.9g %v", upper, err)
	}
	if err := CheckCollisionBudget(s, 2, 0); err == nil {
		t.Fatal("zero collision-risk budget accepted due to float64 underflow")
	}
	if err := CheckCollisionBudget(s, 2, 1e-12); err != nil {
		t.Fatalf("positive tolerance for large address space rejected: %v", err)
	}
}

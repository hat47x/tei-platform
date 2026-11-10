package hashid

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestSharedCrossRuntimeVectors(t *testing.T) {
	// The same fixture is consumed by hashid.crosslang.test.mjs.
	// The Node test independently frames and hashes with node:crypto.
	raw, err := os.ReadFile("vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion string `json:"schemaVersion"`
		InputUTF8     string `json:"inputUtf8"`
		Scheme        struct {
			SchemeID         string `json:"schemeId"`
			Purpose          string `json:"purpose"`
			Scope            string `json:"scope"`
			InputKind        string `json:"inputKind"`
			Canonicalization string `json:"canonicalization"`
		} `json:"scheme"`
		Cases struct {
			SHA256Hex           string `json:"sha256hex"`
			SHAKE160Hex         string `json:"shake160hex"`
			SHAKEExact32        string `json:"shakeExact32"`
			SHAKEExactDecimal30 string `json:"shakeExactDecimal30"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != "reference-hashid-v1" {
		t.Fatalf("unexpected fixture schema %q", fixture.SchemaVersion)
	}
	base := Scheme{
		SchemeID:         fixture.Scheme.SchemeID,
		Purpose:          fixture.Scheme.Purpose,
		Scope:            fixture.Scheme.Scope,
		InputKind:        fixture.Scheme.InputKind,
		Canonicalization: fixture.Scheme.Canonicalization,
	}
	for _, tc := range []struct {
		name       string
		algorithm  string
		mode       string
		outputBits int
		encoding   string
		alphabet   string
		length     int
		want       string
		digestHex  bool
	}{
		{"sha256", AlgorithmSHA256, ModeEncodedDigest, 256, EncodingHex, "", 0, fixture.Cases.SHA256Hex, true},
		{"shake160", AlgorithmSHAKE256, ModeEncodedDigest, 160, EncodingHex, "", 0, fixture.Cases.SHAKE160Hex, true},
		{"exact32", AlgorithmSHAKE256, ModeExactCharacters, 0, "", "0123456789abcdefghjkmnpqrstvwxyz", 32, fixture.Cases.SHAKEExact32, false},
		{"exactDecimal30", AlgorithmSHAKE256, ModeExactCharacters, 0, "", "0123456789", 30, fixture.Cases.SHAKEExactDecimal30, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := base
			s.Algorithm, s.Mode = tc.algorithm, tc.mode
			s.OutputBits, s.Encoding = tc.outputBits, tc.encoding
			s.Alphabet, s.Length = tc.alphabet, tc.length
			result, err := Derive(s, []byte(fixture.InputUTF8))
			if err != nil {
				t.Fatal(err)
			}
			got := result.Text
			if tc.digestHex {
				got = hex.EncodeToString(result.Digest)
			}
			if got != tc.want {
				t.Fatalf("cross-runtime mismatch: got=%q, want=%q", got, tc.want)
			}
		})
	}
}

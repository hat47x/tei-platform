package hashid

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func schemeWireFixture(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile("scheme_wire_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion string            `json:"schemaVersion"`
		Schemes       map[string]string `json:"schemes"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != "tei.hashid.wire-v1.fixture" {
		t.Fatalf("invalid fixture version %q", fixture.SchemaVersion)
	}
	return fixture.Schemes
}

func TestCanonicalSchemeJSONCrossLanguageFixture(t *testing.T) {
	fixtures := schemeWireFixture(t)
	for name, canonical := range fixtures {
		t.Run(name, func(t *testing.T) {
			s, err := ParseSchemeJSON([]byte(canonical))
			if err != nil {
				t.Fatal(err)
			}
			actual, err := MarshalSchemeJSON(s)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, []byte(canonical)) {
				t.Fatalf("wire mismatch: got=%s want=%s", actual, canonical)
			}
			input := []byte("hello TEI")
			before, err := Derive(s, input)
			if err != nil {
				t.Fatal(err)
			}
			after, err := Derive(s, input)
			if err != nil || before.Text != after.Text ||
				!bytes.Equal(before.Digest, after.Digest) {
				t.Fatal("scheme wire changed derivation result")
			}
		})
	}
}

func TestSchemeJSONRefusesSilentWireInterpretation(t *testing.T) {
	fixtures := schemeWireFixture(t)
	exact := fixtures["exact32"]
	special := fixtures["specialAscii"]
	unknownRoot := strings.Replace(exact, `"format":`, `"newKey":1,"format":`, 1)
	unknownScheme := strings.Replace(exact, `"schemeId":`, `"unrecognized":"x","schemeId":`, 1)
	duplicate := strings.Replace(exact, `"schemeId":`, `"schemeId":"bad","schemeId":`, 1)
	missing := strings.Replace(exact, `"outputBits":0,`, "", 1)
	nullField := strings.Replace(exact, `"alphabet":"0123456789abcdefghjkmnpqrstvwxyz"`, `"alphabet":null`, 1)
	decimal := strings.Replace(exact, `"length":32`, `"length":32.0`, 1)
	noncanonicalEscape := strings.Replace(special, "<", `\u003c`, 1)
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"unknown root", unknownRoot},
		{"unknown scheme field", unknownScheme},
		{"duplicate key", duplicate},
		{"missing required zero value", missing},
		{"null in required field", nullField},
		{"noninteger number", decimal},
		{"unknown wire version", strings.Replace(exact, SchemeWireFormat, "tei.hashid.scheme/v2", 1)},
		{"whitespace", " " + exact},
		{"trailing whitespace", exact + "\n"},
		{"trailing object", exact + "{}"},
		{"HTML-escaped alphabet", noncanonicalEscape},
		{"oversized input", strings.Repeat("x", maxSchemeJSONBytes+1)},
		{"wrong JSON kind", "[]"},
		{"invalid Unicode", string([]byte{0xff, 0xfe})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseSchemeJSON([]byte(tc.raw)); err == nil {
				t.Fatalf("noncanonical scheme was accepted: %q", tc.raw)
			}
		})
	}
}

func TestSchemeWireDoesNotRebindEncodingIntoDigest(t *testing.T) {
	a := fixtureScheme()
	a.OutputBits = 160
	a.Algorithm = AlgorithmSHAKE256
	a.Encoding = EncodingHex
	one, err := Derive(a, []byte("constant source"))
	if err != nil {
		t.Fatal(err)
	}
	a.Encoding = EncodingBase32
	two, err := Derive(a, []byte("constant source"))
	if err != nil {
		t.Fatal(err)
	}
	wire1, err := MarshalSchemeJSON(one.Scheme)
	if err != nil {
		t.Fatal(err)
	}
	wire2, err := MarshalSchemeJSON(two.Scheme)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(wire1, wire2) {
		t.Fatal("representation change must be explicit in scheme wire")
	}
	if !bytes.Equal(one.Digest, two.Digest) {
		t.Fatal("representation change must not modify the binary digest")
	}
}

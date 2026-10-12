package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

func TestParseJSONRejectsUnpairedSurrogateEscapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"high", `"\uD800"`},
		{"low", `"\uDC00"`},
		{"high followed by a scalar", `"\uD800\u0041"`},
		{"high followed by high", `"\uD800\uDBFF"`},
		{"high followed by plain text", `"\uD800a"`},
		{"in object key", `{"\uD800":1}`},
		{"in nested array", `[{"x":["\uDC00"]}]`},
		{"in mixed case escape", `"\udAbC"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := transform.ParseJSON([]byte(tt.raw)); err == nil {
				t.Fatalf("malformed surrogate accepted: %s", tt.name)
			}
		})
	}
}

func TestParseJSONValidSurrogatePairsAndEscapedBackslash(t *testing.T) {
	for _, raw := range []string{
		`"\uD83D\uDE00"`,
		`{"\uD83D\uDE00":"\ud83d\ude00"}`,
		`"\\uD800"`, // Literal backslash followed by 'uD800'.
		`"�"`,       // Explicit replacement character remains permitted.
		`"\uFFFD"`,
	} {
		value, err := transform.ParseJSON([]byte(raw))
		if err != nil {
			t.Fatalf("valid JSON %q rejected: %v", raw, err)
		}
		canonical, err := value.JSON()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := transform.ParseJSON(canonical); err != nil {
			t.Fatalf("canonical form %q not parseable: %v", canonical, err)
		}
	}
}

func TestParseSourceJSONRejectsUnpairedSurrogateInIdentityAndLiteral(t *testing.T) {
	source := canonicalTestMetaSource(t)
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.ParseSourceJSON(raw); err != nil {
		t.Fatalf("valid baseline Source rejected: %v", err)
	}

	identityField := []byte(`"contractVersion":"` + source.Identity.ContractVersion + `"`)
	withIdentityEscape := bytes.Replace(raw, identityField, []byte(`"contractVersion":"\uD800"`), 1)
	if bytes.Equal(raw, withIdentityEscape) {
		t.Fatal("identity fixture was not replaced")
	}
	if _, err := transform.ParseSourceJSON(withIdentityEscape); err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("expected surrogate diagnostic for Source identity, got %v", err)
	}

	// Use an otherwise valid literal expression to exercise raw-message parsing.
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{Op: transform.OpLiteral, Literal: json.RawMessage(`"ok"`)}
	withLiteral, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.ParseSourceJSON(withLiteral); err != nil {
		t.Fatalf("valid literal Source rejected: %v", err)
	}
	withLiteralEscape := bytes.Replace(withLiteral, []byte(`"literal":"ok"`), []byte(`"literal":"\uDC00"`), 1)
	if bytes.Equal(withLiteral, withLiteralEscape) {
		t.Fatal("literal fixture was not replaced")
	}
	if _, err := transform.ParseSourceJSON(withLiteralEscape); err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("expected surrogate diagnostic for Source literal, got %v", err)
	}

	// Verify the error does not echo the invalid field contents.
	if _, err := transform.ParseSourceJSON(withLiteralEscape); err != nil &&
		strings.Contains(err.Error(), "DC00") {
		t.Fatalf("diagnostic exposed raw JSON value: %v", err)
	}
}

func TestParseSourceJSONUnicodeScalarCanonicalization(t *testing.T) {
	source := canonicalTestMetaSource(t)
	source.Identity.ContractVersion = "contract/😀"
	direct, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	escaped := bytes.Replace(direct, []byte("contract/😀"), []byte(`contract/\uD83D\uDE00`), 1)
	if bytes.Equal(direct, escaped) {
		t.Fatal("unicode fixture was not escaped")
	}
	parsedDirect, err := transform.ParseSourceJSON(direct)
	if err != nil {
		t.Fatalf("valid UTF-8 scalar rejected: %v", err)
	}
	parsedEscaped, err := transform.ParseSourceJSON(escaped)
	if err != nil {
		t.Fatalf("valid UTF-16 surrogate pair rejected: %v", err)
	}
	if parsedDirect.Identity.ContractVersion != parsedEscaped.Identity.ContractVersion {
		t.Fatal("escaped and direct Unicode scalars decoded differently")
	}
	fingerprintDirect, err := transform.SourceFingerprint(parsedDirect)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintEscaped, err := transform.SourceFingerprint(parsedEscaped)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprintDirect != fingerprintEscaped {
		t.Fatal("equivalent Unicode spellings changed canonical Source fingerprint")
	}

	// A malformed Source must not alias the fingerprint of explicit U+FFFD.
	invalid := bytes.Replace(escaped, []byte(`\uD83D\uDE00`), []byte(`\uD83D`), 1)
	if bytes.Equal(invalid, escaped) {
		t.Fatal("invalid surrogate fixture was not replaced")
	}
	if _, err := transform.ParseSourceJSON(invalid); err == nil || !strings.Contains(err.Error(), "surrogate") {
		t.Fatalf("invalid Unicode escape must be rejected, got %v", err)
	}
}

func TestMaterializeSourceValidPairedSurrogateLiteral(t *testing.T) {
	source := canonicalTestMetaSource(t)
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{
		Op:      transform.OpLiteral,
		Literal: json.RawMessage(`"\uD83D\uDE00"`),
	}
	raw, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatalf("valid surrogate-pair literal rejected: %v", err)
	}
	parsed, err := transform.ParseSourceJSON(raw)
	if err != nil {
		t.Fatalf("valid materialized Source cannot be parsed: %v", err)
	}
	after, err := transform.SourceFingerprint(parsed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("materialized valid Unicode literal changed Source fingerprint")
	}
}

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
	source := transform.TransformSourceRealizerV1()
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}

	identityField := []byte(`"contractVersion":"` + source.Identity.ContractVersion + `"`)
	withIdentityEscape := bytes.Replace(raw, identityField, []byte(`"contractVersion":"\uD800"`), 1)
	if bytes.Equal(raw, withIdentityEscape) {
		t.Fatal("identity fixture was not replaced")
	}
	if _, err := transform.ParseSourceJSON(withIdentityEscape); err == nil {
		t.Fatal("unpaired surrogate in Source identity was accepted")
	}

	// Use an otherwise valid literal expression to exercise raw-message parsing.
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{Op: transform.OpLiteral, Literal: json.RawMessage(`"ok"`)}
	withLiteral, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	withLiteralEscape := bytes.Replace(withLiteral, []byte(`"literal":"ok"`), []byte(`"literal":"\uDC00"`), 1)
	if bytes.Equal(withLiteral, withLiteralEscape) {
		t.Fatal("literal fixture was not replaced")
	}
	if _, err := transform.ParseSourceJSON(withLiteralEscape); err == nil {
		t.Fatal("unpaired surrogate in Source literal was accepted")
	}

	// Verify the error does not echo the invalid field contents.
	if _, err := transform.ParseSourceJSON(withLiteralEscape); err != nil &&
		strings.Contains(err.Error(), "DC00") {
		t.Fatalf("diagnostic exposed raw JSON value: %v", err)
	}
}

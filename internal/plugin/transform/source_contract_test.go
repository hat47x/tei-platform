package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

func TestCanonicalSourcePathsShareStrictReadableBytes(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	artifact, digest, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact) > transform.MaxSerializedSourceBytes {
		t.Fatalf("canonical artifact exceeds limit: %d", len(artifact))
	}
	if !bytes.HasSuffix(artifact, []byte("\n")) {
		t.Fatal("canonical artifact lacks trailing newline")
	}
	if !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("unexpected materialized digest: %q", digest)
	}
	parsed, err := transform.ParseSourceJSON(artifact)
	if err != nil {
		t.Fatalf("materialized Source is not readable: %v", err)
	}
	originalFingerprint, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	parsedFingerprint, err := transform.SourceFingerprint(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if originalFingerprint != parsedFingerprint {
		t.Fatalf("canonical fingerprints differ: %s, %s", originalFingerprint, parsedFingerprint)
	}
	programmatic, err := transform.SourceValue(source)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := transform.SourceValue(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if !transform.Equal(programmatic, decoded) {
		t.Fatal("SourceValue differs across programmatic and file origins")
	}
}

func TestCanonicalSourcePathsRejectOversizedProgrammaticInput(t *testing.T) {
	tests := map[string]func(*transform.Source){
		"string": func(s *transform.Source) {
			s.Identity.ContractVersion = strings.Repeat("a", transform.MaxSerializedSourceBytes)
		},
		"literal": func(s *transform.Source) {
			s.Program.Output = transform.Expr{
				Op:      transform.OpLiteral,
				Literal: json.RawMessage("\"" + strings.Repeat("a", transform.MaxSerializedSourceBytes) + "\""),
			}
		},
		"expanded literal": func(s *transform.Source) {
			s.Program.Output = transform.Expr{
				Op:      transform.OpLiteral,
				Literal: json.RawMessage("\"" + strings.Repeat("<", 2<<20) + "\""),
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			source := transform.TransformSourceRealizerV1()
			mutate(&source)
			if _, _, err := transform.MaterializeSource(source); err == nil {
				t.Fatal("MaterializeSource accepted oversized Source")
			}
			if _, err := transform.SourceFingerprint(source); err == nil {
				t.Fatal("SourceFingerprint accepted oversized Source")
			}
			if _, err := transform.SourceValue(source); err == nil {
				t.Fatal("SourceValue accepted oversized Source")
			}
		})
	}
}

func TestCanonicalSourcePathsRejectCyclicExpressions(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	cycle := &transform.Expr{Op: transform.OpSelectField, Field: "x"}
	cycle.From = cycle
	source.Program.Output = *cycle
	if _, _, err := transform.MaterializeSource(source); err == nil {
		t.Fatal("MaterializeSource did not reject cyclic expressions")
	}
}

func TestStrictJSONRejectsUnpairedUTF16Escapes(t *testing.T) {
	for _, raw := range []string{
		"{\"x\":\"\\uD800\"}",
		"{\"x\":\"\\uDBFF\\u0041\"}",
		"{\"x\":\"\\uDC00\"}",
		"{\"\\uDFFF\":true}",
		"{\"values\":[\"\\uD800\"]}",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := transform.ParseJSON([]byte(raw)); err == nil {
				t.Fatal("accepted an unpaired surrogate escape")
			}
		})
	}
	// A double-escaped backslash is not a UTF-16 escape.
	if _, err := transform.ParseJSON([]byte("{\"x\":\"\\\\uD800\"}")); err != nil {
		t.Fatalf("rejected a literal backslash: %v", err)
	}
	parsed, err := transform.ParseJSON([]byte("{\"face\":\"\\uD83D\\uDE00\"}"))
	if err != nil {
		t.Fatalf("rejected a valid surrogate pair: %v", err)
	}
	face, ok := parsed.Field("face")
	if !ok || face.String != "😀" {
		t.Fatalf("valid surrogate pair decoded incorrectly: %#v", face)
	}
}

func TestStrictSourceParserDoesNotConflateInvalidEscapeWithReplacementRune(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	raw, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	bad := bytes.Replace(raw, []byte("tei.plugin-transform/v1"), []byte("\\uD800"), 1)
	if _, err := transform.ParseSourceJSON(bad); err == nil {
		t.Fatal("invalid contract version surrogate escape was accepted")
	}
	good := bytes.Replace(raw, []byte("tei.plugin-transform/v1"), []byte("�"), 1)
	if _, err := transform.ParseSourceJSON(good); err != nil {
		t.Fatalf("explicit replacement rune must not be confused with malformed JSON: %v", err)
	}
}

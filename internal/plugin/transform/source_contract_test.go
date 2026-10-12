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


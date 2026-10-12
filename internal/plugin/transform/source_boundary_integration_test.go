package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

// This regression crosses the size-limited programmatic Source path and the
// strict Unicode file boundary. Both behaviors must agree on canonical bytes.
func TestSourceBoundaryPairedUnicodeCanonicalizesAcrossOrigins(t *testing.T) {
	source := canonicalTestMetaSource(t)
	source.Program.Assertions = nil
	source.Identity.ContractVersion = "contract/😀"
	source.Program.Output = transform.Expr{
		Op:      transform.OpLiteral,
		Literal: json.RawMessage("\"\\uD83D\\uDE00<>&\""),
	}
	direct := source
	direct.Program.Output.Literal = json.RawMessage("\"😀<>&\"")

	artifact, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	directArtifact, _, err := transform.MaterializeSource(direct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(artifact, directArtifact) {
		t.Fatal("paired-surrogate and UTF-8 literal produced different artifacts")
	}
	parsed, err := transform.ParseSourceJSON(artifact)
	if err != nil {
		t.Fatalf("materialized Source cannot cross strict JSON boundary: %v", err)
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
		t.Fatal("Source fingerprint changed after strict Unicode round trip")
	}
	beforeValue, err := transform.SourceValue(source)
	if err != nil {
		t.Fatal(err)
	}
	afterValue, err := transform.SourceValue(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if !transform.Equal(beforeValue, afterValue) {
		t.Fatal("SourceValue differs across programmatic and file origins")
	}
}

func TestSourceBoundaryExactLimitWithPairedUnicode(t *testing.T) {
	source := canonicalTestMetaSource(t)
	source.Program.Assertions = nil
	source.Identity.ContractVersion = "contract/😀"
	source.Program.Output = transform.Expr{
		Op:      transform.OpLiteral,
		Literal: json.RawMessage("\"\\uD83D\\uDE00<>&\""),
	}
	base, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	padding := transform.MaxSerializedSourceBytes - len(base)
	if padding < 0 {
		t.Fatal("baseline Source unexpectedly exceeds the limit")
	}
	source.Identity.ContractVersion += strings.Repeat("a", padding)

	atLimit, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatalf("Unicode Source exactly at 8 MiB rejected: %v", err)
	}
	if len(atLimit) != transform.MaxSerializedSourceBytes {
		t.Fatalf("Unicode Source size = %d; expected %d", len(atLimit), transform.MaxSerializedSourceBytes)
	}
	parsed, err := transform.ParseSourceJSON(atLimit)
	if err != nil {
		t.Fatalf("exact-limit materialized Source cannot be read: %v", err)
	}
	sourceFP, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	parsedFP, err := transform.SourceFingerprint(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if sourceFP != parsedFP {
		t.Fatal("exact-limit Unicode Source fingerprint drifted")
	}
	source.Identity.ContractVersion += "a"
	if _, _, err := transform.MaterializeSource(source); err == nil {
		t.Fatal("Unicode Source over 8 MiB must be rejected")
	}
}

func TestSourceBoundaryUnpairedSurrogateNeverMaterializes(t *testing.T) {
	for _, literal := range []string{"\"\\uD800\"", "\"\\uDC00\"", "\"\\uD800\\u0041\""} {
		source := canonicalTestMetaSource(t)
		source.Program.Assertions = nil
		source.Program.Output = transform.Expr{Op: transform.OpLiteral, Literal: json.RawMessage(literal)}
		if _, _, err := transform.MaterializeSource(source); err == nil {
			t.Fatalf("MaterializeSource accepted malformed literal %q", literal)
		}
		if _, err := transform.SourceFingerprint(source); err == nil {
			t.Fatalf("SourceFingerprint accepted malformed literal %q", literal)
		}
		if _, err := transform.SourceValue(source); err == nil {
			t.Fatalf("SourceValue accepted malformed literal %q", literal)
		}
	}

	// Rejection must also hold when malformed JSON enters as a file,
	// rather than through the programmatic constructors.
	source := canonicalTestMetaSource(t)
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{Op: transform.OpLiteral, Literal: json.RawMessage("\"ok\"")}
	valid, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	malformed := bytes.Replace(valid, []byte("\"literal\":\"ok\""), []byte("\"literal\":\"\\uD800\""), 1)
	if bytes.Equal(malformed, valid) {
		t.Fatal("malformed fixture was not applied")
	}
	if _, err := transform.ParseSourceJSON(malformed); err == nil {
		t.Fatal("ParseSourceJSON accepted malformed surrogate from file")
	}
}

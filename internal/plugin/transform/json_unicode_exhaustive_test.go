package transform_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"

	"tei/internal/plugin/transform"
)

// Every single high or low surrogate must be rejected before encoding/json
// can silently replace it with U+FFFD.
func TestParseJSONRejectsEveryUnpairedSurrogate(t *testing.T) {
	for unit := 0xd800; unit <= 0xdfff; unit++ {
		raw := fmt.Sprintf(`"\u%04x"`, unit)
		if _, err := transform.ParseJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted unpaired UTF-16 surrogate U+%04X", unit)
		}
	}
}

// Check both edges of the low-surrogate range for every high surrogate.
// The resulting scalar must match the Unicode UTF-16 definition.
func TestParseJSONDecodesPairedSurrogateBoundaryValues(t *testing.T) {
	for high := 0xd800; high <= 0xdbff; high++ {
		for _, low := range []int{0xdc00, 0xdfff} {
			raw := fmt.Sprintf(`"\u%04x\u%04x"`, high, low)
			value, err := transform.ParseJSON([]byte(raw))
			if err != nil {
				t.Fatalf("rejected pair U+%04X/U+%04X: %v", high, low, err)
			}
			want := string(utf16.DecodeRune(rune(high), rune(low)))
			if value.Kind != transform.ValueString || value.String != want {
				t.Fatalf("decoded pair U+%04X/U+%04X incorrectly", high, low)
			}
		}
	}
}

// An odd number of consecutive backslashes leaves a real Unicode escape.
// An even number escapes the backslash itself, making "uD800" literal text.
func TestParseJSONUnicodeBackslashParity(t *testing.T) {
	for count := 0; count <= 9; count++ {
		raw := `"` + strings.Repeat(`\`, count) + `uD800"`
		_, err := transform.ParseJSON([]byte(raw))
		wantReject := count%2 == 1
		if (err != nil) != wantReject {
			t.Fatalf("backslash count %d: rejection=%v, expected=%v", count, err != nil, wantReject)
		}
	}
}

// The programmatic Source path must not accept malformed literal JSON and
// derive a valid artifact or fingerprint from encoding/json's replacement.
func TestProgrammaticSourceRejectsMalformedSurrogate(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	source.Program.Assertions = nil
	source.Program.Output = transform.Expr{
		Op:      transform.OpLiteral,
		Literal: json.RawMessage(`"\uD800"`),
	}
	if _, err := transform.SourceFingerprint(source); err == nil {
		t.Fatal("SourceFingerprint accepted an unpaired surrogate")
	}
	if _, _, err := transform.MaterializeSource(source); err == nil {
		t.Fatal("MaterializeSource accepted an unpaired surrogate")
	}
	if _, err := transform.SourceValue(source); err == nil {
		t.Fatal("SourceValue accepted an unpaired surrogate")
	}
}

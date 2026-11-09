package transform

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// Read the authoritative builder program from the repository rather than
// recreating its transform AST in Go.
func repositoryBudgetSource(t *testing.T, name string) Source {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve transform test source path")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "..", "reference", "plugin-self-build", "builder-sources", name))
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseSourceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func TestSourceSizePreflightMatchesCanonicalJSONEncoding(t *testing.T) {
	cases := []struct {
		name   string
		source Source
	}{
		{name: "source realizer", source: repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")},
		{name: "plugin package", source: repositoryBudgetSource(t, "plugin-package-v1.transform.json")},
	}
	withEscapes := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
	withEscapes.Program.Output = Expr{
		Op:      OpLiteral,
		Literal: json.RawMessage("\"\\u2028<>&\\\\\""),
	}
	// Use a literal as output so the existing input bindings remain valid.
	cases = append(cases, struct {
		name   string
		source Source
	}{name: "escapes and omitted fields", source: withEscapes})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := NormalizeSource(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(normalized)
			if err != nil {
				t.Fatal(err)
			}
			if err := sourceSerializedSize(normalized, len(raw)); err != nil {
				t.Fatalf("size preflight rejected exactly fitting JSON: %v", err)
			}
			if err := sourceSerializedSize(normalized, len(raw)-1); err == nil {
				t.Fatal("size preflight underestimated the serialized size")
			}
			// The size check reserves a byte for the final line feed.
			if err := sourceSerializedSize(normalized, MaxSerializedSourceBytes-1); err != nil {
				t.Fatalf("ordinary reference Source is oversized: %v", err)
			}
		})
	}
}

func TestSourceSizePreflightMatchesDecodedUnicodeEscapes(t *testing.T) {
	source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
	source.Program.Output = Expr{
		Op:      OpLiteral,
		Literal: json.RawMessage(`"\u0061\uD83D\uDE00\/<>&"`),
	}
	normalized, err := NormalizeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourceSerializedSize(source, len(canonical)); err != nil {
		t.Fatalf("preflight rejected a Source that fits after Unicode normalization: %v", err)
	}
	if err := sourceSerializedSize(source, len(canonical)-1); err == nil {
		t.Fatal("preflight accepted a Source above the exact canonical size")
	}
}

func TestSourceSizePreflightBoundsCyclesAndUnboundedStrings(t *testing.T) {
	large := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
	large.Identity.ContractVersion = strings.Repeat("x", MaxSerializedSourceBytes)
	if err := sourceSerializedSize(large, MaxSerializedSourceBytes-1); err == nil {
		t.Fatal("unbounded string escaped preflight")
	}

	cyclic := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
	p := &Expr{Op: OpSelectField, Field: "next"}
	p.From = p
	cyclic.Program.Output = *p
	if err := sourceSerializedSize(cyclic, MaxSerializedSourceBytes-1); err == nil {
		t.Fatal("cyclic expression escaped preflight")
	}
}

func TestSourceSizePreflightRejectsExpandedRawLiteralsBeforeNormalization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		literal json.RawMessage
	}{
		{
			name:    "HTML escaping expansion",
			literal: json.RawMessage("\"" + strings.Repeat("<", 2<<20) + "\""),
		},
		{
			name:    "Unicode line separator expansion",
			literal: json.RawMessage("\"" + strings.Repeat("\u2028", 2<<20) + "\""),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
			source.Program.Output = Expr{Op: OpLiteral, Literal: tc.literal}
			if err := sourceSerializedSize(source, MaxSerializedSourceBytes-1); err == nil {
				t.Fatal("preflight should reject literal whose canonical form expands beyond 8 MiB")
			}
		})
	}
}

func TestSourceSizePreflightBoundsRawLiteralElements(t *testing.T) {
	source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
	source.Program.Output = Expr{
		Op:      OpLiteral,
		Literal: json.RawMessage("[" + strings.Repeat("0,", 1_000_001) + "0]"),
	}
	if err := sourceSerializedSize(source, MaxSerializedSourceBytes-1); err == nil {
		t.Fatal("preflight should reject a raw literal with too many elements")
	}
}

func TestSourceSizePreflightNeverUnderestimatesCanonicalLiteral(t *testing.T) {
	alphabet := []rune{'a', '<', '>', '&', '\\', '"', '\u2028', '\u2029', ' ', '\t', 'é', '日', '😀', '/', '\n'}
	for seed := 0; seed < 600; seed++ {
		runes := make([]rune, seed%155)
		for i := range runes {
			runes[i] = alphabet[(seed*29+i*23+i*i)%len(alphabet)]
		}
		encodedString, err := json.Marshal(string(runes))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(encodedString)
		for _, replacement := range [][2]string{
			{"\\u003c", "<"}, {"\\u003e", ">"},
			{"\\u0026", "&"}, {"\\u2028", "\u2028"},
			{"\\u2029", "\u2029"},
		} {
			raw = strings.ReplaceAll(raw, replacement[0], replacement[1])
		}
		if seed%3 == 0 {
			raw = "{\"value\":" + raw + ", \"array\" :[1,2,3]}"
		} else if seed%3 == 1 {
			raw = "[" + raw + ", true, null]"
		}
		source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
		source.Identity.ContractVersion = strconv.Itoa(seed + 1)
		source.Program.Output = Expr{Op: OpLiteral, Literal: json.RawMessage(raw)}
		normalized, err := NormalizeSource(source)
		if err != nil {
			t.Fatalf("case %d normalization failed: %v", seed, err)
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			t.Fatal(err)
		}
		if err := sourceSerializedSize(source, len(encoded)-1); err == nil {
			t.Fatalf("case %d underestimated canonical length %d", seed, len(encoded))
		}
		if err := sourceSerializedSize(normalized, len(encoded)); err != nil {
			t.Fatalf("case %d rejected the exact normalized length: %v", seed, err)
		}
	}
}

func TestCanonicalSourceMatchesExistingNormalizedEncoding(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source Source
	}{
		{name: "source realizer", source: repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")},
		{name: "plugin package", source: repositoryBudgetSource(t, "plugin-package-v1.transform.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Keep the pre-refactor encoding as an independent byte oracle:
			// normalization followed by encoding/json, without the new helper.
			normalized, err := NormalizeSource(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			original, err := json.Marshal(normalized)
			if err != nil {
				t.Fatal(err)
			}
			got, err := canonicalSourceBytes(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, original) {
				t.Fatal("canonical bytes changed from the existing encoding")
			}
			expectedFingerprint := sha256.Sum256(original)
			fingerprint, err := SourceFingerprint(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if fingerprint != hex.EncodeToString(expectedFingerprint[:]) {
				t.Fatal("Source fingerprint changed from the existing encoding")
			}
			materialized, digest, err := MaterializeSource(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			expectedArtifact := append(append([]byte{}, original...), '\n')
			artifactSum := sha256.Sum256(expectedArtifact)
			if !bytes.Equal(materialized, expectedArtifact) ||
				digest != "sha256:"+hex.EncodeToString(artifactSum[:]) {
				t.Fatal("materialized Source diverged from the existing byte contract")
			}
			value, err := SourceValue(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			expectedValue, err := ParseJSON(original)
			if err != nil {
				t.Fatal(err)
			}
			if !Equal(value, expectedValue) {
				t.Fatal("Source value differs from the existing normalized JSON projection")
			}
		})
	}
}

func TestSourceSizePreflightBoundsUnnormalizedLiteralBytes(t *testing.T) {
	t.Run("one whitespace-heavy literal", func(t *testing.T) {
		source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
		source.Program.Output = Expr{
			Op:      OpLiteral,
			Literal: json.RawMessage("1" + strings.Repeat(" ", MaxSerializedSourceBytes)),
		}
		if err := sourceSerializedSize(source, MaxSerializedSourceBytes-1); err == nil {
			t.Fatal("preflight accepted oversized raw input that compacts to one byte")
		}
	})
	t.Run("cumulative literal bytes", func(t *testing.T) {
		source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
		spaces := strings.Repeat(" ", MaxSerializedSourceBytes/2)
		actual := Expr{Op: OpInput, Name: "transformSource"}
		source.Program.Output = Expr{
			Op:     OpAssertEnum,
			Actual: &actual,
			Allowed: []json.RawMessage{
				json.RawMessage("1" + spaces),
				json.RawMessage("2" + spaces),
			},
		}
		if err := sourceSerializedSize(source, MaxSerializedSourceBytes-1); err == nil {
			t.Fatal("preflight failed to enforce cumulative raw literal byte limit")
		}
	})
}

func TestSourceSizePreflightRandomizedCanonicalJSON(t *testing.T) {
	// Reproducible coverage of nested RawMessage values, insignificant
	// whitespace and different spellings of escaped Unicode.
	random := rand.New(rand.NewSource(1064))
	alphabet := []rune{'a', '<', '>', '&', '\\', '"', '\u2028', '\u2029', '\n', '\t', '日', 'é', '😀', 0}
	for i := 0; i < 3000; i++ {
		runes := make([]rune, random.Intn(280))
		for j := range runes {
			runes[j] = alphabet[random.Intn(len(alphabet))]
		}
		encoded, err := json.Marshal(string(runes))
		if err != nil {
			t.Fatal(err)
		}
		raw := string(encoded)
		for _, replacement := range [][2]string{
			{"\\u003c", "<"}, {"\\u003e", ">"},
			{"\\u0026", "&"}, {"\\u2028", "\u2028"},
			{"\\u2029", "\u2029"},
		} {
			raw = strings.ReplaceAll(raw, replacement[0], replacement[1])
		}
		if i%2 == 0 {
			raw = "{\"text\": " + raw + ", \"number\" : " + strconv.Itoa(i) + " }"
		}
		if i%3 == 0 {
			raw = "[" + raw + ", null]"
		}
		source := repositoryBudgetSource(t, "transform-source-realizer-v1.transform.json")
		source.Program.Output = Expr{Op: OpLiteral, Literal: json.RawMessage(raw)}
		normalized, err := NormalizeSource(source)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		want, err := json.Marshal(normalized)
		if err != nil {
			t.Fatal(err)
		}
		if err := sourceSerializedSize(source, len(want)-1); err == nil {
			t.Fatalf("case %d: size preflight underestimated %d canonical bytes", i, len(want))
		}
		if err := sourceSerializedSize(normalized, len(want)); err != nil {
			t.Fatalf("case %d: normalized source must fit exactly: %v", i, err)
		}
	}
}

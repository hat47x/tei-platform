package transform

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestSourceSizePreflightMatchesCanonicalJSONEncoding(t *testing.T) {
	cases := []struct {
		name   string
		source Source
	}{
		{name: "source realizer", source: TransformSourceRealizerV1()},
		{name: "plugin package", source: PluginPackageSourceV1()},
	}
	withEscapes := TransformSourceRealizerV1()
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

func TestSourceSizePreflightBoundsCyclesAndUnboundedStrings(t *testing.T) {
	large := TransformSourceRealizerV1()
	large.Identity.ContractVersion = strings.Repeat("x", MaxSerializedSourceBytes)
	if err := sourceSerializedSize(large, MaxSerializedSourceBytes-1); err == nil {
		t.Fatal("unbounded string escaped preflight")
	}

	cyclic := TransformSourceRealizerV1()
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
			source := TransformSourceRealizerV1()
			source.Program.Output = Expr{Op: OpLiteral, Literal: tc.literal}
			if err := sourceSerializedSize(source, MaxSerializedSourceBytes-1); err == nil {
				t.Fatal("preflight should reject literal whose canonical form expands beyond 8 MiB")
			}
		})
	}
}

func TestSourceSizePreflightBoundsRawLiteralElements(t *testing.T) {
	source := TransformSourceRealizerV1()
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
		source := TransformSourceRealizerV1()
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

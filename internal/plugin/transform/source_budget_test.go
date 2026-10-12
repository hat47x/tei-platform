package transform

import (
	"encoding/json"
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

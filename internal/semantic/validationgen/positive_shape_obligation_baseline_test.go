package validationgen_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
)

// This fixed, explicit expectation is a baseline for the first self-described
// Required Shape experiment. It is NOT a generated contract, a second semantic
// authority, product Inspector evidence, or a claim that M1/M2 is complete.
//
// The valid Rule has an identity, one numeric Attribute, no Relation, and a
// strictly positive decision. The Go target package is checked separately.
// Changing the handwritten generator to silently accept a malformed shape
// must fail this test before the obligation is moved to generated checking.
func TestPositiveRuleShapeIndependentBaseline(t *testing.T) {
	base := semantic.Validation{
		ID:        "sales.order_line.quantity_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	t.Run("valid exact shape", func(t *testing.T) {
		got, err := validationgen.GeneratePositive(base, "std.decimal", "main")
		if err != nil {
			t.Fatalf("valid numeric Rule rejected: %v", err)
		}
		if !bytes.Contains(got, []byte("source.Number(\"sales.order_line.quantity\")")) ||
			!bytes.Contains(got, []byte("value.Sign() > 0")) {
			t.Fatalf("generated Rule does not read and decide the independently chosen positive obligation:\n%s", got)
		}
	})

	cases := []struct {
		name      string
		change    func(*semantic.Validation)
		typeRef   core.CanonicalID
		wantError string
	}{
		{
			name: "missing rule identity",
			change: func(r *semantic.Validation) {
				r.ID = ""
			},
			wantError: "positive validation must have an ID",
		},
		{
			name: "missing referenced attribute",
			change: func(r *semantic.Validation) {
				r.Attribute = ""
			},
			wantError: "positive validation must have an ID",
		},
		{
			name: "unexpected relation",
			change: func(r *semantic.Validation) {
				r.Relation = "sales.order_line.items"
			},
			wantError: "positive validation must have an ID",
		},
		{
			name: "unsupported rule kind",
			change: func(r *semantic.Validation) {
				r.Kind = "unsupported"
			},
			wantError: "unsupported generated kind",
		},
		{
			name:      "non numeric resolved type",
			typeRef:   "std.text",
			wantError: "requires resolved std.decimal",
		},
		{
			name:      "unrecognized attribute type",
			typeRef:   "unknown.type",
			wantError: "requires resolved std.decimal",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule := base
			if tc.change != nil {
				tc.change(&rule)
			}
			attributeType := core.CanonicalID("std.decimal")
			if tc.typeRef != "" {
				attributeType = tc.typeRef
			}
			got, err := validationgen.GeneratePositive(rule, attributeType, "main")
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("want rejection containing %q, got error %v", tc.wantError, err)
			}
			if len(got) != 0 {
				t.Fatalf("rejected shape still produced executable Go (%d bytes)", len(got))
			}
		})
	}
	// A Go package is a Target-lowering parameter, not a TEI Rule shape.
	t.Run("invalid target package name", func(t *testing.T) {
		got, err := validationgen.GeneratePositive(base, "std.decimal", "not-a-go-identifier")
		if err == nil || !strings.Contains(err.Error(), "invalid Go package name") || len(got) != 0 {
			t.Fatalf("invalid Target package passed: generated=%d bytes, error=%v", len(got), err)
		}
	})
}

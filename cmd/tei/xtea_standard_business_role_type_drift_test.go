package main

import (
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// TestXTEAExampleRoleTypeDriftKeepsUnknownSeparateFromMismatch connects the
// checked-in example's parsed attribute to the existing role type validator.
// This mutates an in-memory World; it does not rewrite a source fixture or
// establish Provider, Target, or Owner acceptance.
func TestXTEAExampleRoleTypeDriftKeepsUnknownSeparateFromMismatch(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "examples", "xtea-lowcode"))
	if err != nil {
		t.Fatal(err)
	}

	const subjectID = "master.tax_rate.rate"
	const mismatchCode = "SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH"
	const unknownCode = "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE"

	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Subject != nil && diagnostic.Subject.ID == subjectID &&
			(diagnostic.Code == mismatchCode || diagnostic.Code == unknownCode) {
			t.Fatalf("checked-in example unexpectedly has a role/type error: %+v", diagnostic)
		}
	}

	tests := []struct {
		name             string
		typeRef          string
		wantCode         string
		foreignPrimitive bool
	}{
		{"wrong-primitive", "std.date", mismatchCode, false},
		{"unresolved-type", "master.unknown_rate_type", unknownCode, false},
		{"same-suffix-foreign-primitive", "foreign.decimal", mismatchCode, true},
		{"alias-to-foreign-primitive", "master.foreign_rate", mismatchCode, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			world := loaded.World
			world.Semantics = append([]semantic.Namespace(nil), world.Semantics...)
			if tc.foreignPrimitive {
				// A different publisher's primitive is not the trusted std.decimal
				// contract merely because its canonical ID ends in ".decimal".
				world.Semantics = append(world.Semantics, semantic.Namespace{
					ID: "foreign",
					Types: []semantic.Type{{
						ID: "foreign.decimal", Symbol: "decimal", Kind: semantic.TypePrimitive,
					}},
				})
			}
			found := false
			for i, ns := range world.Semantics {
				if ns.ID != "master" {
					continue
				}
				if tc.typeRef == "master.foreign_rate" {
					ns.Types = append(append([]semantic.Type(nil), ns.Types...), semantic.Type{
						ID: "master.foreign_rate", Symbol: "foreign_rate",
						Kind: semantic.TypeAlias, Underlying: "foreign.decimal",
					})
				}
				ns.Data = append([]semantic.Data(nil), ns.Data...)
				for j, data := range ns.Data {
					if data.ID != "master.tax_rate" {
						continue
					}
					data.Attributes = append([]semantic.Attribute(nil), data.Attributes...)
					for k, attr := range data.Attributes {
						if attr.Role == "tax-value" {
							if attr.ID != subjectID || attr.TypeRef != "std.decimal" {
								t.Fatalf("unexpected checked-in attribute: %+v", attr)
							}
							data.Attributes[k].TypeRef = core.CanonicalID(tc.typeRef)
							found = true
						}
					}
					ns.Data[j] = data
				}
				world.Semantics[i] = ns
			}
			if !found {
				t.Fatal("expected master.tax_rate tax-value attribute not found")
			}

			var codes []string
			for _, diagnostic := range validation.Validate(world) {
				if diagnostic.Subject != nil && diagnostic.Subject.ID == subjectID &&
					(diagnostic.Code == mismatchCode || diagnostic.Code == unknownCode) {
					codes = append(codes, diagnostic.Code)
				}
			}
			if len(codes) != 1 || codes[0] != tc.wantCode {
				t.Fatalf("expected exactly %s for %s, got %v", tc.wantCode, subjectID, codes)
			}
		})
	}
}

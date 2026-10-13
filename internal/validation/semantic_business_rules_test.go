package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func semanticRuleWorld() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{
		ID: "sales",
		Data: []semantic.Data{{
			ID: "sales.order_line", Symbol: "OrderLine",
			Attributes: []semantic.Attribute{
				{ID: "sales.order_line.order_id", Symbol: "orderId", TypeRef: "std.identifier"},
				{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
				{ID: "sales.order_line.item_code", Symbol: "itemCode", TypeRef: "std.code"},
			},
			CandidateKeys: []semantic.CandidateKey{{
				ID: "sales.order_line.business_line", Symbol: "businessLine",
				Attributes: []core.CanonicalID{"sales.order_line.order_id", "sales.order_line.line_number"},
			}},
			FunctionalDependencies: []semantic.FunctionalDependency{{
				ID: "sales.order_line.item_by_line", Symbol: "itemByLine",
				Determinants: []core.CanonicalID{"sales.order_line.order_id", "sales.order_line.line_number"},
				Dependents:   []core.CanonicalID{"sales.order_line.item_code"},
			}},
		}},
	}}}
}

func TestBusinessRuleValidatorPreservesExplicitCorrectDefinitions(t *testing.T) {
	if diagnostics := validateSemanticBusinessRules(semanticRuleWorld()); len(diagnostics) != 0 {
		t.Fatalf("valid logical identity was rejected: %+v", diagnostics)
	}
}

func TestBusinessRuleValidatorRejectsUnsafeWorldDefinitions(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(*semantic.Data)
		code string
	}{
		{"empty candidate", func(d *semantic.Data) { d.CandidateKeys[0].Attributes = nil }, "EMPTY_SEMANTIC_RULE_FIELDS"},
		{"unknown candidate attribute", func(d *semantic.Data) {
			d.CandidateKeys[0].Attributes = []core.CanonicalID{"sales.order_line.missing"}
		}, "UNKNOWN_SEMANTIC_RULE_ATTRIBUTE"},
		{"foreign candidate attribute", func(d *semantic.Data) {
			d.CandidateKeys[0].Attributes = []core.CanonicalID{"customer.customer.customer_id"}
		}, "UNKNOWN_SEMANTIC_RULE_ATTRIBUTE"},
		{"duplicate candidate attribute", func(d *semantic.Data) {
			d.CandidateKeys[0].Attributes = []core.CanonicalID{"sales.order_line.order_id", "sales.order_line.order_id"}
		}, "DUPLICATE_SEMANTIC_RULE_ATTRIBUTE"},
		{"missing FD determinant", func(d *semantic.Data) { d.FunctionalDependencies[0].Determinants = nil }, "EMPTY_SEMANTIC_RULE_FIELDS"},
		{"missing FD dependent", func(d *semantic.Data) { d.FunctionalDependencies[0].Dependents = nil }, "EMPTY_SEMANTIC_RULE_FIELDS"},
		{"duplicate FD dependent", func(d *semantic.Data) {
			d.FunctionalDependencies[0].Dependents = []core.CanonicalID{"sales.order_line.item_code", "sales.order_line.item_code"}
		}, "DUPLICATE_SEMANTIC_RULE_ATTRIBUTE"},
		{"unknown FD determinant", func(d *semantic.Data) {
			d.FunctionalDependencies[0].Determinants = []core.CanonicalID{"sales.order_line.unknown"}
		}, "UNKNOWN_SEMANTIC_RULE_ATTRIBUTE"},
		{"foreign ID", func(d *semantic.Data) {
			d.CandidateKeys[0].ID = "sales.other.business_line"
		}, "SEMANTIC_RULE_PARENT_MISMATCH"},
		{"duplicate candidate and FD ID", func(d *semantic.Data) {
			d.FunctionalDependencies[0].ID = d.CandidateKeys[0].ID
		}, "DUPLICATE_SEMANTIC_RULE_IDENTITY"},
		{"duplicate relation member ID", func(d *semantic.Data) {
			d.CandidateKeys[0].ID = "sales.order_line.order_id"
		}, "DUPLICATE_SEMANTIC_RULE_IDENTITY"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := semanticRuleWorld()
			data := &world.Semantics[0].Data[0]
			tt.edit(data)
			found := false
			for _, diagnostic := range validateSemanticBusinessRules(world) {
				if diagnostic.Code == tt.code {
					found = true
					if diagnostic.Subject == nil || diagnostic.Subject.Kind != "semantic-candidate-key" &&
						diagnostic.Subject.Kind != "semantic-functional-dependency" {
						t.Fatalf("diagnostic did not name typed rule: %+v", diagnostic)
					}
				}
			}
			if !found {
				t.Fatalf("%s did not reject malformed rule (%s)", tt.name, tt.code)
			}
		})
	}
}

package syntax

import (
	"reflect"
	"testing"

	"tei/internal/core"
)

const businessRuleSemanticSource = `namespace sales

data OrderLine {
    attribute orderId type std.identifier
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    candidate-key businessLine {
        field sales.order_line.order_id
        field sales.order_line.line_number
    }
    functional-dependency itemByLine {
        determinant sales.order_line.order_id
        determinant sales.order_line.line_number
        dependent sales.order_line.item_code
    }
}
`

func TestSemanticBusinessRuleParserKeepsExplicitMeaningAndOrder(t *testing.T) {
	parsed, err := NewParser(businessRuleSemanticSource).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Data) != 1 || len(parsed.Data[0].CandidateKeys) != 1 || len(parsed.Data[0].FunctionalDependencies) != 1 {
		t.Fatalf("lost business rule Data declarations: %+v", parsed.Data)
	}
	data := parsed.Data[0]
	key := data.CandidateKeys[0]
	if key.ID != "sales.order_line.business_line" || key.Symbol != "businessLine" ||
		!reflect.DeepEqual(key.Attributes, []core.CanonicalID{"sales.order_line.order_id", "sales.order_line.line_number"}) {
		t.Fatalf("candidate key was confused with physical row identity: %+v", key)
	}
	fd := data.FunctionalDependencies[0]
	if fd.ID != "sales.order_line.item_by_line" || fd.Symbol != "itemByLine" ||
		!reflect.DeepEqual(fd.Determinants, []core.CanonicalID{"sales.order_line.order_id", "sales.order_line.line_number"}) ||
		!reflect.DeepEqual(fd.Dependents, []core.CanonicalID{"sales.order_line.item_code"}) {
		t.Fatalf("FD lost determinant or dependent groups: %+v", fd)
	}
	anchored, err := ParseSemanticFileAnchored(businessRuleSemanticSource)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind, id, token string
	}{
		{"semantic-candidate-key", "sales.order_line.business_line", "businessLine"},
		{"semantic-functional-dependency", "sales.order_line.item_by_line", "itemByLine"},
	} {
		found := false
		for _, a := range anchored.Anchors {
			if a.Kind != tc.kind || a.ID != tc.id {
				continue
			}
			found = true
			if a.Anchor.Start.Offset == a.Anchor.End.Offset ||
				businessRuleSemanticSource[a.Anchor.Start.Offset:a.Anchor.End.Offset] != tc.token {
				t.Fatalf("incorrect identifier anchor: %+v", a)
			}
		}
		if !found {
			t.Fatalf("missing anchor for %s:%s", tc.kind, tc.id)
		}
	}
}

func TestSemanticBusinessRuleParserRejectsMalformedExplicitRules(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"empty key", "candidate-key businessLine {}"},
		{"duplicate key field", "candidate-key businessLine { field sales.order_line.order_id field sales.order_line.order_id }"},
		{"foreign field", "candidate-key businessLine { field customer.customer.customer_id }"},
		{"unknown token", "candidate-key businessLine { value sales.order_line.order_id }"},
		{"no key brace", "candidate-key businessLine field sales.order_line.order_id"},
		{"empty FD", "functional-dependency itemByLine {}"},
		{"missing dependent", "functional-dependency itemByLine { determinant sales.order_line.order_id }"},
		{"missing determinant", "functional-dependency itemByLine { dependent sales.order_line.item_code }"},
		{"repeated determinant", "functional-dependency itemByLine { determinant sales.order_line.order_id determinant sales.order_line.order_id dependent sales.order_line.item_code }"},
		{"repeated dependent", "functional-dependency itemByLine { determinant sales.order_line.order_id dependent sales.order_line.item_code dependent sales.order_line.item_code }"},
		{"late determinant", "functional-dependency itemByLine { dependent sales.order_line.item_code determinant sales.order_line.order_id }"},
		{"foreign determinant", "functional-dependency itemByLine { determinant customer.customer.customer_id dependent sales.order_line.item_code }"},
		{"unknown FD member", "functional-dependency itemByLine { order sales.order_line.order_id dependent sales.order_line.item_code }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "namespace sales\ndata OrderLine {\n" + tc.source + "\n}"
			if _, err := NewParser(src).ParseSemanticFile(); err == nil {
				t.Fatal("malformed canonical business rule was accepted")
			}
		})
	}
}

func TestSemanticBusinessRuleParserLeavesExistingKeylessModelUnchanged(t *testing.T) {
	old := `namespace sales
data OrderLine {
    attribute lineNumber type std.integer
}
`
	parsed, err := NewParser(old).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Data) != 1 || len(parsed.Data[0].CandidateKeys) != 0 || len(parsed.Data[0].FunctionalDependencies) != 0 {
		t.Fatalf("implicit schema-only candidate key was introduced: %+v", parsed.Data)
	}
}

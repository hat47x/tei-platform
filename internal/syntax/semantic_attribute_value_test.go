package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

const attributeValueSource = `namespace sales

data Order {
    attribute quantity type std.decimal value "2"
    attribute note type std.text value ""
    attribute status type std.code
}
`

func TestSemanticAttributeValueClauseIsThreeValued(t *testing.T) {
	ns, err := syntax.NewParser(attributeValueSource).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Data) != 1 || len(ns.Data[0].Attributes) != 3 {
		t.Fatalf("unexpected parsed data: %+v", ns.Data)
	}
	attributes := map[string]*string{}
	states := map[string]string{}
	for _, attribute := range ns.Data[0].Attributes {
		attributes[attribute.Symbol] = attribute.Value
		switch {
		case attribute.Value == nil:
			states[attribute.Symbol] = "absent"
		case *attribute.Value == "":
			states[attribute.Symbol] = "empty"
		default:
			states[attribute.Symbol] = "value"
		}
	}
	if got := states["quantity"]; got != "value" || attributes["quantity"] == nil || *attributes["quantity"] != "2" {
		t.Fatalf("quantity state=%q value=%v", got, attributes["quantity"])
	}
	if got := states["note"]; got != "empty" || attributes["note"] == nil || *attributes["note"] != "" {
		t.Fatalf("note state=%q value=%v", got, attributes["note"])
	}
	if got := states["status"]; got != "absent" || attributes["status"] != nil {
		t.Fatalf("status state=%q value=%v", got, attributes["status"])
	}
}

func TestSemanticAttributeValueRequiresAString(t *testing.T) {
	source := "namespace sales\n\ndata Order {\n    attribute quantity type std.decimal value 2\n}\n"
	if _, err := syntax.NewParser(source).ParseSemanticFile(); err == nil {
		t.Fatal("unquoted value was accepted")
	}
}

func TestAnchoredParseSkipsAttributeValueAndOtherOptions(t *testing.T) {
	source := `namespace sales

data Order {
    attribute quantity type std.decimal required immutable role key value "2"
}
`
	anchored, err := syntax.ParseSemanticFileAnchored(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Anchors) < 3 {
		t.Fatalf("anchors=%+v", anchored.Anchors)
	}
	if anchored.Value.Data[0].Attributes[0].Value == nil || *anchored.Value.Data[0].Attributes[0].Value != "2" {
		t.Fatalf("attribute value was not parsed: %+v", anchored.Value.Data[0].Attributes[0])
	}
}

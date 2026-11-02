package syntax_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticAttributeRoles(t *testing.T) {
	src := `namespace sales

data Order {
    attribute orderId type std.identifier required immutable
    attribute note type std.text
}
`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Data) != 1 || len(ns.Data[0].Attributes) != 2 {
		t.Fatalf("data=%+v", ns.Data)
	}
	if !ns.Data[0].Attributes[0].Required || !ns.Data[0].Attributes[0].Immutable {
		t.Fatalf("first attribute=%+v", ns.Data[0].Attributes[0])
	}
	if ns.Data[0].Attributes[1].Required || ns.Data[0].Attributes[1].Immutable {
		t.Fatalf("second attribute=%+v", ns.Data[0].Attributes[1])
	}

	raw, err := json.Marshal(ns.Data[0].Attributes[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "required") || strings.Contains(string(raw), "immutable") {
		t.Fatalf("role flags changed existing attribute JSON: %s", raw)
	}
}

func TestParseSemanticAttributeRolesRejectUnknownOption(t *testing.T) {
	src := `namespace sales

data Order {
    attribute orderId type std.identifier mandatory
}
`
	if _, err := syntax.NewParser(src).ParseSemanticFile(); err == nil {
		t.Fatal("unknown attribute option was accepted")
	}
}

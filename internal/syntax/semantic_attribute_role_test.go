package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticAttributeRole(t *testing.T) {
	src := `namespace sales

data Order {
    attribute orderId type std.identifier role business-key
}`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespace.Data) != 1 || len(namespace.Data[0].Attributes) != 1 {
		t.Fatalf("namespace=%+v", namespace)
	}
	if role := namespace.Data[0].Attributes[0].Role; role != "business-key" {
		t.Fatalf("role=%s", role)
	}
}

package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseRelationOnDelete(t *testing.T) {
	src := `namespace sales

data Order {
    attribute orderId type std.identifier
    relation customer to customer.customer one on-delete restrict
}`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespace.Data) != 1 || len(namespace.Data[0].Relations) != 1 {
		t.Fatalf("data=%+v", namespace.Data)
	}
	if namespace.Data[0].Relations[0].OnDelete != "restrict" {
		t.Fatalf("on-delete=%s", namespace.Data[0].Relations[0].OnDelete)
	}
}

package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseRecordPrintTemplate(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    print-template a4-v1
    interaction {
        item sales.order.order_id
    }
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.PrintTemplate != "a4-v1" {
		t.Fatalf("print-template=%s", got.PrintTemplate)
	}
}

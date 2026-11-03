package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseRecordApplicationPatternLayout(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
        item sales.order.customer_id
    }
    layout {
        order sales.order.customer_id
        order sales.order.order_id
    }
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.Layout == nil || len(got.Layout.Order) != 2 || got.Layout.Order[0] != "sales.order.customer_id" {
		t.Fatalf("layout=%#v", got.Layout)
	}
}

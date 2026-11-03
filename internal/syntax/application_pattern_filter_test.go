package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseBrowseApplicationPatternFilters(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    filter sales.order.status { eq ne }
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Filters) != 1 || got.Filters[0].Attribute != "sales.order.status" {
		t.Fatalf("filters=%#v", got.Filters)
	}
	if len(got.Filters[0].Operators) != 2 || got.Filters[0].Operators[0] != "eq" || got.Filters[0].Operators[1] != "ne" {
		t.Fatalf("operators=%#v", got.Filters[0].Operators)
	}
}

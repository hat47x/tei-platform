package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseBrowseApplicationPatternSorts(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    sort sales.order.order_id asc
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sorts) != 1 || got.Sorts[0].Attribute != "sales.order.order_id" || got.Sorts[0].Direction != "asc" {
		t.Fatalf("sorts=%#v", got.Sorts)
	}
}

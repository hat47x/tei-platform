package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseBrowseApplicationPatternPageSize(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    page-size 20
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.PageSize != 20 {
		t.Fatalf("page-size=%d", got.PageSize)
	}
}

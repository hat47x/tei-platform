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

func TestParseBrowseApplicationPatternQueryParameters(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    filter sales.order.status { eq ne } parameter status operator-parameter status_operator
    page-size 20 parameter page_size offset page_offset
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Filters) != 1 || got.Filters[0].Parameter != "status" || got.Filters[0].OperatorParameter != "status_operator" {
		t.Fatalf("filter=%+v", got.Filters)
	}
	if got.PageSize != 20 || got.PageSizeParameter != "page_size" || got.PageOffsetParameter != "page_offset" {
		t.Fatalf("page=%d size=%q offset=%q", got.PageSize, got.PageSizeParameter, got.PageOffsetParameter)
	}
}

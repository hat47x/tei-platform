package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseQueryAuthoringInputs(t *testing.T) {
	src := `query order_entry.recent_orders {
    input status std.code
    datasource sales
    sql postgres file "recent_orders.sql"
}`
	got, err := syntax.NewParser(src).ParseQueryAuthoring("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Value.Inputs) != 1 || got.Value.Inputs[0].Name != "status" || got.Value.Inputs[0].Type != "std.code" {
		t.Fatalf("inputs=%#v", got.Value.Inputs)
	}
}

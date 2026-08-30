package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseQueryAuthoringKeepsEmbeddedSQLCompatible(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres {
        SELECT order_id FROM public.orders
    }
}`
	got, err := syntax.NewParser(src).ParseQueryAuthoring("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if got.NativeSQLPath != "" {
		t.Fatalf("native path=%q", got.NativeSQLPath)
	}
	if got.Value.ID != "order_entry.recent_orders" || got.Value.Application != "order_entry" {
		t.Fatalf("query identity=%#v", got.Value)
	}
	if got.Value.Datasource != "sales" || got.Value.Dialect != "postgres" {
		t.Fatalf("query metadata=%#v", got.Value)
	}
	if got.Value.Source != "SELECT order_id FROM public.orders" {
		t.Fatalf("source=%q", got.Value.Source)
	}
}

func TestParseQueryAuthoringKeepsNativePathOutsideQueryValue(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}`
	got, err := syntax.NewParser(src).ParseQueryAuthoring("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if got.NativeSQLPath != "recent_orders.sql" {
		t.Fatalf("native path=%q", got.NativeSQLPath)
	}
	if got.Value.Source != "" {
		t.Fatalf("native path leaked into query source=%q", got.Value.Source)
	}
	if got.Value.ID != "order_entry.recent_orders" || got.Value.Dialect != "postgres" || got.Value.Datasource != "sales" {
		t.Fatalf("query metadata=%#v", got.Value)
	}
}

func TestParseQueryAuthoringRejectsEmptyNativePath(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres file ""
}`
	if _, err := syntax.NewParser(src).ParseQueryAuthoring("order_entry"); err == nil {
		t.Fatal("empty SQL file path was accepted")
	}
}

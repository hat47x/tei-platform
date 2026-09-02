package syntax_test

import (
	"reflect"
	"strings"
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

	legacy, err := syntax.NewParser(src).ParseQuery("order_entry")
	if err != nil {
		t.Fatalf("existing ParseQuery rejected embedded SQL: %v", err)
	}
	if !reflect.DeepEqual(legacy, got.Value) {
		t.Fatalf("existing ParseQuery changed value: legacy=%#v authoring=%#v", legacy, got.Value)
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

	if _, err := syntax.NewParser(src).ParseQuery("order_entry"); err == nil || !strings.Contains(err.Error(), "Workspace loader") {
		t.Fatalf("existing ParseQuery should require Workspace resolution for native SQL, err=%v", err)
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

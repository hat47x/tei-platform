package syntax_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/syntax"
)

func TestParseBrowseApplicationPattern(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse pattern: %v", err)
	}
	if got.ID != "order_entry.recent_orders" || got.Kind != application.PatternBrowse {
		t.Fatalf("unexpected pattern: %#v", got)
	}
	if got.Data != "sales.order" || got.Query != "order_entry.recent_orders" {
		t.Fatalf("unexpected references: %#v", got)
	}
	if got.Datasource != "" {
		t.Fatalf("datasource should remain optional, got %q", got.Datasource)
	}
}

func TestParseBrowseApplicationPatternAcceptsExplicitDatasource(t *testing.T) {
	src := `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    datasource sales
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse pattern: %v", err)
	}
	if got.Datasource != "sales" {
		t.Fatalf("datasource=%q", got.Datasource)
	}
}

func TestParseApplicationPatternRejectsUnsupportedKind(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    query order_entry.recent_orders
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("unsupported Pattern Kind must be rejected until its Reference is implemented")
	}
}

func TestParseBrowseApplicationPatternRequiresDataAndQuery(t *testing.T) {
	for name, src := range map[string]string{
		"data": `pattern order_entry.recent_orders browse {
    query order_entry.recent_orders
}`,
		"query": `pattern order_entry.recent_orders browse {
    data sales.order
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("browse Pattern without %s must be rejected", name)
			}
		})
	}
}

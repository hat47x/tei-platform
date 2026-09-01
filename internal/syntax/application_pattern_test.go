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

func TestParseRecordApplicationPatternUsesDataOnly(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse record pattern: %v", err)
	}
	if got.ID != "order_entry.order_record" || got.Kind != application.PatternRecord {
		t.Fatalf("unexpected record pattern: %#v", got)
	}
	if got.Data != "sales.order" {
		t.Fatalf("record data=%q", got.Data)
	}
	if got.Query != "" || got.Datasource != "" {
		t.Fatalf("record should not acquire query or datasource: %#v", got)
	}
}

func TestParseRecordApplicationPatternRequiresData(t *testing.T) {
	src := `pattern order_entry.order_record record {
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("record Pattern without data must be rejected")
	}
}

func TestParseRecordApplicationPatternRejectsBrowseOnlyFields(t *testing.T) {
	for name, src := range map[string]string{
		"query": `pattern order_entry.order_record record {
    data sales.order
    query order_entry.recent_orders
}`,
		"datasource": `pattern order_entry.order_record record {
    data sales.order
    datasource sales
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("record Pattern with %s must be rejected until the Reference requires it", name)
			}
		})
	}
}

func TestParseApplicationPatternRejectsUnsupportedKind(t *testing.T) {
	src := `pattern order_entry.customer_reference reference {
    data sales.order
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

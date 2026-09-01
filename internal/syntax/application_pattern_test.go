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
	if got.Relation != "" || got.Capability != "" || got.Datasource != "" {
		t.Fatalf("browse should not acquire unrelated references: %#v", got)
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

func TestParseBrowseApplicationPatternRejectsUnrelatedFields(t *testing.T) {
	for name, src := range map[string]string{
		"relation": `pattern order_entry.recent_orders browse {
    data sales.order
    relation sales.order.customer
    query order_entry.recent_orders
}`,
		"capability": `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    capability sales.order.submit
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("browse Pattern with %s must be rejected", name)
			}
		})
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
	if got.Relation != "" || got.Query != "" || got.Capability != "" || got.Datasource != "" {
		t.Fatalf("record should not acquire unrelated references: %#v", got)
	}
}

func TestParseRecordApplicationPatternRequiresData(t *testing.T) {
	src := `pattern order_entry.order_record record {
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("record Pattern without data must be rejected")
	}
}

func TestParseRecordApplicationPatternRejectsOtherFields(t *testing.T) {
	for name, src := range map[string]string{
		"relation": `pattern order_entry.order_record record {
    data sales.order
    relation sales.order.customer
}`,
		"query": `pattern order_entry.order_record record {
    data sales.order
    query order_entry.recent_orders
}`,
		"capability": `pattern order_entry.order_record record {
    data sales.order
    capability sales.order.submit
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

func TestParseReferenceApplicationPatternUsesRelationAndDatasource(t *testing.T) {
	src := `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse reference pattern: %v", err)
	}
	if got.ID != "order_entry.customer_reference" || got.Kind != application.PatternReference {
		t.Fatalf("unexpected reference pattern: %#v", got)
	}
	if got.Relation != "sales.order.customer" || got.Datasource != "master" {
		t.Fatalf("unexpected reference links: %#v", got)
	}
	if got.Data != "" || got.Query != "" || got.Capability != "" {
		t.Fatalf("reference must not duplicate relation meaning through other fields: %#v", got)
	}
}

func TestParseReferenceApplicationPatternRequiresRelationAndDatasource(t *testing.T) {
	for name, src := range map[string]string{
		"relation": `pattern order_entry.customer_reference reference {
    datasource master
}`,
		"datasource": `pattern order_entry.customer_reference reference {
    relation sales.order.customer
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("reference Pattern without %s must be rejected", name)
			}
		})
	}
}

func TestParseReferenceApplicationPatternRejectsDuplicatedMeaning(t *testing.T) {
	for name, src := range map[string]string{
		"data": `pattern order_entry.customer_reference reference {
    data sales.order
    relation sales.order.customer
    datasource master
}`,
		"query": `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    query order_entry.recent_orders
    datasource master
}`,
		"capability": `pattern order_entry.customer_reference reference {
    relation sales.order.customer
    capability sales.order.submit
    datasource master
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("reference Pattern with %s must be rejected", name)
			}
		})
	}
}

func TestParseCommandApplicationPatternUsesCapabilityOnly(t *testing.T) {
	src := `pattern order_entry.submit_order command {
    capability sales.order.submit
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse command pattern: %v", err)
	}
	if got.ID != "order_entry.submit_order" || got.Kind != application.PatternCommand {
		t.Fatalf("unexpected command pattern: %#v", got)
	}
	if got.Capability != "sales.order.submit" {
		t.Fatalf("command capability=%q", got.Capability)
	}
	if got.Data != "" || got.Relation != "" || got.Query != "" || got.Datasource != "" {
		t.Fatalf("command should keep Capability identity only: %#v", got)
	}
}

func TestParseCommandApplicationPatternRequiresCapability(t *testing.T) {
	src := `pattern order_entry.submit_order command {
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("command Pattern without capability must be rejected")
	}
}

func TestParseCommandApplicationPatternRejectsOtherFields(t *testing.T) {
	for name, src := range map[string]string{
		"data": `pattern order_entry.submit_order command {
    data sales.order
    capability sales.order.submit
}`,
		"relation": `pattern order_entry.submit_order command {
    relation sales.order.customer
    capability sales.order.submit
}`,
		"query": `pattern order_entry.submit_order command {
    query order_entry.recent_orders
    capability sales.order.submit
}`,
		"datasource": `pattern order_entry.submit_order command {
    capability sales.order.submit
    datasource sales
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
				t.Fatalf("command Pattern with %s must be rejected until the Reference requires it", name)
			}
		})
	}
}

func TestParseApplicationPatternRejectsUnsupportedKind(t *testing.T) {
	src := `pattern order_entry.order_lines header-detail {
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

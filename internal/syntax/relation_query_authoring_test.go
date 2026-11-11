package syntax

import (
	"testing"

	"tei/internal/core"
	"tei/internal/query"
)

func TestParseRelationQueryAcceptsMinimalQuery(t *testing.T) {
	src := `
relation-query order_entry.order_with_customer {
    datasource master
    from sales.order
    relation sales.order.customer
    result {
        column order_id -> sales.order.order_id
        column name -> customer.customer.name
    }
}
`
	got, err := NewParser(src).ParseRelationQuery(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got.ID != "order_entry.order_with_customer" {
		t.Fatalf("unexpected id: %s", got.ID)
	}
	if got.Datasource != "master" {
		t.Fatalf("unexpected datasource: %s", got.Datasource)
	}
	if got.Root != "sales.order" {
		t.Fatalf("unexpected root: %s", got.Root)
	}
	if len(got.Relations) != 1 || got.Relations[0].Relation != "sales.order.customer" {
		t.Fatalf("unexpected relations: %+v", got.Relations)
	}
	if len(got.ResultColumns) != 2 {
		t.Fatalf("expected 2 result columns, got %d", len(got.ResultColumns))
	}
}

func TestParseRelationQueryRejectsMissingFrom(t *testing.T) {
	src := `
relation-query order_entry.bad {
    relation sales.order.customer
    result { column order_id -> sales.order.order_id }
}
`
	if _, err := NewParser(src).ParseRelationQuery(core.MustCanonicalID("order_entry")); err == nil {
		t.Fatal("expected error for missing from clause")
	}
}

func TestParseRelationQueryRejectsMissingRelation(t *testing.T) {
	src := `
relation-query order_entry.bad {
    from sales.order
    result { column order_id -> sales.order.order_id }
}
`
	if _, err := NewParser(src).ParseRelationQuery(core.MustCanonicalID("order_entry")); err == nil {
		t.Fatal("expected error for missing relation")
	}
}

func TestParseRelationQueryAcceptsFetchAndBatch(t *testing.T) {
	src := `
relation-query order_entry.order_with_customer {
    datasource master
    from sales.order
    relation sales.order.customer
    fetch batch
    batch 100
    result {
        column order_id -> sales.order.order_id
        column name -> customer.customer.name
    }
}
`
	got, err := NewParser(src).ParseRelationQuery(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got.FetchMode != query.RelationFetchModeBatch {
		t.Fatalf("unexpected fetch mode: %s", got.FetchMode)
	}
	if got.BatchSize != 100 {
		t.Fatalf("unexpected batch size: %d", got.BatchSize)
	}
}

func TestParseRelationQueryAcceptsTemporalFilter(t *testing.T) {
	src := `
relation-query order_entry.order_with_customer_temporal {
    datasource master
    from sales.order
    relation sales.order.customer
    filter {
        status = 'active' AS OF '2024-01-01'
    }
    result {
        column order_id -> sales.order.order_id
        column name -> customer.customer.name
    }
}
`
	got, err := NewParser(src).ParseRelationQuery(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got.Filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(got.Filters))
	}
	if got.Filters[0].Temporal == nil {
		t.Fatal("expected temporal filter to be parsed")
	}
	if got.Filters[0].Temporal.ValidAt != "2024-01-01" {
		t.Fatalf("unexpected validAt: %s", got.Filters[0].Temporal.ValidAt)
	}
}

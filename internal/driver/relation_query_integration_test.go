package driver

import (
	"testing"

	"tei/internal/database/persistence"
	postgressqlgen "tei/internal/database/sqlgen/postgres"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestRelationQuerySQLMatchesExplicitSQL(t *testing.T) {
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	mappings := []persistence.Mapping{
		{
			SemanticData: "sales.order",
			Table:        "orders",
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "id"},
				{Attribute: "sales.order.customer_id", Column: "customer_id"},
			},
		},
		{
			SemanticData: "customer.customer",
			Table:        "customers",
			Columns: []persistence.ColumnMapping{
				{Attribute: "customer.customer.customer_id", Column: "id"},
				{Attribute: "customer.customer.name", Column: "name"},
			},
		},
	}

	builder := RelationQuerySQL{
		Dialect:  postgressqlgen.NewDialect(),
		Resolved: resolved,
		Mappings: mappings,
	}
	relationSQL, err := builder.BuildSelect()
	if err != nil {
		t.Fatalf("BuildSelect error: %v", err)
	}

	expected := "SELECT orders.id, customers.name FROM orders LEFT JOIN customers AS rel_0 ON orders.customer_id = $1"
	if relationSQL.Statement != expected {
		t.Errorf("Relation Query SQL does not match equivalent explicit SQL:\ngot:  %s\nwant: %s", relationSQL.Statement, expected)
	}
}

func TestMapRelationQueryMapsRawResult(t *testing.T) {
	resolved := query.ResolvedRelationQuery{
		Query: query.RelationQuery{
			ID:          "order_entry.order_with_customer",
			Application: "order_entry",
			Datasource:  "master",
			Root:        "sales.order",
			Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
			ResultColumns: []query.ResultColumnMapping{
				{Column: "order_id", Attribute: "sales.order.order_id"},
				{Column: "name", Attribute: "customer.customer.name"},
			},
		},
		RootData: semantic.Data{ID: "sales.order"},
		Steps: []query.ResolvedRelationStep{
			{
				Relation: semantic.Relation{ID: "sales.order.customer"},
				Source:   semantic.Data{ID: "sales.order"},
				Target:   semantic.Data{ID: "customer.customer"},
			},
		},
	}
	raw := queryresult.RawResult{
		Columns: []string{"order_id", "name"},
		Rows: [][]any{
			{1, "Alice"},
			{2, "Bob"},
		},
	}
	mapped, err := queryresult.MapRelationQuery(resolved, raw)
	if err != nil {
		t.Fatalf("MapRelationQuery error: %v", err)
	}
	if len(mapped.Columns) != 2 {
		t.Fatalf("expected 2 mapped columns, got %d", len(mapped.Columns))
	}
	if mapped.Columns[0].Attribute != "sales.order.order_id" {
		t.Errorf("unexpected attribute: %s", mapped.Columns[0].Attribute)
	}
	if mapped.Columns[1].Attribute != "customer.customer.name" {
		t.Errorf("unexpected attribute: %s", mapped.Columns[1].Attribute)
	}
	if len(mapped.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(mapped.Rows))
	}
}

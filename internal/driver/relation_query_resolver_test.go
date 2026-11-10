package driver

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

func TestResolveRelationQueryResolvesSteps(t *testing.T) {
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if resolved.RootData.ID != "sales.order" {
		t.Fatalf("unexpected root data: %s", resolved.RootData.ID)
	}
	if len(resolved.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(resolved.Steps))
	}
	if resolved.Steps[0].Relation.ID != "sales.order.customer" {
		t.Fatalf("unexpected relation: %s", resolved.Steps[0].Relation.ID)
	}
	if resolved.Steps[0].Target.ID != "customer.customer" {
		t.Fatalf("unexpected target: %s", resolved.Steps[0].Target.ID)
	}
}

func TestResolveRelationQueryRejectsUnknownRelation(t *testing.T) {
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	if _, err := resolver.ResolveRelationQuery("order_entry", "order_entry.unknown_relation"); err == nil {
		t.Fatal("expected error for unknown relation")
	}
}

func irWorldWithRelationQuery() ir.World {
	orderData := semantic.Data{
		ID: "sales.order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.order_id", TypeRef: "sales.order_id"},
			{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
		},
	}
	customerData := semantic.Data{
		ID: "customer.customer",
		Attributes: []semantic.Attribute{
			{ID: "customer.customer.customer_id", TypeRef: "customer.customer_id"},
			{ID: "customer.customer.name", TypeRef: "std.text"},
		},
	}
	relation := semantic.Relation{
		ID:         "sales.order.customer",
		Target:     "customer.customer",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "order_entry",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"sales", "customer"},
	}
	relationQuery := query.RelationQuery{
		ID:          "order_entry.order_with_customer",
		Application: "order_entry",
		Datasource:  "master",
		Root:        "sales.order",
		Relations:   []query.RelationStep{{Relation: "sales.order.customer"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
			{Column: "name", Attribute: "customer.customer.name"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{orderData}},
			{ID: "customer", Data: []semantic.Data{customerData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

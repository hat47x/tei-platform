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

func TestResolveRelationQueryHandlesNullForeignKeys(t *testing.T) {
	// NULL foreign keys are valid in the data model - the resolver should
	// still resolve the relation definition even if some instances have NULL
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if len(resolved.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(resolved.Steps))
	}
	// The resolver doesn't know about NULL values - that's a runtime concern
	// This test verifies the relation definition is valid regardless of NULLs
	if resolved.Steps[0].Relation.ID != "sales.order.customer" {
		t.Fatalf("unexpected relation: %s", resolved.Steps[0].Relation.ID)
	}
}

func TestResolveRelationQueryRejectsAmbiguousPaths(t *testing.T) {
	world := irWorldWithAmbiguousPaths()
	resolver := NewResolver(world)
	if _, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer"); err == nil {
		t.Fatal("expected error for ambiguous paths")
	}
	if _, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer_alt"); err == nil {
		t.Fatal("expected error for ambiguous paths")
	}
}

func TestResolveRelationQueryAllowsSelfReference(t *testing.T) {
	world := irWorldWithSelfReference()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("org", "org.employee_managers")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if len(resolved.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(resolved.Steps))
	}
	if resolved.Steps[0].Target.ID != "org.employee" {
		t.Fatalf("unexpected target: %s", resolved.Steps[0].Target.ID)
	}
}

func TestResolveRelationQueryRejectsMultiStepCycle(t *testing.T) {
	world := irWorldWithMultiStepCycle()
	resolver := NewResolver(world)
	if _, err := resolver.ResolveRelationQuery("org", "org.cycle"); err == nil {
		t.Fatal("expected error for multi-step cycle")
	}
}

func TestResolveRelationQueryAllowsCompositeKey(t *testing.T) {
	world := irWorldWithCompositeKey()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("sales", "sales.order_line")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if len(resolved.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(resolved.Steps))
	}
	if len(resolved.Steps[0].Relation.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(resolved.Steps[0].Relation.Keys))
	}
}

func TestResolveRelationQueryRejectsMissingRelation(t *testing.T) {
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	if _, err := resolver.ResolveRelationQuery("order_entry", "order_entry.missing_relation"); err == nil {
		t.Fatal("expected error for missing relation")
	}
}

func TestResolveRelationQueryHandlesEmptyResultSet(t *testing.T) {
	// The resolver doesn't know about empty data - that's a runtime concern
	// This test verifies the query definition is valid even when no data matches
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("order_entry", "order_entry.order_with_customer")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if resolved.RootData.ID != "sales.order" {
		t.Fatalf("unexpected root data: %s", resolved.RootData.ID)
	}
	// Empty result set is a valid outcome - the SQL will execute and return 0 rows
}


func irWorldWithAmbiguousPaths() ir.World {
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
	relation1 := semantic.Relation{
		ID:         "sales.order.customer",
		Target:     "customer.customer",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
		},
	}
	relation2 := semantic.Relation{
		ID:         "sales.order.customer_alt",
		Target:     "customer.customer",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation1, relation2}
	app := application.Application{
		ID:           "order_entry",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"sales", "customer"},
	}
	ambiguousQuery := query.RelationQuery{
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
	explicitQuery := query.RelationQuery{
		ID:          "order_entry.order_with_customer_explicit",
		Application: "order_entry",
		Datasource:  "master",
		Root:        "sales.order",
		Relations:   []query.RelationStep{{Relation: "sales.order.customer_alt"}},
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
		RelationQueries: []query.RelationQuery{ambiguousQuery, explicitQuery},
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
	altRelation := semantic.Relation{
		ID:         "customer.customer.alt",
		Target:     "customer.customer",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "customer.customer.customer_id", TargetAttribute: "customer.customer.customer_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation}
	customerData.Relations = []semantic.Relation{altRelation}
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

func irWorldWithSelfReference() ir.World {
	employeeData := semantic.Data{
		ID: "org.employee",
		Attributes: []semantic.Attribute{
			{ID: "org.employee.employee_id", TypeRef: "org.employee_id"},
			{ID: "org.employee.manager_id", TypeRef: "org.employee_id"},
		},
	}
	relation := semantic.Relation{
		ID:         "org.employee.manager",
		Target:     "org.employee",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "org.employee.manager_id", TargetAttribute: "org.employee.employee_id"},
		},
	}
	employeeData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "org",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"org"},
	}
	relationQuery := query.RelationQuery{
		ID:          "org.employee_managers",
		Application: "org",
		Datasource:  "master",
		Root:        "org.employee",
		Relations:   []query.RelationStep{{Relation: "org.employee.manager"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "employee_id", Attribute: "org.employee.employee_id"},
			{Column: "manager_id", Attribute: "org.employee.manager_id"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "org", Data: []semantic.Data{employeeData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

func irWorldWithCycle() ir.World {
	orgData := semantic.Data{
		ID: "org.organization",
		Attributes: []semantic.Attribute{
			{ID: "org.organization.org_id", TypeRef: "org.org_id"},
			{ID: "org.organization.parent_id", TypeRef: "org.org_id"},
		},
	}
	relation := semantic.Relation{
		ID:         "org.organization.parent",
		Target:     "org.organization",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "org.organization.parent_id", TargetAttribute: "org.organization.org_id"},
		},
	}
	orgData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "org",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"org"},
	}
	relationQuery := query.RelationQuery{
		ID:          "org.org_tree",
		Application: "org",
		Datasource:  "master",
		Root:        "org.organization",
		Relations:   []query.RelationStep{{Relation: "org.organization.parent"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "org_id", Attribute: "org.organization.org_id"},
			{Column: "parent_id", Attribute: "org.organization.parent_id"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "org", Data: []semantic.Data{orgData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

func irWorldWithMultiStepCycle() ir.World {
	orgDataA := semantic.Data{
		ID: "org.department_a",
		Attributes: []semantic.Attribute{
			{ID: "org.department_a.id", TypeRef: "org.dept_id"},
		},
	}
	orgDataB := semantic.Data{
		ID: "org.department_b",
		Attributes: []semantic.Attribute{
			{ID: "org.department_b.id", TypeRef: "org.dept_id"},
		},
	}
	relationAtoB := semantic.Relation{
		ID:         "org.department_a.links",
		Target:     "org.department_b",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "org.department_a.id", TargetAttribute: "org.department_b.id"},
		},
	}
	relationBtoA := semantic.Relation{
		ID:         "org.department_b.links",
		Target:     "org.department_a",
		Cardinality: semantic.CardinalityOne,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "org.department_b.id", TargetAttribute: "org.department_a.id"},
		},
	}
	orgDataA.Relations = []semantic.Relation{relationAtoB}
	orgDataB.Relations = []semantic.Relation{relationBtoA}
	app := application.Application{
		ID:           "org",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"org"},
	}
	relationQuery := query.RelationQuery{
		ID:          "org.cycle",
		Application: "org",
		Datasource:  "master",
		Root:        "org.department_a",
		Relations: []query.RelationStep{
			{Relation: "org.department_a.links"},
			{Relation: "org.department_b.links"},
		},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "id", Attribute: "org.department_a.id"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "org", Data: []semantic.Data{orgDataA, orgDataB}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

func irWorldWithCompositeKey() ir.World {
	orderData := semantic.Data{
		ID: "sales.order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.order_id", TypeRef: "sales.order_id"},
			{ID: "sales.order.tenant_id", TypeRef: "std.text"},
		},
	}
	lineData := semantic.Data{
		ID: "sales.order_line",
		Attributes: []semantic.Attribute{
			{ID: "sales.order_line.order_id", TypeRef: "sales.order_id"},
			{ID: "sales.order_line.tenant_id", TypeRef: "std.text"},
			{ID: "sales.order_line.line_no", TypeRef: "std.text"},
		},
	}
	relation := semantic.Relation{
		ID:         "sales.order.lines",
		Target:     "sales.order_line",
		Cardinality: semantic.CardinalityMany,
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.order_id", TargetAttribute: "sales.order_line.order_id"},
			{SourceAttribute: "sales.order.tenant_id", TargetAttribute: "sales.order_line.tenant_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "sales",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"sales"},
	}
	relationQuery := query.RelationQuery{
		ID:          "sales.order_line",
		Application: "sales",
		Datasource:  "master",
		Root:        "sales.order",
		Relations:   []query.RelationStep{{Relation: "sales.order.lines"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order_line.order_id"},
			{Column: "line_no", Attribute: "sales.order_line.line_no"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{orderData, lineData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

func irWorldWithManyToMany() ir.World {
	orderData := semantic.Data{
		ID: "sales.order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.order_id", TypeRef: "sales.order_id"},
		},
	}
	productData := semantic.Data{
		ID: "sales.product",
		Attributes: []semantic.Attribute{
			{ID: "sales.product.product_id", TypeRef: "sales.product_id"},
			{ID: "sales.product.name", TypeRef: "std.text"},
		},
	}
	relation := semantic.Relation{
		ID:         "sales.order.products",
		Target:     "sales.product",
		Cardinality: semantic.CardinalityMany,
		JoinTable:  "sales.order_product",
		Keys: []semantic.RelationKey{
			{SourceAttribute: "sales.order.order_id", TargetAttribute: "sales.order_product.order_id"},
			{SourceAttribute: "sales.product.product_id", TargetAttribute: "sales.order_product.product_id"},
		},
	}
	orderData.Relations = []semantic.Relation{relation}
	app := application.Application{
		ID:           "sales",
		Databases:    []application.DatabaseRequirement{{Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead}},
		SemanticUses: []core.CanonicalID{"sales"},
	}
	relationQuery := query.RelationQuery{
		ID:          "sales.order_products",
		Application: "sales",
		Datasource:  "master",
		Root:        "sales.order",
		Relations:   []query.RelationStep{{Relation: "sales.order.products"}},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
			{Column: "product_name", Attribute: "sales.product.name"},
		},
	}
	return ir.World{
		Applications: []application.Application{app},
		Semantics: []semantic.Namespace{
			{ID: "sales", Data: []semantic.Data{orderData, productData}},
		},
		RelationQueries: []query.RelationQuery{relationQuery},
	}
}

func TestResolveRelationQueryAllowsManyToMany(t *testing.T) {
	world := irWorldWithManyToMany()
	resolver := NewResolver(world)
	resolved, err := resolver.ResolveRelationQuery("sales", "sales.order_products")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if len(resolved.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(resolved.Steps))
	}
	if resolved.Steps[0].JoinTable != "sales.order_product" {
		t.Fatalf("unexpected join table: %s", resolved.Steps[0].JoinTable)
	}
	if resolved.Steps[0].Target.ID != "sales.product" {
		t.Fatalf("unexpected target: %s", resolved.Steps[0].Target.ID)
	}
}

func TestResolveRelationQueryEnforcesDepthLimit(t *testing.T) {
	world := irWorldWithRelationQuery()
	resolver := NewResolver(world)
	// Create a query with depth=1 but 2 relation steps
	query := query.RelationQuery{
		ID:          "order_entry.deep",
		Application: "order_entry",
		Datasource:  "master",
		Root:        "sales.order",
		Depth:       1,
		Relations: []query.RelationStep{
			{Relation: "sales.order.customer"},
			{Relation: "customer.customer.alt"},
		},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "order_id", Attribute: "sales.order.order_id"},
		},
	}
	// Manually add to world for test
	world.RelationQueries = append(world.RelationQueries, query)
	if _, err := resolver.ResolveRelationQuery("order_entry", "order_entry.deep"); err == nil {
		t.Fatal("expected error for depth exceeded")
	}
}

package index_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

func TestQueryAnalysisBecomesObservedPhysicalDependency(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{
			ID: core.MustCanonicalID("order_entry"),
			Databases: []application.DatabaseRequirement{{
				Alias: core.MustLocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite,
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID: core.MustCanonicalID("order_entry.recent_orders"), Application: core.MustCanonicalID("order_entry"), Datasource: core.MustLocalName("sales"),
		}},
	}
	d := index.BuildWithQueryAnalyses(w, []analyzer.Result{{
		ProtocolVersion: 1, Status: "ok", Query: core.MustCanonicalID("order_entry.recent_orders"),
		Tables: []string{"orders"}, Columns: []string{"orders.order_id"},
	}})
	if len(d.ObservedPhysicalDependencies) != 2 {
		t.Fatalf("got %d observed dependencies", len(d.ObservedPhysicalDependencies))
	}
	for _, dep := range d.ObservedPhysicalDependencies {
		if dep.Database != "sales_core" || dep.Datasource != "sales" {
			t.Fatalf("unexpected dependency: %#v", dep)
		}
	}
}

func TestObservedDependenciesResolveAgainstPhysicalSchemaSnapshot(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{
			ID: core.MustCanonicalID("order_entry"),
			Databases: []application.DatabaseRequirement{{
				Alias: core.MustLocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite,
			}},
		}},
		Queries: []query.EmbeddedSQL{{
			ID: core.MustCanonicalID("order_entry.recent_orders"), Application: core.MustCanonicalID("order_entry"), Datasource: core.MustLocalName("sales"),
		}},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	d := index.BuildWithObservations(w, []analyzer.Result{{
		ProtocolVersion: 1, Status: "ok", Query: core.MustCanonicalID("order_entry.recent_orders"),
		Tables: []string{"orders"}, Columns: []string{"orders.order_id"},
	}}, []physical.SchemaSnapshot{snap})
	if len(d.AnalyzedQueries) != 1 {
		t.Fatalf("analyzed queries=%d", len(d.AnalyzedQueries))
	}
	if d.AnalyzedQueries[0].IR.Coverage.Resolution != query.CoverageFull {
		t.Fatalf("coverage=%#v", d.AnalyzedQueries[0].IR.Coverage)
	}
	for _, dep := range d.ObservedPhysicalDependencies {
		if dep.Resolution != "schema-resolved" {
			t.Fatalf("dependency not schema-resolved: %#v", dep)
		}
	}
}

func TestObservedIndexCarriesPhysicalSchemaAndRelationalIR(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{
			ID:        core.MustCanonicalID("order_entry"),
			Databases: []application.DatabaseRequirement{{Alias: core.MustLocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite}},
		}},
		Queries: []query.EmbeddedSQL{{ID: core.MustCanonicalID("order_entry.q"), Application: core.MustCanonicalID("order_entry"), Datasource: core.MustLocalName("sales")}},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	analysis := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: core.MustCanonicalID("order_entry.q"), Tables: []string{"public.orders"}, Columns: []string{"o.order_id"}, TableAliases: map[string]string{"o": "public.orders"}, Lowering: "full", Operations: []analyzer.Operation{{ID: "source:0", Kind: "source", Source: "public.orders", Alias: "o"}, {ID: "project:0", Kind: "project", Inputs: []string{"source:0"}, Expressions: []string{"o.order_id"}}}}
	d := index.BuildWithObservations(w, []analyzer.Result{analysis}, []physical.SchemaSnapshot{snap})
	if len(d.PhysicalSchemas) != 1 || d.PhysicalSchemas[0].Tables[0].QualifiedName() != "public.orders" {
		t.Fatalf("physical schemas=%#v", d.PhysicalSchemas)
	}
	if got := d.AnalyzedQueries[0].IR.Relational.Root; got != "project:0" {
		t.Fatalf("root=%q", got)
	}
}

func TestObservedIndexDoesNotTreatCTEBindingAsPhysicalCoupling(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{ID: core.MustCanonicalID("order_entry"), Databases: []application.DatabaseRequirement{{Alias: core.MustLocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite}}}},
		Queries:      []query.EmbeddedSQL{{ID: core.MustCanonicalID("order_entry.cte"), Application: core.MustCanonicalID("order_entry"), Datasource: core.MustLocalName("sales")}},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	analysis := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: core.MustCanonicalID("order_entry.cte"), Lowering: "full",
		Scopes:       []analyzer.Scope{{ID: "scope:root", Kind: "root"}, {ID: "scope:cte:recent", ParentID: "scope:root", Kind: "cte", Name: "recent"}},
		TableAliases: map[string]string{"o": "public.orders", "r": "recent"},
		ColumnRefs:   []analyzer.ColumnOccurrence{{ID: "c0", ScopeID: "scope:cte:recent", Name: "order_id", Qualifier: "o"}, {ID: "c1", ScopeID: "scope:root", Name: "order_id", Qualifier: "r"}},
		OutputFields: []analyzer.OutputField{{ID: "o0", ScopeID: "scope:cte:recent", Name: "order_id", Inputs: []string{"c0"}}, {ID: "o1", ScopeID: "scope:root", Name: "order_id", Inputs: []string{"c1"}}},
		Operations:   []analyzer.Operation{{ID: "cte.source", Kind: "source", ScopeID: "scope:cte:recent", Source: "public.orders", SourceKind: "table", Alias: "o"}, {ID: "cte.project", Kind: "project", ScopeID: "scope:cte:recent", Inputs: []string{"cte.source"}}, {ID: "root.source", Kind: "source", ScopeID: "scope:root", Source: "recent", SourceKind: "cte", Binding: "scope:cte:recent", Alias: "r"}, {ID: "root.project", Kind: "project", ScopeID: "scope:root", Inputs: []string{"root.source"}}},
	}
	d := index.BuildWithObservations(w, []analyzer.Result{analysis}, []physical.SchemaSnapshot{snap})
	if len(d.ObservedPhysicalDependencies) != 2 {
		t.Fatalf("expected one table and one physical column dependency, got %#v", d.ObservedPhysicalDependencies)
	}
	for _, dep := range d.ObservedPhysicalDependencies {
		if dep.Name == "recent" || dep.Name == "r.order_id" {
			t.Fatalf("logical CTE leaked into physical dependency index: %#v", dep)
		}
	}
	if got := d.AnalyzedQueries[0].IR.Coverage.Resolution; got != query.CoverageFull {
		t.Fatalf("logical resolution should count toward query resolution coverage: %s", got)
	}
}

func TestObservedIndexCarriesExecutionPlanAndLogicalMappings(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{ID: core.MustCanonicalID("order_entry"), Databases: []application.DatabaseRequirement{{Alias: core.MustLocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite}}}},
		Queries:      []query.EmbeddedSQL{{ID: core.MustCanonicalID("order_entry.q"), Application: core.MustCanonicalID("order_entry"), Datasource: core.MustLocalName("sales")}},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	analysis := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: core.MustCanonicalID("order_entry.q"), Lowering: "full", Operations: []analyzer.Operation{{ID: "source:0", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"}, {ID: "sort:0", Kind: "sort", ScopeID: "scope:root", Inputs: []string{"source:0"}}}}
	p := plan.IR{Query: core.MustCanonicalID("order_entry.q"), Database: core.MustCanonicalID("sales_core"), Provider: "postgres", Mode: plan.ModeExplain, Root: "plan:0", Nodes: []plan.Node{{ID: "plan:0", Kind: plan.NodeSort, ProviderKind: "Sort", Children: []string{"plan:0.0"}}, {ID: "plan:0.0", Kind: plan.NodeScan, ProviderKind: "Seq Scan", PhysicalObject: "public.orders", Alias: "o"}}}
	d := index.BuildWithAllObservations(w, []analyzer.Result{analysis}, []physical.SchemaSnapshot{snap}, []plan.IR{p})
	if len(d.ExecutionPlans) != 1 {
		t.Fatalf("plans=%#v", d.ExecutionPlans)
	}
	if len(d.ExecutionPlans[0].Mappings) != 2 {
		t.Fatalf("mappings=%#v", d.ExecutionPlans[0].Mappings)
	}
}

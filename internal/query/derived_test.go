package query_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

func TestDerivedQueryIRResolvesAgainstPhysicalSnapshot(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.q"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: q.ID, Tables: []string{"orders"}, Columns: []string{"orders.order_id"}}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Name: "orders", Columns: []physical.Column{{Name: "order_id", PhysicalType: "bigint"}}}}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if got.Coverage.Parse != query.CoverageFull || got.Coverage.Resolution != query.CoverageFull {
		t.Fatalf("unexpected coverage: %#v", got.Coverage)
	}
	if len(got.Sources) != 1 || got.Sources[0].Resolution != "schema-resolved" {
		t.Fatalf("source resolution: %#v", got.Sources)
	}
	if len(got.Fields) != 1 || got.Fields[0].Resolution != "schema-resolved" {
		t.Fatalf("field resolution: %#v", got.Fields)
	}
}

func TestDerivedQueryIRKeepsNameOnlyWhenNoSnapshot(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.q"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: q.ID, Tables: []string{"orders"}}
	got := query.DeriveFromAnalysis(q, core.MustCanonicalID("sales_core"), result, nil)
	if got.Sources[0].Resolution != "name-only" || got.Coverage.Resolution != query.CoverageNone {
		t.Fatalf("unexpected derived ir: %#v", got)
	}
}

func TestDerivedQueryIRResolvesAliasAndRelationalPlan(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.joined"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Tables: []string{"public.orders"}, Columns: []string{"o.order_id"},
		TableAliases: map[string]string{"o": "public.orders"}, Lowering: "full",
		Operations: []analyzer.Operation{
			{ID: "source:0", Kind: "source", Source: "public.orders", Alias: "o"},
			{ID: "filter:0", Kind: "filter", Inputs: []string{"source:0"}, Expressions: []string{"o.status = 'OPEN'"}},
			{ID: "project:0", Kind: "project", Inputs: []string{"filter:0"}, Expressions: []string{"o.order_id"}},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id", PhysicalType: "bigint"}}}}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if got.Coverage.Resolution != query.CoverageFull || got.Coverage.Lowering != query.CoverageFull {
		t.Fatalf("coverage=%#v", got.Coverage)
	}
	if len(got.Fields) != 1 || got.Fields[0].Resolution != "schema-resolved" {
		t.Fatalf("fields=%#v", got.Fields)
	}
	if len(got.Relational.Nodes) != 3 || got.Relational.Root != "project:0" {
		t.Fatalf("relational=%#v", got.Relational)
	}
	if got.Relational.Nodes[0].Resolution != "schema-resolved" {
		t.Fatalf("source resolution=%#v", got.Relational.Nodes[0])
	}
}

func TestDerivedQueryIRResolvesUnqualifiedFieldWhenUnique(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.simple"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: q.ID, Tables: []string{"public.orders"}, Columns: []string{"order_id"}, Lowering: "full", Operations: []analyzer.Operation{{ID: "source:0", Kind: "source", Source: "public.orders"}, {ID: "project:0", Kind: "project", Inputs: []string{"source:0"}, Expressions: []string{"order_id"}}}}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if got.Fields[0].Resolution != "schema-resolved" || got.Fields[0].PhysicalSource != "public.orders" {
		t.Fatalf("field=%#v", got.Fields[0])
	}
}

func TestDerivedQueryIRKeepsUnqualifiedFieldAmbiguousAcrossSources(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.ambiguous"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{ProtocolVersion: 1, Status: "ok", Query: q.ID, Tables: []string{"public.orders", "archive.orders"}, Columns: []string{"order_id"}, Lowering: "full", Operations: []analyzer.Operation{{ID: "source:0", Kind: "source", Source: "public.orders", Alias: "a"}, {ID: "source:1", Kind: "source", Source: "archive.orders", Alias: "b"}, {ID: "join:0", Kind: "join", Inputs: []string{"source:0", "source:1"}, JoinKind: "inner"}, {ID: "project:0", Kind: "project", Inputs: []string{"join:0"}, Expressions: []string{"order_id"}}}}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}, {Schema: "archive", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if got.Fields[0].Resolution != "name-only" || got.Fields[0].PhysicalSource != "" {
		t.Fatalf("field should remain ambiguous: %#v", got.Fields[0])
	}
	if len(got.Sources) != 2 || got.Sources[0].Alias != "a" || got.Sources[1].Alias != "b" {
		t.Fatalf("source occurrences lost: %#v", got.Sources)
	}
}

func TestDerivedQueryIRPreservesCTEScopeAndLineage(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.cte"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:cte:recent", ParentID: "scope:root", Kind: "cte", Name: "recent"},
		},
		TableAliases: map[string]string{"o": "public.orders"},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:0", ScopeID: "scope:cte:recent", Name: "order_id", Qualifier: "o"},
			{ID: "col:1", ScopeID: "scope:root", Name: "order_id", Qualifier: "r"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:cte:0", ScopeID: "scope:cte:recent", Name: "order_id", Expression: "o.order_id", Inputs: []string{"col:0"}},
			{ID: "out:root:0", ScopeID: "scope:root", Name: "order_id", Expression: "r.order_id", Inputs: []string{"col:1"}},
		},
		Operations: []analyzer.Operation{
			{ID: "cte.source:0", Kind: "source", ScopeID: "scope:cte:recent", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "cte.project:0", Kind: "project", ScopeID: "scope:cte:recent", Inputs: []string{"cte.source:0"}, Expressions: []string{"o.order_id"}},
			{ID: "root.source:0", Kind: "source", ScopeID: "scope:root", Source: "recent", SourceKind: "cte", Binding: "scope:cte:recent", Alias: "r"},
			{ID: "root.project:0", Kind: "project", ScopeID: "scope:root", Inputs: []string{"root.source:0"}, Expressions: []string{"r.order_id"}},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}}}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Scopes) != 2 || got.Scopes[1].ParentID != "scope:root" {
		t.Fatalf("scopes=%#v", got.Scopes)
	}
	if got.Relational.Root != "root.project:0" {
		t.Fatalf("root=%q", got.Relational.Root)
	}
	if len(got.Lineage) != 3 || len(got.OutputFields) != 2 {
		t.Fatalf("lineage=%#v outputs=%#v", got.Lineage, got.OutputFields)
	}
	if got.Fields[1].Resolution != "logical-resolved" || got.Fields[1].LogicalSourceField != got.OutputFields[0].ID {
		t.Fatalf("cte field should resolve through logical output: %#v", got.Fields[1])
	}
	if got.OutputFields[1].ID != "app.cte#output:scope_root:0" {
		t.Fatalf("root output id should be scope-local and stable: %q", got.OutputFields[1].ID)
	}
	if got.Sources[1].SourceKind != "cte" || got.Sources[1].Resolution != "logical" || got.Sources[1].Binding != "scope:cte:recent" {
		t.Fatalf("cte source=%#v", got.Sources[1])
	}
}

func TestDerivedQueryIRPreservesCorrelatedColumnScope(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.correlated"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes:       []analyzer.Scope{{ID: "scope:root", Kind: "root"}, {ID: "scope:subquery:0", ParentID: "scope:root", Kind: "subquery"}},
		TableAliases: map[string]string{"o": "public.orders", "l": "public.order_lines"},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:inner", ScopeID: "scope:subquery:0", Name: "order_id", Qualifier: "l"},
			{ID: "col:outer", ScopeID: "scope:subquery:0", Name: "order_id", Qualifier: "o", Correlated: true, OuterScopeID: "scope:root"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "root.project", Kind: "project", ScopeID: "scope:root", Inputs: []string{"root.source"}, Expressions: []string{"o.order_id"}},
			{ID: "sub.source", Kind: "source", ScopeID: "scope:subquery:0", Source: "public.order_lines", SourceKind: "table", Alias: "l"},
			{ID: "sub.filter", Kind: "filter", ScopeID: "scope:subquery:0", Inputs: []string{"sub.source"}, Expressions: []string{"l.order_id = o.order_id"}},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
		{Schema: "public", Name: "order_lines", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 2 || !got.Fields[1].Correlated || got.Fields[1].OuterScopeID != "scope:root" {
		t.Fatalf("fields=%#v", got.Fields)
	}
	if got.Fields[1].PhysicalSource != "public.orders" || got.Fields[1].Resolution != "schema-resolved" {
		t.Fatalf("correlated field=%#v", got.Fields[1])
	}
	if got.Relational.Root != "root.project" {
		t.Fatalf("nested scope displaced root: %q", got.Relational.Root)
	}
}

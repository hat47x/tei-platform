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

func TestDerivedQueryIRResolvesShadowedAliasesByScope(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.alias_shadow"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:subquery:0", Kind: "subquery", ParentID: "scope:root"},
		},
		// A flattened map is insufficient when x is reused in two scopes.
		TableAliases: map[string]string{"x": "public.orders"},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:root", ScopeID: "scope:root", Qualifier: "x", Name: "order_id"},
			{ID: "col:inner", ScopeID: "scope:subquery:0", Qualifier: "x", Name: "order_id"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", SourceKind: "table", Source: "public.orders", Alias: "x"},
			{ID: "inner.source", Kind: "source", ScopeID: "scope:subquery:0", SourceKind: "table", Source: "archive.audit", Alias: "x"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
		{Schema: "archive", Name: "audit", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 2 ||
		got.Fields[0].PhysicalSource != "public.orders" ||
		got.Fields[1].PhysicalSource != "archive.audit" ||
		got.Fields[0].Resolution != "schema-resolved" ||
		got.Fields[1].Resolution != "schema-resolved" {
		t.Fatalf("shadowed alias resolved across scopes: %#v", got.Fields)
	}
}

func TestDerivedQueryIRUnqualifiedFieldDoesNotSeeNestedScope(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.nested_unqualified"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:subquery:0", Kind: "subquery", ParentID: "scope:root"},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:root", ScopeID: "scope:root", Name: "order_id"},
			{ID: "col:inner", ScopeID: "scope:subquery:0", Name: "order_id"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", SourceKind: "table", Source: "public.orders", Alias: "o"},
			{ID: "inner.source", Kind: "source", ScopeID: "scope:subquery:0", SourceKind: "table", Source: "archive.audit", Alias: "a"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
		{Schema: "archive", Name: "audit", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 2 ||
		got.Fields[0].PhysicalSource != "public.orders" ||
		got.Fields[1].PhysicalSource != "archive.audit" {
		t.Fatalf("nested source polluted unqualified binding: %#v", got.Fields)
	}
}

func TestDerivedQueryIRUnknownSourceDoesNotProveUniqueColumn(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.unknown_source"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:unqualified", ScopeID: "scope:root", Name: "order_id"},
			{ID: "col:qualified", ScopeID: "scope:root", Qualifier: "o", Name: "order_id"},
		},
		Operations: []analyzer.Operation{
			{ID: "source:known", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "source:unknown", Kind: "source", ScopeID: "scope:root", Source: "public.unknown", SourceKind: "table", Alias: "u"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 2 || got.Fields[0].Resolution != "name-only" || got.Fields[0].PhysicalSource != "" {
		t.Fatalf("unknown source must not prove uniqueness: %#v", got.Fields)
	}
	if got.Fields[1].Resolution != "schema-resolved" || got.Fields[1].PhysicalSource != "public.orders" {
		t.Fatalf("explicit qualifier should still resolve known source: %#v", got.Fields[1])
	}
}

func TestDerivedQueryIRSelfJoinKeepsOccurrenceAmbiguity(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.self_join_ambiguous"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes:     []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{{ID: "col:0", ScopeID: "scope:root", Name: "order_id"}},
		Operations: []analyzer.Operation{
			{ID: "source:a", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "a"},
			{ID: "source:b", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "b"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 1 || got.Fields[0].Resolution == "schema-resolved" ||
		got.Fields[0].PhysicalSource != "" || got.Coverage.Resolution == query.CoverageFull {
		t.Fatalf("self-join occurrence ambiguity was silently deduplicated: %#v", got)
	}
}

func TestDerivedQueryIRRetainsCorrelatedOuterSource(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.outer_source"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:subquery:0", Kind: "subquery", ParentID: "scope:root"},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:outer", ScopeID: "scope:subquery:0", Qualifier: "o", Name: "order_id", Correlated: true, OuterScopeID: "scope:root"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "inner.source", Kind: "source", ScopeID: "scope:subquery:0", Source: "archive.audit", SourceKind: "table", Alias: "a"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
		{Schema: "archive", Name: "audit", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 1 || got.Fields[0].Resolution != "schema-resolved" ||
		got.Fields[0].PhysicalSource != "public.orders" || got.Fields[0].OuterScopeID != "scope:root" {
		t.Fatalf("explicit outer-scope column lost: %#v", got.Fields)
	}
}

func TestDerivedQueryIRDuplicateCTEOutputIsNotArbitrarilyBound(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.duplicate_cte_outputs"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:cte:dup", Kind: "cte", Name: "dup", ParentID: "scope:root"},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{{ID: "col:outer", ScopeID: "scope:root", Qualifier: "d", Name: "v"}},
		OutputFields: []analyzer.OutputField{
			{ID: "out:cte:0", ScopeID: "scope:cte:dup", Name: "v"},
			{ID: "out:cte:1", ScopeID: "scope:cte:dup", Name: "v"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", SourceKind: "cte", Source: "dup", Alias: "d", Binding: "scope:cte:dup"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 1 || got.Fields[0].Resolution == "logical-resolved" ||
		got.Fields[0].LogicalSourceField != "" || len(got.Lineage) != 0 {
		t.Fatalf("duplicate CTE column selected arbitrary lineage: %#v", got)
	}
}

func TestDerivedQueryIRMissingCTEBindingIsUnresolved(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.missing_cte"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", SourceKind: "cte", Source: "missing", Alias: "m", Binding: "scope:cte:missing"},
		},
	}
	got := query.DeriveFromAnalysis(q, core.MustCanonicalID("sales_core"), result, nil)
	if len(got.Sources) != 1 || got.Sources[0].Resolution == "logical" ||
		got.Coverage.Resolution == query.CoverageFull ||
		len(got.Relational.Nodes) != 1 || got.Relational.Nodes[0].Resolution == "logical" {
		t.Fatalf("missing CTE binding was treated as resolved: %#v", got)
	}
}

func TestDerivedQueryIRUnqualifiedMixedPhysicalAndCTESourceIsAmbiguous(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.mixed_source"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:cte:recent", Kind: "cte", Name: "recent", ParentID: "scope:root"},
		},
		ColumnRefs:   []analyzer.ColumnOccurrence{{ID: "col:root", ScopeID: "scope:root", Name: "order_id"}},
		OutputFields: []analyzer.OutputField{{ID: "out:recent", ScopeID: "scope:cte:recent", Name: "order_id"}},
		Operations: []analyzer.Operation{
			{ID: "root.table", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "root.cte", Kind: "source", ScopeID: "scope:root", Source: "recent", SourceKind: "cte", Alias: "r", Binding: "scope:cte:recent"},
		},
	}
	snap := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{
		{Schema: "public", Name: "orders", Columns: []physical.Column{{Name: "order_id"}}},
	}}
	got := query.DeriveFromAnalysis(q, snap.Database, result, &snap)
	if len(got.Fields) != 1 || got.Fields[0].Resolution == "schema-resolved" ||
		got.Fields[0].Resolution == "logical-resolved" || got.Fields[0].PhysicalSource != "" ||
		got.Fields[0].LogicalSourceField != "" || got.Coverage.Resolution == query.CoverageFull {
		t.Fatalf("unqualified column was arbitrarily assigned to the physical or CTE source: %#v", got)
	}
}

func TestDerivedQueryIRUnknownSourceAndMissingOuterScopeStayUnresolved(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.unknown_cte_scope"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:cte:recent", Kind: "cte", ParentID: "scope:root"},
			{ID: "scope:subquery:0", Kind: "subquery", ParentID: "scope:root"},
		},
		OutputFields: []analyzer.OutputField{{ID: "out:cte:0", ScopeID: "scope:cte:recent", Name: "order_id"}},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:unqualified", ScopeID: "scope:root", Name: "order_id"},
			{ID: "col:qualified", ScopeID: "scope:root", Qualifier: "r", Name: "order_id"},
			{ID: "col:missing-outer", ScopeID: "scope:subquery:0", Qualifier: "r", Name: "order_id", Correlated: true},
		},
		Operations: []analyzer.Operation{
			{ID: "known", Kind: "source", ScopeID: "scope:root", Source: "recent", SourceKind: "cte", Alias: "r", Binding: "scope:cte:recent"},
			{ID: "unknown", Kind: "source", ScopeID: "scope:root", Source: "missing", SourceKind: "cte", Alias: "m", Binding: "scope:cte:missing"},
		},
	}
	got := query.DeriveFromAnalysis(q, core.MustCanonicalID("sales_core"), result, nil)
	if len(got.Fields) != 3 || got.Fields[0].Resolution != "name-only" || got.Fields[0].LogicalSourceField != "" {
		t.Fatalf("unknown CTE must not prove uniqueness: %#v", got.Fields)
	}
	if got.Fields[1].Resolution != "logical-resolved" || got.Fields[1].LogicalSourceField == "" {
		t.Fatalf("explicit known alias should resolve: %#v", got.Fields[1])
	}
	if got.Fields[2].Resolution != "name-only" || got.Fields[2].LogicalSourceField != "" {
		t.Fatalf("missing outer scope must not default to root: %#v", got.Fields[2])
	}
}

func TestDerivedQueryIRCorrelatedCTEColumnUsesExplicitOuterScope(t *testing.T) {
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.correlated_cte"), Datasource: core.MustLocalName("sales")}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID,
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:cte:recent", Kind: "cte", Name: "recent", ParentID: "scope:root"},
			{ID: "scope:subquery:0", Kind: "subquery", ParentID: "scope:root"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:cte:0", ScopeID: "scope:cte:recent", Name: "order_id"},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:outer", ScopeID: "scope:subquery:0", Qualifier: "r", Name: "order_id", Correlated: true, OuterScopeID: "scope:root"},
		},
		Operations: []analyzer.Operation{
			{ID: "root.source", Kind: "source", ScopeID: "scope:root", Source: "recent", SourceKind: "cte", Alias: "r", Binding: "scope:cte:recent"},
		},
	}
	got := query.DeriveFromAnalysis(q, core.MustCanonicalID("sales_core"), result, nil)
	if len(got.Fields) != 1 || got.Fields[0].Resolution != "logical-resolved" ||
		len(got.OutputFields) != 1 || got.Fields[0].LogicalSourceField != got.OutputFields[0].ID ||
		!got.Fields[0].Correlated || got.Fields[0].OuterScopeID != "scope:root" {
		t.Fatalf("outer CTE output was not resolved at the recorded scope: %#v", got)
	}
}

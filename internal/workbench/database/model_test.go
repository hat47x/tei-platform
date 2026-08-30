package databaseworkbench_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/index"
	"tei/internal/query"
	databaseworkbench "tei/internal/workbench/database"
)

func TestBuildWorkbenchProjection(t *testing.T) {
	d := index.Derived{
		PhysicalSchemas: []physical.SchemaSnapshot{{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Kind: physical.TableKindTable, Columns: []physical.Column{{Name: "order_id", PhysicalType: "bigint"}}}}}},
		AnalyzedQueries: []index.AnalyzedQueryEntry{{Application: core.MustCanonicalID("order_entry"), IR: query.DerivedIR{
			Query: core.MustCanonicalID("order_entry.q"), Datasource: core.MustLocalName("sales"), Database: core.MustCanonicalID("sales_core"),
			Scopes:     []query.QueryScope{{ID: "scope:root", Kind: "root"}},
			Relational: query.RelationalIR{Nodes: []query.RelNode{{ID: "source:0", Kind: query.RelSource, ScopeID: "scope:root", Source: "public.orders", Resolution: "schema-resolved"}}, Root: "source:0"},
			Fields:     []query.FieldRef{{ID: "order_entry.q#fieldref:0", ScopeID: "scope:root", Name: "order_id", PhysicalSource: "public.orders", Resolution: "schema-resolved"}},
			Lineage:    []query.LineageEdge{{From: "order_entry.q#fieldref:0", To: "order_entry.q#output:scope_root:0"}},
		}}},
		ExecutionPlans: []index.ExecutionPlanEntry{{Application: core.MustCanonicalID("order_entry"), Plan: plan.IR{Query: core.MustCanonicalID("order_entry.q"), Database: core.MustCanonicalID("sales_core"), Provider: "postgres", Mode: plan.ModeExplainAnalyze, Root: "plan:0", Nodes: []plan.Node{{ID: "plan:0", Kind: plan.NodeScan, ProviderKind: "Seq Scan", PhysicalObject: "public.orders", Estimate: plan.Estimate{Rows: floatPtr(10)}, Actual: &plan.Actual{Rows: floatPtr(8), Loops: floatPtr(1)}}}}, Mappings: []plan.LogicalMapping{{RelNodeID: "source:0", PlanNodeID: "plan:0", Confidence: plan.MappingExact}}}},
	}
	got := databaseworkbench.Build(d)
	if len(got.Schemas) != 1 || got.Schemas[0].Tables[0].Name != "public.orders" {
		t.Fatalf("schemas=%#v", got.Schemas)
	}
	if len(got.Queries) != 1 || len(got.Queries[0].Graph) != 1 || len(got.Queries[0].Lineage) != 1 {
		t.Fatalf("queries=%#v", got.Queries)
	}
	if len(got.Queries[0].Plans) != 1 || len(got.Queries[0].Plans[0].Nodes) != 1 || got.Queries[0].Plans[0].Nodes[0].RowEstimateRatio == nil || *got.Queries[0].Plans[0].Nodes[0].RowEstimateRatio != 0.8 {
		t.Fatalf("plans=%#v", got.Queries[0].Plans)
	}
}

func floatPtr(v float64) *float64 { return &v }

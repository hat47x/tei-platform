package query_test

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

// QX-14d characterizes a read-only projection boundary. An analyzer.Result
// currently carries Query ID and protocol, not a digest of the SQL source
// actually analyzed. A caller must not treat parse coverage as proof that the
// result describes the current source bytes.
//
// This test intentionally locks down the existing observable behavior; it does
// not prescribe rejecting stale input inside DeriveFromAnalysis, alter D2's
// source-digest preflight, or establish that the real worker is incorrect.
func TestQX14dDerivedIRSameQueryIDOldSQLResultIsNotCurrentSourceProof(t *testing.T) {
	id := core.MustCanonicalID("app.qx14.origin")
	oldQuery := query.EmbeddedSQL{
		ID:         id,
		Datasource: core.MustLocalName("sales"),
		Source:     "SELECT order_id FROM public.orders WHERE active = 1",
	}
	currentQuery := oldQuery
	currentQuery.Source = "SELECT order_id FROM public.orders WHERE active = 0"

	// Synthetic result for the old request. Its Query ID is deliberately
	// identical to that of the edited Query.
	oldAnalysis := analyzer.Result{
		ProtocolVersion: 1,
		Status:          "ok",
		Query:           id,
		Tables:          []string{"public.orders"},
		Columns:         []string{"order_id"},
		Lowering:        "full",
		Operations: []analyzer.Operation{
			{ID: "source:0", Kind: "source", Source: "public.orders"},
			{ID: "filter:0", Kind: "filter", Inputs: []string{"source:0"}, Expressions: []string{"active = 1"}},
			{ID: "project:0", Kind: "project", Inputs: []string{"filter:0"}, Expressions: []string{"order_id"}},
		},
	}
	snapshot := physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{{
			Schema: "public",
			Name:   "orders",
			Columns: []physical.Column{
				{Name: "order_id", PhysicalType: "bigint"},
				{Name: "active", PhysicalType: "integer"},
			},
		}},
	}

	before := query.DeriveFromAnalysis(oldQuery, snapshot.Database, oldAnalysis, &snapshot)
	after := query.DeriveFromAnalysis(currentQuery, snapshot.Database, oldAnalysis, &snapshot)

	// This is a characterization: current DeriveFromAnalysis checks Query ID
	// and result status, not whether analyzer.Result belongs to q.Source.
	if before.Coverage.Parse != query.CoverageFull || after.Coverage.Parse != query.CoverageFull {
		t.Fatalf("expected parse coverage for same-ID synthetic result: before=%#v after=%#v",
			before.Coverage, after.Coverage)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("the current source changed but the synthetic analysis was held fixed; expected identical derived IR; before=%#v after=%#v",
			before, after)
	}

	// The observed derived IR still contains the predicate of the OLD SQL.
	// Therefore a Consumer must not call it the CURRENT SQL's analysis
	// without an independent, same-invocation source/worker binding witness.
	foundOldPredicate := false
	for _, node := range after.Relational.Nodes {
		if node.Kind != query.RelFilter {
			continue
		}
		for _, expr := range node.Expressions {
			if expr == "active = 1" {
				foundOldPredicate = true
			}
			if expr == "active = 0" {
				t.Fatalf("current predicate was invented without being in oldAnalysis: %#v", node)
			}
		}
	}
	if !foundOldPredicate {
		t.Fatalf("stale predicate must be observable in the synthetic derived plan: %#v", after.Relational)
	}
}

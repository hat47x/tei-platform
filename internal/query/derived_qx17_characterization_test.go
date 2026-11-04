package query_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"tei/internal/core"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

// These synthetic Operations are deliberately authored by the test, not produced
// by SQLGlot. This test exercises the REAL Go DeriveFromAnalysis implementation.
// A structurally connected graph is not an accepted SQL or business claim.
func TestQX17DerivedIRFromSyntheticAnalyzerOperations(t *testing.T) {
	tests := []struct {
		name, root, aggregateID, column, qualifier string
		ops                                        []analyzer.Operation
		wantAggregateSources                       []string
		wantRootSources                            []string
	}{
		{
			name: "aggregate-before-join", root: "join-groups", aggregateID: "agg-line",
			column: "amount", qualifier: "l",
			ops: []analyzer.Operation{
				{ID: "src-line", Kind: "source", ScopeID: "scope:root", Source: "line", Alias: "l"},
				{ID: "src-delivery", Kind: "source", ScopeID: "scope:root", Source: "delivery", Alias: "d"},
				{ID: "agg-line", Kind: "aggregate", ScopeID: "scope:root", Inputs: []string{"src-line"}, Expressions: []string{"SUM(l.amount)"}},
				{ID: "agg-delivery", Kind: "aggregate", ScopeID: "scope:root", Inputs: []string{"src-delivery"}, Expressions: []string{"COUNT(*)"}},
				{ID: "join-groups", Kind: "join", ScopeID: "scope:root", Inputs: []string{"agg-line", "agg-delivery"}, JoinKind: "inner"},
			},
			wantAggregateSources: []string{"src-line"}, wantRootSources: []string{"src-delivery", "src-line"},
		},
		{
			name: "join-before-aggregate", root: "agg-after-join", aggregateID: "agg-after-join",
			column: "amount", qualifier: "l",
			ops: []analyzer.Operation{
				{ID: "src-line", Kind: "source", ScopeID: "scope:root", Source: "line", Alias: "l"},
				{ID: "src-delivery", Kind: "source", ScopeID: "scope:root", Source: "delivery", Alias: "d"},
				{ID: "join-rows", Kind: "join", ScopeID: "scope:root", Inputs: []string{"src-line", "src-delivery"}, JoinKind: "inner"},
				{ID: "agg-after-join", Kind: "aggregate", ScopeID: "scope:root", Inputs: []string{"join-rows"}, Expressions: []string{"SUM(l.amount)"}},
			},
			wantAggregateSources: []string{"src-delivery", "src-line"}, wantRootSources: []string{"src-delivery", "src-line"},
		},
		{
			name: "left-join-unmatched-filter", root: "anti-filter", aggregateID: "",
			column: "id", qualifier: "h",
			ops: []analyzer.Operation{
				{ID: "src-header", Kind: "source", ScopeID: "scope:root", Source: "header", Alias: "h"},
				{ID: "src-delivery", Kind: "source", ScopeID: "scope:root", Source: "delivery", Alias: "d"},
				{ID: "left-join", Kind: "join", ScopeID: "scope:root", Inputs: []string{"src-header", "src-delivery"}, JoinKind: "left"},
				{ID: "anti-filter", Kind: "filter", ScopeID: "scope:root", Inputs: []string{"left-join"}, Expressions: []string{"d.id IS NULL"}},
			},
			wantRootSources: []string{"src-delivery", "src-header"},
		},
	}
	q := query.EmbeddedSQL{ID: core.MustCanonicalID("app.qx17"), Datasource: core.MustLocalName("sales")}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Result{
				ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
				Operations:   tt.ops,
				ColumnRefs:   []analyzer.ColumnOccurrence{{ID: "col:0", ScopeID: "scope:root", Name: tt.column, Qualifier: tt.qualifier}},
				OutputFields: []analyzer.OutputField{{ID: "out:0", ScopeID: "scope:root", Name: "display_value", Expression: "synthetic expression", Inputs: []string{"col:0"}}},
			}
			got := query.DeriveFromAnalysis(q, core.MustCanonicalID("sales_core"), result, nil)
			if got.Relational.Root != tt.root {
				t.Fatalf("root=%q, want=%q", got.Relational.Root, tt.root)
			}
			if len(got.Relational.Nodes) != len(tt.ops) {
				t.Fatalf("operations lost: got=%d, want=%d", len(got.Relational.Nodes), len(tt.ops))
			}
			if got.Coverage.Parse != query.CoverageFull || got.Coverage.Lowering != query.CoverageFull {
				t.Fatalf("unexpected parse/lowering coverage: %+v", got.Coverage)
			}
			// No physical SchemaSnapshot was supplied; a full lowering claim is
			// independent of source/column resolution and Provider completeness.
			if got.Coverage.Resolution != query.CoverageNone {
				t.Fatalf("unexpected physical resolution without schema: %+v", got.Coverage)
			}
			if !reflect.DeepEqual(qx17SourceOperations(got.Relational, got.Relational.Root), tt.wantRootSources) {
				t.Fatalf("root source occurrences mismatch: %+v", got.Relational)
			}
			if tt.aggregateID != "" && !reflect.DeepEqual(qx17SourceOperations(got.Relational, tt.aggregateID), tt.wantAggregateSources) {
				t.Fatalf("aggregate source dependencies mismatch: %+v", got.Relational)
			}
			if len(got.OutputFields) != 1 || len(got.OutputFields[0].Inputs) != 1 || len(got.Lineage) != 1 {
				t.Fatalf("expected one separate field lineage, got outputs=%+v edges=%+v", got.OutputFields, got.Lineage)
			}
			if got.OutputFields[0].Inputs[0] != got.Fields[0].ID || string(got.OutputFields[0].Inputs[0]) == tt.root {
				t.Fatalf("Field ID and RelNode ID must remain distinct: %+v", got)
			}
			blob, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var roundtrip query.DerivedIR
			if err := json.Unmarshal(blob, &roundtrip); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(roundtrip.Relational, got.Relational) || !reflect.DeepEqual(roundtrip.OutputFields, got.OutputFields) {
				t.Fatal("DerivedIR JSON roundtrip lost structure or output Field IDs")
			}
		})
	}
}

// This helper is strictly a graph reachability check. It neither interprets
// Expressions nor proves that any SQL result corresponds to the graph.
func qx17SourceOperations(ir query.RelationalIR, target string) []string {
	byID := make(map[string]query.RelNode, len(ir.Nodes))
	for _, node := range ir.Nodes {
		byID[node.ID] = node
	}
	seen, sources := map[string]bool{}, []string{}
	var walk func(string)
	walk = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		node, ok := byID[id]
		if !ok {
			return
		}
		if node.Kind == query.RelSource {
			sources = append(sources, node.ID)
			return
		}
		for _, upstream := range node.Inputs {
			walk(upstream)
		}
	}
	walk(target)
	sort.Strings(sources)
	return sources
}

package query_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

// SM-16 connects the independent SQLite NATURAL JOIN counterexample (SM-15)
// to the real Go DerivedIR projector. The analyzer.Result is deliberately
// synthetic and unchanged: it is NOT an actual SQLGlot/worker conformance test.
// It checks that Go's schema-resolution success alone is not evidence of
// implicit SQL dependency completeness or source/schema co-versioning.
func TestSM16GoDerivedIRDoesNotAttestNaturalJoinSchemaClosure(t *testing.T) {
	q := query.EmbeddedSQL{
		ID:         core.MustCanonicalID("sm16.natural_join"),
		Datasource: core.MustLocalName("sales"),
		Source:     "SELECT o.order_id FROM orders o NATURAL JOIN payments p",
	}
	baseline := physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{
			{Name: "orders", Columns: []physical.Column{
				{Name: "tenant_id"}, {Name: "order_id"}, {Name: "status"},
			}},
			{Name: "payments", Columns: []physical.Column{
				{Name: "tenant_id"}, {Name: "order_id"}, {Name: "amount"},
			}},
		},
	}
	modified := physical.SchemaSnapshot{
		Database: baseline.Database,
		Tables: []physical.Table{
			baseline.Tables[0],
			{Name: "payments", Columns: []physical.Column{
				{Name: "tenant_id"}, {Name: "order_id"}, {Name: "amount"},
				{Name: "status"},
			}},
		},
	}
	if reflect.DeepEqual(baseline, modified) {
		t.Fatal("SM-16 must change referenced input schema")
	}
	// A deliberately stale-but-well-formed analyzer response. Existing
	// protocol and query ID checks do not bind it to the schema bytes.
	result := analyzer.Result{
		ProtocolVersion: 1,
		Status:          "ok",
		Query:           q.ID,
		Lowering:        "full",
		Scopes:          []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:order_id", ScopeID: "scope:root", Name: "order_id", Qualifier: "o"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:order_id", ScopeID: "scope:root", Name: "order_id",
				Expression: "o.order_id", Inputs: []string{"col:order_id"}},
		},
		Operations: []analyzer.Operation{
			{ID: "src:orders", Kind: "source", ScopeID: "scope:root", Source: "orders", Alias: "o"},
			{ID: "src:payments", Kind: "source", ScopeID: "scope:root", Source: "payments", Alias: "p"},
			{ID: "join", Kind: "join", ScopeID: "scope:root", Inputs: []string{"src:orders", "src:payments"},
				JoinKind: "natural"},
			{ID: "project", Kind: "project", ScopeID: "scope:root", Inputs: []string{"join"},
				Expressions: []string{"o.order_id"}},
		},
	}
	first := query.DeriveFromAnalysis(q, baseline.Database, result, &baseline)
	second := query.DeriveFromAnalysis(q, modified.Database, result, &modified)
	if first.Coverage.Parse != query.CoverageFull || second.Coverage.Parse != query.CoverageFull {
		t.Fatalf("fixture must pass the existing query identity/parse gate: %#v / %#v",
			first.Coverage, second.Coverage)
	}
	if first.Coverage.Resolution != query.CoverageFull || second.Coverage.Resolution != query.CoverageFull {
		t.Fatalf("fixture must resolve both schemas: %#v / %#v", first.Coverage, second.Coverage)
	}
	b1, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("this probe expects unchanged Go projection of synthetic analysis, got:\n%s\n%s", b1, b2)
	}
	// SM-15 independently established that this SQL's real SQLite output
	// changes after adding payments.status. This test does NOT reproduce that
	// SQL execution, nor does it claim a real worker emits a stale response.
}

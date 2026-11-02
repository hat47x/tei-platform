package query

import (
	"encoding/json"
	"testing"

	"tei/internal/query/analyzer"
)

// qx27Result is a manually constructed analyzer response. It exercises the
// actual Go DerivedIR builder but is not evidence about the SQLGlot worker.
func qx27Result(predicate string, withPredicateColumn bool) (EmbeddedSQL, analyzer.Result) {
	q := EmbeddedSQL{ID: "qx27.subject", Source: "SELECT o.id FROM public.orders o WHERE " + predicate}
	result := analyzer.Result{
		ProtocolVersion: 1,
		Status:          "ok",
		Query:           q.ID,
		Lowering:        "full",
		Scopes:          []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:id", ScopeID: "scope:root", Name: "id", Qualifier: "o"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:id", ScopeID: "scope:root", Name: "id", Expression: "o.id", Inputs: []string{"col:id"}},
		},
		Operations: []analyzer.Operation{
			{ID: "src", Kind: "source", ScopeID: "scope:root", Source: "public.orders", Alias: "o"},
			{ID: "filter", Kind: "filter", ScopeID: "scope:root", Inputs: []string{"src"}, Expressions: []string{predicate}},
			{ID: "project", Kind: "project", ScopeID: "scope:root", Inputs: []string{"filter"}, Expressions: []string{"o.id"}},
		},
	}
	if withPredicateColumn {
		result.ColumnRefs = append(result.ColumnRefs, analyzer.ColumnOccurrence{
			ID: "col:secret", ScopeID: "scope:root", Name: "secret_score", Qualifier: "o",
		})
	}
	return q, result
}

func TestQX27OutputLineageExcludesHiddenPredicateInput(t *testing.T) {
	q, result := qx27Result("o.secret_score > 50", true)
	ir := DeriveFromAnalysis(q, "db", result, nil)
	if len(ir.OutputFields) != 1 || len(ir.OutputFields[0].Inputs) != 1 {
		t.Fatalf("expected one direct projection input, got %#v", ir.OutputFields)
	}
	if len(ir.Fields) != 2 || ir.Fields[1].Name != "secret_score" {
		t.Fatalf("hidden predicate input missing from recorded field occurrences: %#v", ir.Fields)
	}
	if ir.OutputFields[0].Inputs[0] != ir.Fields[0].ID || ir.OutputFields[0].Inputs[0] == ir.Fields[1].ID {
		t.Fatalf("projection lineage must not be confused with all condition inputs: %#v", ir)
	}
	if len(ir.Relational.Nodes) != 3 || ir.Relational.Nodes[1].Kind != RelFilter ||
		ir.Relational.Nodes[1].Expressions[0] != "o.secret_score > 50" {
		t.Fatalf("filter expression not preserved: %#v", ir.Relational)
	}
}

func TestQX27VisibleProjectionIdentityCannotProvePredicateEquality(t *testing.T) {
	qa, ra := qx27Result("o.secret_score > 50", true)
	qb, rb := qx27Result("o.secret_score > 70", true)
	a := DeriveFromAnalysis(qa, "db", ra, nil)
	b := DeriveFromAnalysis(qb, "db", rb, nil)
	outA, err := json.Marshal(a.OutputFields)
	if err != nil {
		t.Fatal(err)
	}
	outB, err := json.Marshal(b.OutputFields)
	if err != nil {
		t.Fatal(err)
	}
	if a.Query != b.Query || string(outA) != string(outB) {
		t.Fatalf("fixture needs identical Query/output projection identity: %q %q", outA, outB)
	}
	fullA, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	fullB, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(fullA) == string(fullB) || b.Relational.Nodes[1].Expressions[0] != "o.secret_score > 70" {
		t.Fatalf("same projection must not imply same condition inputs and predicate: %s vs %s", fullA, fullB)
	}
}

func TestQX27OrderingInputMayNotBeOutputInput(t *testing.T) {
	q, result := qx27Result("o.id IS NOT NULL", false)
	result.ColumnRefs = append(result.ColumnRefs, analyzer.ColumnOccurrence{
		ID: "col:sort", ScopeID: "scope:root", Name: "secret_rank", Qualifier: "o",
	})
	result.Operations = []analyzer.Operation{
		{ID: "src", Kind: "source", ScopeID: "scope:root", Source: "public.orders", Alias: "o"},
		{ID: "sort", Kind: "sort", ScopeID: "scope:root", Inputs: []string{"src"}, Expressions: []string{"o.secret_rank DESC"}},
		{ID: "project", Kind: "project", ScopeID: "scope:root", Inputs: []string{"sort"}, Expressions: []string{"o.id"}},
	}
	ir := DeriveFromAnalysis(q, "db", result, nil)
	if len(ir.Fields) != 2 || ir.Fields[1].Name != "secret_rank" {
		t.Fatalf("sort input should remain a separate field occurrence: %#v", ir.Fields)
	}
	if len(ir.OutputFields) != 1 || len(ir.OutputFields[0].Inputs) != 1 ||
		ir.OutputFields[0].Inputs[0] == ir.Fields[1].ID {
		t.Fatalf("sort dependency must not be inferred from output-only lineage: %#v", ir.OutputFields)
	}
	if ir.Relational.Nodes[1].Expressions[0] != "o.secret_rank DESC" {
		t.Fatalf("ordering expression lost: %#v", ir.Relational)
	}
}

func TestQX27CorrelatedExistenceHasNonProjectionInputs(t *testing.T) {
	q := EmbeddedSQL{ID: "qx27.correlated"}
	result := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "partial",
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "scope:sub", ParentID: "scope:root", Kind: "subquery"},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:out", ScopeID: "scope:root", Name: "id", Qualifier: "o"},
			{ID: "col:inner", ScopeID: "scope:sub", Name: "hidden_flag", Qualifier: "s"},
			{ID: "col:outer", ScopeID: "scope:sub", Name: "id", Qualifier: "o", Correlated: true, OuterScopeID: "scope:root"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:0", ScopeID: "scope:root", Name: "id", Expression: "o.id", Inputs: []string{"col:out"}},
		},
		Operations: []analyzer.Operation{
			{ID: "src:root", Kind: "source", ScopeID: "scope:root", Source: "public.orders", Alias: "o"},
			{ID: "proj:root", Kind: "project", ScopeID: "scope:root", Inputs: []string{"src:root"}, Expressions: []string{"o.id"}},
			{ID: "src:sub", Kind: "source", ScopeID: "scope:sub", Source: "private.secret", Alias: "s"},
			{ID: "filter:sub", Kind: "filter", ScopeID: "scope:sub", Inputs: []string{"src:sub"}, Expressions: []string{"s.hidden_flag = 1 AND s.id=o.id"}},
		},
	}
	ir := DeriveFromAnalysis(q, "db", result, nil)
	if ir.Coverage.Lowering != CoveragePartial || ir.Relational.Root != "proj:root" {
		t.Fatalf("partial nested condition must not displace root: %#v", ir)
	}
	if len(ir.Fields) != 3 || !ir.Fields[2].Correlated || ir.Fields[2].OuterScopeID != "scope:root" {
		t.Fatalf("cross-scope input missing: %#v", ir.Fields)
	}
	if len(ir.OutputFields) != 1 || len(ir.OutputFields[0].Inputs) != 1 {
		t.Fatalf("only the direct projection input should be in output lineage: %#v", ir.OutputFields)
	}
}

func TestQX27LoweringFullIsNotProofOfInputOccurrenceCompleteness(t *testing.T) {
	q, incomplete := qx27Result("o.secret_score > 50", false)
	_, complete := qx27Result("o.secret_score > 50", true)
	a := DeriveFromAnalysis(q, "db", incomplete, nil)
	b := DeriveFromAnalysis(q, "db", complete, nil)
	if a.Coverage.Lowering != CoverageFull || b.Coverage.Lowering != CoverageFull {
		t.Fatalf("fixture intentionally reports analyzer-provided full lowering: %+v %+v", a.Coverage, b.Coverage)
	}
	if len(a.Fields) != 1 || len(b.Fields) != 2 {
		t.Fatalf("the Go layer has no independent completeness check for omitted column refs: %#v %#v", a.Fields, b.Fields)
	}
	if a.Relational.Nodes[1].Expressions[0] != b.Relational.Nodes[1].Expressions[0] {
		t.Fatal("same filter text should be preserved in both synthetic responses")
	}
	// Deliberately malformed analyzer input: this test does NOT show that
	// the real SQLGlot worker omits condition columns. It shows that the
	// derived layer trusts analyzer-provided lowering coverage as-is.
}

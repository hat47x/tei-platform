package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"tei/internal/query/analyzer"
)

// qx20SyntheticAnalysis deliberately supplies an analyzer.Result by hand.
// It characterizes the real Go IR projection/JSON surface only: it does not
// validate the SQLGlot worker, SQL validity, or the actual disclosure policy.
func qx20SyntheticAnalysis(privateTable, privateToken string) (EmbeddedSQL, analyzer.Result) {
	q := EmbeddedSQL{
		ID: "qx20.query",
		Source: fmt.Sprintf(
			"SELECT o.amount FROM public.orders o JOIN %s x ON x.order_id = o.id WHERE x.token = '%s'",
			privateTable, privateToken,
		),
	}
	result := analyzer.Result{
		Query:  q.ID,
		Status: "ok",
		Scopes: []analyzer.Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{{
			ID: "column:0", ScopeID: "scope:root", Name: "amount", Qualifier: "o",
		}},
		OutputFields: []analyzer.OutputField{{
			ID: "output:0", ScopeID: "scope:root", Name: "amount",
			Expression: "o.amount", Inputs: []string{"column:0"},
		}},
		Operations: []analyzer.Operation{
			{ID: "source:public", Kind: "source", ScopeID: "scope:root", Source: "public.orders", SourceKind: "table", Alias: "o"},
			{ID: "source:private", Kind: "source", ScopeID: "scope:root", Source: privateTable, SourceKind: "table", Alias: "x"},
			{ID: "filter:private", Kind: "filter", ScopeID: "scope:root", Inputs: []string{"source:public", "source:private"},
				Expressions: []string{fmt.Sprintf("x.token = '%s'", privateToken)}},
			{ID: "project:public", Kind: "project", ScopeID: "scope:root", Inputs: []string{"filter:private"},
				Expressions: []string{"o.amount"}},
		},
		Lowering: "full",
	}
	return q, result
}

func qx20AsJSON(t *testing.T, ir DerivedIR) string {
	t.Helper()
	data, err := json.Marshal(ir)
	if err != nil {
		t.Fatalf("marshal DerivedIR: %v", err)
	}
	return string(data)
}

func TestQX20GoIRRawJSONCarriesPrivateIdentifiersAndPredicate(t *testing.T) {
	q, result := qx20SyntheticAnalysis("private_a.secret_orders", "marker-alpha")
	ir := DeriveFromAnalysis(q, "db", result, nil)
	raw := qx20AsJSON(t, ir)

	if ir.Coverage.Parse != CoverageFull || ir.Coverage.Lowering != CoverageFull {
		t.Fatalf("unexpected analyzer-provided structural coverage: %+v", ir.Coverage)
	}
	// No physical schema snapshot was supplied. Parsing and lowering are not
	// evidence of physical source resolution or a grant to publish this JSON.
	if ir.Coverage.Resolution == CoverageFull {
		t.Fatalf("unexpected physical resolution from nil snapshot: %+v", ir.Coverage)
	}
	for _, want := range []string{"private_a.secret_orders", "marker-alpha", "source:private", "filter:private"} {
		if !strings.Contains(raw, want) {
			t.Errorf("raw IR JSON lost the synthetic sensitive input %q: %s", want, raw)
		}
	}
}

func TestQX20GoIRSameQueryIDAndFieldIDDoNotProvePublicObservationalEquality(t *testing.T) {
	qA, resultA := qx20SyntheticAnalysis("private_a.secret_orders", "marker-alpha")
	qB, resultB := qx20SyntheticAnalysis("private_b.secret_orders", "marker-beta")
	a := DeriveFromAnalysis(qA, "db", resultA, nil)
	b := DeriveFromAnalysis(qB, "db", resultB, nil)
	rawA, rawB := qx20AsJSON(t, a), qx20AsJSON(t, b)

	if a.Query != b.Query || a.Query != qA.ID {
		t.Fatalf("test fixture must keep one Query ID: %q != %q", a.Query, b.Query)
	}
	if len(a.OutputFields) != 1 || len(b.OutputFields) != 1 || a.OutputFields[0].ID != b.OutputFields[0].ID {
		t.Fatalf("expected stable ordinal-based output field ID")
	}
	if rawA == rawB {
		t.Fatal("raw IR observations should differ for private-source-only changes")
	}
	if strings.Contains(rawA, "marker-beta") || strings.Contains(rawB, "marker-alpha") {
		t.Fatal("the synthetic secret sources crossed their separate analyzer results")
	}

	// This is a TEST-ONLY assumed-safe summary, not a TEI redactor or grant.
	// The real Consumer projection is owned by the existing disclosure lane.
	approvedSummary := func(ir DerivedIR) string {
		summary, err := json.Marshal(struct {
			Query  string
			Status string
		}{Query: string(ir.Query), Status: "withheld"})
		if err != nil {
			t.Fatalf("marshal test-only summary: %v", err)
		}
		return string(summary)
	}
	if approvedSummary(a) != approvedSummary(b) {
		t.Fatal("the test-only summary should not expose private-source differences")
	}
}

func TestQX20GoIRAnalyzerProvidedCoverageDoesNotEraseExpressions(t *testing.T) {
	q, result := qx20SyntheticAnalysis("private_a.secret_orders", "marker-alpha")
	result.Lowering = "partial"
	ir := DeriveFromAnalysis(q, "db", result, nil)
	raw := qx20AsJSON(t, ir)
	if ir.Coverage.Lowering != CoveragePartial {
		t.Fatalf("expected partial structural lowering, got %q", ir.Coverage.Lowering)
	}
	if !strings.Contains(raw, "marker-alpha") {
		t.Fatalf("partial lowering is not a metadata-disclosure policy: %s", raw)
	}
}

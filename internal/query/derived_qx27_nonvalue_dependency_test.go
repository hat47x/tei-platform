package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"tei/internal/query/analyzer"
)

// QX-27 characterizes the real Go DeriveFromAnalysis projection using synthetic
// analyzer.Result instances. These tests do not invoke the SQLGlot worker,
// resolve a real DB schema, or establish an authorization/disclosure decision.
func qx27SyntheticResult(kind, hiddenField, expression string) (EmbeddedSQL, analyzer.Result) {
	q := EmbeddedSQL{
		ID:     "qx27.same-query",
		Source: fmt.Sprintf("SELECT o.id FROM orders o %s %s", kind, expression),
	}
	root := "scope:root"
	result := analyzer.Result{
		ProtocolVersion: 1,
		Query:           q.ID,
		Status:          "ok",
		Lowering:        "full",
		Scopes:          []analyzer.Scope{{ID: root, Kind: "root"}},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:visible", ScopeID: root, Name: "id", Qualifier: "o"},
			{ID: "col:condition", ScopeID: root, Name: hiddenField, Qualifier: "o"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:visible", ScopeID: root, Name: "id",
				Expression: "o.id", Inputs: []string{"col:visible"}},
		},
		Operations: []analyzer.Operation{
			{ID: "source:o", Kind: string(RelSource), ScopeID: root, Source: "orders", SourceKind: "table", Alias: "o"},
			{ID: "condition", Kind: kind, ScopeID: root, Inputs: []string{"source:o"}, Expressions: []string{expression}},
			{ID: "project", Kind: string(RelProject), ScopeID: root,
				Inputs: []string{"condition"}, Expressions: []string{"o.id"}},
		},
	}
	return q, result
}

func qx27IR(t *testing.T, kind, field, expression string) DerivedIR {
	t.Helper()
	q, analysis := qx27SyntheticResult(kind, field, expression)
	ir := DeriveFromAnalysis(q, "qx27.database", analysis, nil)
	if ir.Coverage.Parse != CoverageFull || ir.Coverage.Lowering != CoverageFull {
		t.Fatalf("synthetic parser/lowering status was not preserved: %+v", ir.Coverage)
	}
	if len(ir.OutputFields) != 1 || len(ir.Fields) != 2 || len(ir.Relational.Nodes) != 3 {
		t.Fatalf("unexpected actual IR projection: %#v", ir)
	}
	return ir
}

func qx27HasLineage(ir DerivedIR, from, to FieldID) bool {
	for _, edge := range ir.Lineage {
		if edge.From == from && edge.To == to {
			return true
		}
	}
	return false
}

func TestQX27FilterInputExistsWithoutValueLineageToOutput(t *testing.T) {
	ir := qx27IR(t, string(RelFilter), "hidden_score", "o.hidden_score > 0")
	out := ir.OutputFields[0]
	if !qx27HasLineage(ir, ir.Fields[0].ID, out.ID) {
		t.Fatalf("visible value input lost its lineage edge: %#v", ir.Lineage)
	}
	if qx27HasLineage(ir, ir.Fields[1].ID, out.ID) {
		t.Fatalf("fixture expects hidden predicate input outside output value lineage: %#v", ir.Lineage)
	}
	if ir.Fields[1].Name != "hidden_score" {
		t.Fatalf("non-value input field was lost: %#v", ir.Fields)
	}
	if ir.Relational.Nodes[1].Kind != RelFilter ||
		len(ir.Relational.Nodes[1].Expressions) != 1 ||
		ir.Relational.Nodes[1].Expressions[0] != "o.hidden_score > 0" {
		t.Fatalf("filter expression was not preserved: %#v", ir.Relational.Nodes)
	}
	// The absence of a value-lineage edge is not evidence that the filter
	// input is unnecessary, unused, or safe to disclose.
}

func TestQX27SortFieldDoesNotBecomeOutputValueLineage(t *testing.T) {
	ir := qx27IR(t, string(RelSort), "secret_rank", "o.secret_rank DESC")
	if ir.Relational.Nodes[1].Kind != RelSort {
		t.Fatalf("sort stage lost: %#v", ir.Relational.Nodes)
	}
	if ir.Fields[1].Name != "secret_rank" || qx27HasLineage(ir, ir.Fields[1].ID, ir.OutputFields[0].ID) {
		t.Fatalf("sort input should be a separate non-value dependency: %+v", ir)
	}
	if got := ir.Relational.Nodes[1].Expressions[0]; got != "o.secret_rank DESC" {
		t.Fatalf("sort expression lost: %s", got)
	}
}

func TestQX27SameQueryAndOutputFieldIDsDoNotBindPredicateMeaning(t *testing.T) {
	a := qx27IR(t, string(RelFilter), "hidden_score", "o.hidden_score > 0")
	b := qx27IR(t, string(RelFilter), "hidden_score", "o.hidden_score > 99")
	if a.Query != b.Query || a.OutputFields[0].ID != b.OutputFields[0].ID {
		t.Fatalf("same logical Query and ordinal output identity expected: %q/%q", a.Query, b.Query)
	}
	if fmt.Sprint(a.Lineage) != fmt.Sprint(b.Lineage) {
		t.Fatalf("test fixture needs unchanged value-only lineage: %+v %+v", a.Lineage, b.Lineage)
	}
	if a.Relational.Nodes[1].Expressions[0] == b.Relational.Nodes[1].Expressions[0] {
		t.Fatal("test fixture must vary the predicate without changing projected value")
	}
	// Query/Output IDs and value-lineage equality alone cannot qualify a
	// cached WHERE/EXISTS/COUNT/top-N result for reuse after a predicate edit.
}

func TestQX27HiddenConditionAppearsInRawIRJSON(t *testing.T) {
	ir := qx27IR(t, string(RelFilter), "confidential_flag", "o.confidential_flag = 'private-marker'")
	raw, err := json.Marshal(ir)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"confidential_flag", "private-marker", "condition"} {
		if !strings.Contains(string(raw), token) {
			t.Fatalf("raw DerivedIR JSON did not carry %s: %s", token, raw)
		}
	}
	// Raw JSON is an internal artifact, not a Consumer-approved projection.
}

func TestQX27ExistsRelationChangesWithoutOutputValueDependency(t *testing.T) {
	root := "scope:root"
	q := EmbeddedSQL{ID: "qx27.exists", Source: "SELECT o.id FROM orders o WHERE EXISTS(SELECT 1 FROM private_gate g WHERE g.order_id=o.id AND g.secret='X')"}
	analysis := analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: q.ID, Lowering: "full",
		Scopes: []analyzer.Scope{
			{ID: root, Kind: "root"},
			{ID: "scope:sub", Kind: "subquery", ParentID: root},
		},
		ColumnRefs: []analyzer.ColumnOccurrence{
			{ID: "col:visible", ScopeID: root, Name: "id", Qualifier: "o"},
			{ID: "col:secret", ScopeID: "scope:sub", Name: "secret", Qualifier: "g"},
		},
		OutputFields: []analyzer.OutputField{
			{ID: "out:id", ScopeID: root, Name: "id",
				Expression: "o.id", Inputs: []string{"col:visible"}},
		},
		Operations: []analyzer.Operation{
			{ID: "src:o", Kind: string(RelSource), ScopeID: root, Source: "orders", Alias: "o"},
			{ID: "src:g", Kind: string(RelSource), ScopeID: "scope:sub", Source: "private_gate", Alias: "g"},
			{ID: "check-exists", Kind: string(RelFilter), ScopeID: root, Inputs: []string{"src:o"},
				Expressions: []string{"EXISTS(SELECT 1 FROM private_gate g WHERE g.order_id=o.id AND g.secret='X')"}},
			{ID: "project", Kind: string(RelProject), ScopeID: root,
				Inputs: []string{"check-exists"}, Expressions: []string{"o.id"}},
		},
	}
	ir := DeriveFromAnalysis(q, "qx27.database", analysis, nil)
	if len(ir.Sources) != 2 || len(ir.Fields) != 2 || len(ir.Lineage) != 1 {
		t.Fatalf("the current derived model must show two sources but value lineage for one output input: %#v", ir)
	}
	if ir.Sources[1].Name != "private_gate" || ir.Fields[1].Name != "secret" {
		t.Fatalf("nested predicate relation/field absent: %#v %#v", ir.Sources, ir.Fields)
	}
	if qx27HasLineage(ir, ir.Fields[1].ID, ir.OutputFields[0].ID) {
		t.Fatalf("test expects non-value EXISTS dependency outside output value lineage: %+v", ir.Lineage)
	}
	if ir.Relational.Root != "project" {
		t.Fatalf("nested EXISTS stage must not replace root: %s", ir.Relational.Root)
	}
	// There is no typed edge here from the EXISTS predicate to output row
	// membership; expressions remain strings. This test characterizes that
	// limitation without inventing a general SQL dependency model.
}

func TestQX27ChangingProjectedInputChangesValueLineage(t *testing.T) {
	q, analysis := qx27SyntheticResult(string(RelFilter), "hidden_score", "o.hidden_score > 0")
	analysis.OutputFields[0].Inputs = []string{"col:condition"}
	ir := DeriveFromAnalysis(q, "qx27.database", analysis, nil)
	if !qx27HasLineage(ir, ir.Fields[1].ID, ir.OutputFields[0].ID) {
		t.Fatalf("control case must link explicitly projected condition field: %+v", ir.Lineage)
	}
	if qx27HasLineage(ir, ir.Fields[0].ID, ir.OutputFields[0].ID) {
		t.Fatalf("control case should no longer link the removed value occurrence: %+v", ir.Lineage)
	}
}

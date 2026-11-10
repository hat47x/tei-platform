package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

// Each TraceRow carries an `issue` that explains why the row is unresolved.
// The values are only useful if something produces them, so every value gets a
// graph that makes it appear. #3906 reported that four of the six cannot be
// reached through the CLI's projection path; that is a statement about which
// graphs the projection builds, and these tests keep the row-level contract
// honest independently of it.
func TestTraceRowIssueValuesAreProduced(t *testing.T) {
	scope := "orders"
	data := traceKey(scope, "data", "order-line")
	fn := traceKey(scope, "function", "submit-order")
	task := traceKey(scope, "task", "confirm-order")

	cases := []struct {
		name   string
		graph  modelref.Graph
		rules  []modelref.Rule
		want   string
		detail string
	}{
		{
			name: "invalid-source: the source key has no node",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: task}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(task), Relation: "triggers"},
				},
			},
			rules:  []modelref.Rule{{Relation: "triggers", FromKind: "function", ToKind: "task"}},
			want:   "invalid-source",
			detail: "an edge whose source is not declared is reported, not dropped",
		},
		{
			name: "ambiguous-source: the source key is declared twice",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: task}, {Key: fn}, {Key: fn}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(task), Relation: "triggers"},
				},
			},
			rules:  []modelref.Rule{{Relation: "triggers", FromKind: "function", ToKind: "task"}},
			want:   "ambiguous-source",
			detail: "two declarations of the source make the edge's origin unclear",
		},
		{
			name: "invalid-target: the target resolves to a key without a scope",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: fn}, {Key: modelref.Key{Subject: data.Subject}}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(modelref.Key{Subject: data.Subject}), Relation: "reads"},
				},
			},
			rules:  []modelref.Rule{{Relation: "reads", FromKind: "function", ToKind: "data"}},
			want:   "invalid-target",
			detail: "a target without a scope is not a usable key",
		},
		{
			name: "invalid-edge-type: no rule covers the triple",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: data}, {Key: fn}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(data), Relation: "reads"},
				},
			},
			rules:  nil,
			want:   "invalid-edge-type",
			detail: "an edge type without a rule is unresolved",
		},
		{
			name: "invalid-rule: the matching rule leaves a field empty",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: data}, {Key: fn}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(data)},
				},
			},
			rules:  []modelref.Rule{{FromKind: "function", ToKind: "data"}},
			want:   "invalid-rule",
			detail: "a rule that does not name its relation cannot be matched",
		},
		{
			name: "conflicting-rule: two rules share a key and disagree",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  []modelref.Node{{Key: data}, {Key: fn}},
				Edges: []modelref.Edge{
					{From: fn, To: traceRef(data), Relation: "reads"},
				},
			},
			rules: []modelref.Rule{
				{Relation: "reads", FromKind: "function", ToKind: "data"},
				{Relation: "reads", FromKind: "function", ToKind: "data", AcyclicGroup: "changes-meaning"},
			},
			want:   "conflicting-rule",
			detail: "two meanings for one triple are reported, not resolved",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// The subject is the edge's source for outgoing traces; the
			// duplicated and missing source cases use an incoming trace so the
			// requested subject itself stays unambiguous.
			subject := fn
			direction := "outgoing"
			if testCase.want == "ambiguous-source" || testCase.want == "invalid-source" {
				subject = task
				direction = "incoming"
			}
			if testCase.want == "invalid-rule" {
				subject = fn
				direction = "outgoing"
			}
			result, err := modelref.TraceDirect(testCase.graph, testCase.rules, subject, modelref.TraceOptions{Direction: direction})
			if err != nil {
				t.Fatalf("%s: %v", testCase.detail, err)
			}
			if len(result.Rows) != 1 {
				t.Fatalf("%s: expected one row, got %+v", testCase.detail, result.Rows)
			}
			if result.Rows[0].Issue != testCase.want || result.Rows[0].Status != "unresolved" {
				t.Fatalf("%s: issue=%q status=%q, want %q", testCase.detail, result.Rows[0].Issue, result.Rows[0].Status, testCase.want)
			}
			if result.Unresolved != 1 {
				t.Fatalf("%s: unresolved count=%d, want 1", testCase.detail, result.Unresolved)
			}
		})
	}
}

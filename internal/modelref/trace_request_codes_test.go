package modelref_test

import (
	"errors"
	"testing"

	"tei/internal/modelref"
)

// TraceDirect refuses before it selects any edge when the graph, the subject or
// the request cannot be used. A caller branches on these codes, so each one is
// produced here (#4032).
func TestTraceRequestCodesAreProduced(t *testing.T) {
	graph, rules, subject := traceGraph()

	cases := []struct {
		name    string
		graph   modelref.Graph
		subject modelref.Key
		options modelref.TraceOptions
		want    string
	}{
		{
			name:    "unsupported schema",
			graph:   modelref.Graph{Schema: "tei.modelref.v9", Nodes: graph.Nodes, Edges: graph.Edges},
			subject: subject,
			options: modelref.TraceOptions{Direction: "outgoing"},
			want:    "UNSUPPORTED_SCHEMA",
		},
		{
			name:    "subject not found",
			graph:   graph,
			subject: traceKey("orders", "data", "missing"),
			options: modelref.TraceOptions{Direction: "outgoing"},
			want:    "SUBJECT_NOT_FOUND",
		},
		{
			name: "subject ambiguous",
			graph: modelref.Graph{
				Schema: "tei.modelref.v0",
				Nodes:  append(append([]modelref.Node{}, graph.Nodes...), modelref.Node{Key: subject}),
				Edges:  graph.Edges,
			},
			subject: subject,
			options: modelref.TraceOptions{Direction: "outgoing"},
			want:    "SUBJECT_AMBIGUOUS",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := modelref.TraceDirect(testCase.graph, rules, testCase.subject, testCase.options)
			if err == nil {
				t.Fatalf("expected %s, got no error", testCase.want)
			}
			var traceErr *modelref.TraceError
			if !errors.As(err, &traceErr) {
				t.Fatalf("expected a trace error with code %s, got %v", testCase.want, err)
			}
			if traceErr.Code != testCase.want {
				t.Fatalf("expected code %s, got %s (%v)", testCase.want, traceErr.Code, err)
			}
		})
	}
}

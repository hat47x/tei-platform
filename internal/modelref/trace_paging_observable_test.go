package modelref_test

import (
	"encoding/json"
	"testing"

	"tei/internal/modelref"
)

// pagingGraph builds a graph with n incoming edges to one subject, so a full
// scan has n rows.
func pagingGraph(n int) (modelref.Graph, []modelref.Rule, modelref.Key) {
	target := traceKey("s", "data", "order")
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: target}}}
	for i := 0; i < n; i++ {
		source := traceKey("s", "data", "source-"+string(rune('a'+i)))
		graph.Nodes = append(graph.Nodes, modelref.Node{Key: source})
		graph.Edges = append(graph.Edges, modelref.Edge{
			From:     source,
			To:       modelref.Ref{Scope: target.Scope, Subject: target.Subject},
			Relation: "reads",
		})
	}
	rules := []modelref.Rule{{Relation: "reads", FromKind: "data", ToKind: "data"}}
	return graph, rules, target
}

// TestOffsetPastTheEndIsObservable pins the #1177 requirement that a truncated
// page can be told from a page that legitimately has no rows left. Before this
// an offset past the last row was clamped silently: the response showed only the
// clamped value, so the caller could not see that its request was out of range.
func TestOffsetPastTheEndIsObservable(t *testing.T) {
	graph, rules, target := pagingGraph(3)

	past, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Mode: "full", Direction: "incoming", Offset: 99,
	})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if past.Offset != 3 {
		t.Fatalf("offset = %d, want the clamped row count 3", past.Offset)
	}
	if past.RequestedOffset != 99 {
		t.Fatalf("requestedOffset = %d, want the caller's 99", past.RequestedOffset)
	}
	if len(past.Rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(past.Rows))
	}
	if past.Total != 3 {
		t.Fatalf("total = %d, want 3", past.Total)
	}

	// The last legitimate page must not look clamped.
	tail, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Mode: "full", Direction: "incoming", Offset: 3,
	})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if tail.Offset != 3 || tail.RequestedOffset != 0 || len(tail.Rows) != 0 {
		t.Fatalf("a legitimate empty tail was marked as clamped: %+v", tail)
	}

	// A page inside the range reports neither a clamp nor a requested offset.
	inside, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Mode: "full", Direction: "incoming", Offset: 1, Limit: 1,
	})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if inside.Offset != 1 || inside.RequestedOffset != 0 || len(inside.Rows) != 1 {
		t.Fatalf("an in-range page was reported as clamped: %+v", inside)
	}
}

// The wire name is what a caller reads, so it is fixed here.
func TestRequestedOffsetWireName(t *testing.T) {
	encoded, err := json.Marshal(modelref.TraceResult{
		SchemaVersion: "tei.modelref.trace/v0", Mode: "full", Total: 3,
		Offset: 3, RequestedOffset: 99, Rows: []modelref.TraceRow{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["requestedOffset"]) != "99" {
		t.Fatalf("requestedOffset = %s", fields["requestedOffset"])
	}
	if string(fields["offset"]) != "3" {
		t.Fatalf("offset = %s", fields["offset"])
	}
	// A page that was not clamped omits the field, so its absence is meaningful.
	plain, err := json.Marshal(modelref.TraceResult{Mode: "full", Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	var plainFields map[string]json.RawMessage
	if err := json.Unmarshal(plain, &plainFields); err != nil {
		t.Fatal(err)
	}
	if _, present := plainFields["requestedOffset"]; present {
		t.Fatalf("an unclamped page carried requestedOffset: %s", plain)
	}
}

// Summary ignores paging entirely, so it must not report a requested offset.
func TestSummaryDoesNotReportRequestedOffset(t *testing.T) {
	graph, rules, target := pagingGraph(2)
	summary, err := modelref.TraceDirect(graph, rules, target, modelref.TraceOptions{
		Mode: "summary", Direction: "incoming",
	})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	if summary.RequestedOffset != 0 || summary.Total != 2 {
		t.Fatalf("summary = %+v", summary)
	}
}

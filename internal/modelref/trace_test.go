package modelref_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func traceKey(scope, kind, id string) modelref.Key {
	return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
}
func traceRef(key modelref.Key) modelref.Ref {
	return modelref.Ref{Scope: key.Scope, Subject: key.Subject}
}
func traceGraph() (modelref.Graph, []modelref.Rule, modelref.Key) {
	data := traceKey("orders", "data", "order-line")
	fn := traceKey("orders", "function", "submit-order")
	task := traceKey("orders", "task", "confirm-order")
	g := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes: []modelref.Node{{Key: data}, {Key: fn}, {Key: task}},
		Edges: []modelref.Edge{
			{From: fn, To: traceRef(data), Relation: "reads"},
			{From: task, To: traceRef(fn), Relation: "triggers"},
		},
	}
	rules := []modelref.Rule{
		{Relation: "reads", FromKind: "function", ToKind: "data"},
		{Relation: "triggers", FromKind: "task", ToKind: "function"},
	}
	return g, rules, fn
}

func TestTraceDirectIncomingOutgoingAndSummary(t *testing.T) {
	g, rules, fn := traceGraph()
	result, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "both"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || result.Unresolved != 0 || len(result.Rows) != 2 ||
		result.NextOffset != nil || result.Authority != "candidate-read-only" ||
		result.SnapshotFingerprint == "" || result.RuleFingerprint != modelref.RuleFingerprint(rules) {
		t.Fatalf("unexpected trace: %+v", result)
	}
	for _, row := range result.Rows {
		if row.Status != "resolved" || row.ResolvedTo == nil || row.Issue != "" {
			t.Fatalf("unresolved declaration mistaken for resolved: %+v", row)
		}
	}
	for _, direction := range []string{"incoming", "outgoing"} {
		got, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: direction})
		if err != nil || got.Total != 1 || len(got.Rows) != 1 {
			t.Fatalf("%s: %+v %v", direction, got, err)
		}
		if direction == "outgoing" && got.Rows[0].Edge.Relation != "reads" ||
			direction == "incoming" && got.Rows[0].Edge.Relation != "triggers" {
			t.Fatalf("%s included wrong direction: %+v", direction, got.Rows[0])
		}
	}
	summary, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Mode: "summary"})
	if err != nil || summary.Total != 2 || len(summary.Rows) != 0 {
		t.Fatalf("summary lost counts or leaked full data: %+v %v", summary, err)
	}
}

func TestTraceDirectDoesNotInventReferences(t *testing.T) {
	g, rules, fn := traceGraph()
	other := traceKey("other", "data", "order-line")
	g.Nodes = append(g.Nodes, modelref.Node{Key: other})
	g.Edges = append(g.Edges,
		modelref.Edge{From: fn, To: modelref.Ref{Subject: other.Subject}, Relation: "reads"},
		modelref.Edge{From: fn, To: modelref.Ref{Scope: "orders", Subject: subject.Subject{Kind: "data", ID: "missing"}}, Relation: "reads"},
		modelref.Edge{From: fn, To: traceRef(other), Relation: "writes"},
		modelref.Edge{From: fn, To: traceRef(other), Relation: "reads", Evidence: []modelref.Evidence{{Authority: "source-observation"}}},
	)
	got, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "outgoing"})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, row := range got.Rows {
		if row.Status == "unresolved" {
			codes[row.Issue] = true
			if row.ResolvedTo != nil {
				t.Fatalf("failed edge was assigned a target: %+v", row)
			}
		}
	}
	if got.Total != 5 || got.Unresolved != 4 ||
		!codes["ambiguous-reference"] || !codes["missing-reference"] ||
		!codes["invalid-edge-type"] || !codes["unbound-source-evidence"] {
		t.Fatalf("missing fail-closed diagnostics: %+v", got)
	}
}

func TestTraceDirectScopedTargetAndCrossScope(t *testing.T) {
	g, rules, fn := traceGraph()
	other := traceKey("reuse", "data", "order-line")
	g.Nodes = append(g.Nodes, modelref.Node{Key: other})
	g.Edges = append(g.Edges, modelref.Edge{From: fn, To: traceRef(other), Relation: "reads"})
	got, err := modelref.TraceDirect(g, rules, other, modelref.TraceOptions{Direction: "incoming"})
	if err != nil || got.Total != 1 || got.Unresolved != 0 ||
		got.Rows[0].ResolvedTo == nil || *got.Rows[0].ResolvedTo != other {
		t.Fatalf("scope identity was lost: %+v %v", got, err)
	}
	main, err := modelref.TraceDirect(g, rules, traceKey("orders", "data", "order-line"), modelref.TraceOptions{Direction: "incoming"})
	if err != nil || main.Total != 1 || *main.Rows[0].ResolvedTo == other {
		t.Fatalf("cross-scope incoming edge leaked: %+v %v", main, err)
	}
}

func TestTraceDirectPaginationAndDeterministicOrder(t *testing.T) {
	g, rules, fn := traceGraph()
	g.Edges = append(g.Edges, modelref.Edge{From: fn, To: traceRef(traceKey("orders", "data", "order-line")), Relation: "reads",
		Evidence: []modelref.Evidence{{Authority: "proposal", Context: "one"}}})
	all, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	first, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Limit: 1})
	if err != nil || first.NextOffset == nil || *first.NextOffset != 1 || len(first.Rows) != 1 {
		t.Fatalf("page 1: %+v %v", first, err)
	}
	second, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Offset: *first.NextOffset, Limit: 1})
	if err != nil || second.NextOffset == nil || *second.NextOffset != 2 {
		t.Fatalf("page 2: %+v %v", second, err)
	}
	last, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Offset: *second.NextOffset, Limit: 1})
	if err != nil || last.NextOffset != nil || len(last.Rows) != 1 {
		t.Fatalf("page 3: %+v %v", last, err)
	}
	if first.SnapshotFingerprint != second.SnapshotFingerprint ||
		second.SnapshotFingerprint != last.SnapshotFingerprint {
		t.Fatalf("pagination changed snapshot without warning")
	}
	gChanged := g
	gChanged.Nodes = append([]modelref.Node(nil), g.Nodes...)
	gChanged.Nodes[0].Version = "2"
	_, err = modelref.TraceDirect(gChanged, rules, fn, modelref.TraceOptions{
		Offset: *first.NextOffset, Limit: 1, ExpectedFingerprint: first.SnapshotFingerprint,
	})
	var stale *modelref.TraceError
	if !errors.As(err, &stale) || stale.Code != "STALE_SNAPSHOT" {
		t.Fatalf("stale continuation was silently accepted: %v", err)
	}
	withPrecondition, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{
		Offset: *first.NextOffset, Limit: 1, ExpectedFingerprint: first.SnapshotFingerprint,
	})
	if err != nil || !reflect.DeepEqual(withPrecondition.Rows, second.Rows) {
		t.Fatalf("valid continuation differs: %+v %v", withPrecondition, err)
	}
	// Continuation is bound to the query selector as well as Graph and Rule
	// contents. It must not mix rows from another subject or direction.
	for _, alternative := range []struct {
		key     modelref.Key
		options modelref.TraceOptions
	}{
		{fn, modelref.TraceOptions{
			Direction: "incoming", Mode: "full", Offset: 1, Limit: 1,
			ExpectedFingerprint: first.SnapshotFingerprint,
		}},
		{traceKey("orders", "data", "order-line"), modelref.TraceOptions{
			Direction: "both", Mode: "full", Offset: 1, Limit: 1,
			ExpectedFingerprint: first.SnapshotFingerprint,
		}},
	} {
		_, err := modelref.TraceDirect(g, rules, alternative.key, alternative.options)
		var mismatch *modelref.TraceError
		if !errors.As(err, &mismatch) || mismatch.Code != "STALE_SNAPSHOT" {
			t.Fatalf("different trace selector reused a continuation token: %+v err=%v", alternative, err)
		}
	}
	combined := append(append([]modelref.TraceRow{}, first.Rows...), second.Rows...)
	combined = append(combined, last.Rows...)
	if !reflect.DeepEqual(all.Rows, combined) {
		t.Fatalf("pagination omitted or reordered entries")
	}
	for i, j := 0, len(g.Edges)-1; i < j; i, j = i+1, j-1 {
		g.Edges[i], g.Edges[j] = g.Edges[j], g.Edges[i]
	}
	reordered, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{})
	if err != nil || !reflect.DeepEqual(all.Rows, reordered.Rows) {
		t.Fatalf("insertion order changed trace: %+v %v", reordered, err)
	}
	if all.Total != 3 || all.Unresolved != 0 {
		t.Fatalf("full trace truncated: %+v", all)
	}
}

func TestTraceDirectRejectsUnknownOrAmbiguousSubjectAndOptions(t *testing.T) {
	g, rules, fn := traceGraph()
	_, err := modelref.TraceDirect(g, rules, traceKey("orders", "data", "absent"), modelref.TraceOptions{})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing identity: %v", err)
	}
	g.Nodes = append(g.Nodes, modelref.Node{Key: fn})
	_, err = modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{})
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("duplicate identity: %v", err)
	}
	g.Nodes = g.Nodes[:len(g.Nodes)-1]
	for _, options := range []modelref.TraceOptions{
		{Direction: "sideways"}, {Mode: "digest"}, {Offset: -1},
		{Limit: -1}, {Mode: "summary", Limit: 1},
	} {
		if _, err := modelref.TraceDirect(g, rules, fn, options); err == nil {
			t.Fatalf("invalid trace options accepted: %+v", options)
		}
	}
	g.Schema = "other"
	if _, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{}); err == nil {
		t.Fatal("unsupported graph version accepted")
	}
}

func TestTraceDirectConflictingRuleIsUnresolved(t *testing.T) {
	g, rules, fn := traceGraph()
	rules = append(rules, modelref.Rule{
		Relation: "reads", FromKind: "function", ToKind: "data", AcyclicGroup: "changes-meaning",
	})
	got, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "outgoing"})
	if err != nil || got.Unresolved != 1 || got.Rows[0].Issue != "conflicting-rule" ||
		got.Rows[0].ResolvedTo != nil {
		t.Fatalf("conflicting meaning was accepted: %+v %v", got, err)
	}
}

func TestTraceDirectNormalizesEvidenceWithoutMutatingGraph(t *testing.T) {
	g, rules, fn := traceGraph()
	first := modelref.Evidence{Authority: "proposal", Context: "a"}
	second := modelref.Evidence{Authority: "proposal", Context: "b"}
	g.Edges[0].Evidence = []modelref.Evidence{second, first}
	before := append([]modelref.Evidence(nil), g.Edges[0].Evidence...)
	a, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "outgoing"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, g.Edges[0].Evidence) {
		t.Fatalf("trace mutated its input graph: %+v", g.Edges[0].Evidence)
	}
	g.Edges[0].Evidence[0], g.Edges[0].Evidence[1] =
		g.Edges[0].Evidence[1], g.Edges[0].Evidence[0]
	b, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "outgoing"})
	if err != nil || !reflect.DeepEqual(a.Rows, b.Rows) {
		t.Fatalf("unordered evidence changed trace output: %+v versus %+v (%v)", a, b, err)
	}
}

func TestTraceDirectLargeCompleteAndSummary(t *testing.T) {
	g, rules, fn := traceGraph()
	const extra = 4096
	for i := 0; i < extra; i++ {
		g.Edges = append(g.Edges, modelref.Edge{
			From: fn, To: traceRef(traceKey("orders", "data", "order-line")),
			Relation: "reads",
			Evidence: []modelref.Evidence{{Authority: "proposal", Context: fmt.Sprintf("case-%04d", i)}},
		})
	}
	full, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Mode: "full"})
	if err != nil || full.Total != extra+2 || len(full.Rows) != extra+2 || full.NextOffset != nil {
		t.Fatalf("full trace silently truncated: total=%d rows=%d err=%v", full.Total, len(full.Rows), err)
	}
	summary, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Mode: "summary"})
	if err != nil || summary.Total != full.Total || summary.Unresolved != full.Unresolved ||
		len(summary.Rows) != 0 || summary.SnapshotFingerprint != full.SnapshotFingerprint {
		t.Fatalf("summary diverged from full trace: %+v err=%v", summary, err)
	}
}

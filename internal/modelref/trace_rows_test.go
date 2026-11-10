package modelref_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"tei/internal/modelref"
)

// TestFullScanAlwaysCarriesRows pins #3693: full mode must distinguish "this
// subject declares no references" from "this response has no rows field". Before
// this, `rows` was behind omitempty, so a zero-row full scan omitted the field
// entirely and a reader could only guess.
func TestFullScanAlwaysCarriesRows(t *testing.T) {
	// A graph whose only subject has no edges.
	key := traceKey("c2c", "data", "order")
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: key}}}

	full, err := modelref.TraceDirect(graph, nil, key, modelref.TraceOptions{Mode: "full"})
	if err != nil {
		t.Fatalf("full scan: %v", err)
	}
	if full.Total != 0 {
		t.Fatalf("fixture should have no edges, total=%d", full.Total)
	}
	raw, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	rows, present := fields["rows"]
	if !present {
		t.Fatalf("full mode with zero rows omitted the field: %s", raw)
	}
	if string(rows) != "[]" {
		t.Fatalf("rows = %s, want an empty array", rows)
	}
}

// Summary keeps the field absent: it returns counts only, and its presence would
// suggest rows were retrieved.
func TestSummaryOmitsRows(t *testing.T) {
	key := traceKey("c2c", "data", "order")
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: key}}}

	summary, err := modelref.TraceDirect(graph, nil, key, modelref.TraceOptions{Mode: "summary"})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, present := fields["rows"]; present {
		t.Fatalf("summary carried a rows field: %s", raw)
	}
	if _, present := fields["total"]; !present {
		t.Fatalf("summary lost its count: %s", raw)
	}
}

// A non-empty page keeps its rows and its field order, so forcing the field for
// the empty case does not change bytes for the populated case.
func TestFullScanWithRowsKeepsFieldOrder(t *testing.T) {
	key := traceKey("s", "data", "order")
	other := traceKey("s", "data", "customer")
	graph := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes:  []modelref.Node{{Key: key}, {Key: other}},
		Edges: []modelref.Edge{{
			From: other, To: modelref.Ref{Scope: key.Scope, Subject: key.Subject}, Relation: "reads",
		}},
	}
	rules := []modelref.Rule{{Relation: "reads", FromKind: "data", ToKind: "data"}}

	result, err := modelref.TraceDirect(graph, rules, key, modelref.TraceOptions{Mode: "full", Direction: "incoming"})
	if err != nil {
		t.Fatalf("full scan: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(result.Rows))
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	// The declaration order is part of the bytes other tools compare.
	want := []string{"schemaVersion", "graphFingerprint", "snapshotFingerprint", "ruleFingerprint",
		"subject", "direction", "mode", "authority", "total", "unresolved", "rows"}
	var got []string
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		t.Fatal(err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		t.Fatalf("not an object: %s", raw)
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, keyTok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	for i, name := range want {
		if i >= len(got) || got[i] != name {
			t.Fatalf("field order drifted at %d: got %v, want prefix %v", i, got, want)
		}
	}
}

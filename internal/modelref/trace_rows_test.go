package modelref

import (
	"encoding/json"
	"testing"
)

// marshalTrace returns the decoded JSON object for a TraceResult.
func marshalTrace(t *testing.T, result TraceResult) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("trace is not a JSON object: %v (%s)", err, raw)
	}
	return doc
}

// A full trace answers "all rows". Zero rows is an empty list, not an absent
// field: the field carries omitempty because summary mode must not emit rows at
// all, and omitempty drops a zero-length slice as well as a nil one. A reader
// that iterates result.rows could not tell "no references" from "this version
// has no such field" (#3693).
func TestFullTraceAlwaysEmitsRows(t *testing.T) {
	cases := []struct {
		name   string
		result TraceResult
	}{
		{"nil rows", TraceResult{Mode: "full", Total: 0}},
		{"empty rows", TraceResult{Mode: "full", Total: 0, Rows: []TraceRow{}}},
		{"one row", TraceResult{Mode: "full", Total: 1, Rows: []TraceRow{{Status: "resolved"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := marshalTrace(t, tc.result)
			raw, ok := doc["rows"]
			if !ok {
				t.Fatalf("rows is absent for a full trace: %v", doc)
			}
			if string(raw) == "null" {
				t.Fatalf("rows was written as null")
			}
			var rows []TraceRow
			if err := json.Unmarshal(raw, &rows); err != nil {
				t.Fatalf("rows is not an array: %v (%s)", err, raw)
			}
			if len(rows) != len(tc.result.Rows) {
				t.Fatalf("rows=%d, want %d", len(rows), len(tc.result.Rows))
			}
		})
	}
}

// Summary mode counts the whole graph without allocating rows, so it must not
// emit the field. Emitting rows:[] there would claim "no rows" for a page that
// deliberately did not build any.
func TestSummaryTraceOmitsRows(t *testing.T) {
	doc := marshalTrace(t, TraceResult{Mode: "summary", Total: 7})
	if raw, ok := doc["rows"]; ok {
		t.Fatalf("summary emitted rows: %s", raw)
	}
}

// The marshaler must not change any other field. This compares the JSON of a
// populated result against the struct's own default marshaling, so a dropped or
// renamed field is caught here rather than by a downstream reader.
func TestMarshalTraceKeepsEveryOtherField(t *testing.T) {
	next := 2
	original := TraceResult{
		SchemaVersion:       "tei.modelref.trace/v0",
		GraphFingerprint:    "sha256:graph",
		SnapshotFingerprint: "sha256:legacy",
		RuleFingerprint:     "sha256:rules",
		Subject:             Key{Scope: "s"},
		Direction:           "both",
		Mode:                "full",
		Authority:           "candidate-read-only",
		Total:               1,
		Unresolved:          0,
		Offset:              0,
		Rows:                []TraceRow{{Status: "resolved"}},
		NextOffset:          &next,
		Continuation:        &TraceContinuation{Fingerprint: "sha256:outer", Field: "result.snapshotFingerprint"},
	}
	got := marshalTrace(t, original)

	type alias TraceResult
	plainRaw, err := json.Marshal(alias(original))
	if err != nil {
		t.Fatal(err)
	}
	var plain map[string]json.RawMessage
	if err := json.Unmarshal(plainRaw, &plain); err != nil {
		t.Fatal(err)
	}
	for key, want := range plain {
		gotRaw, ok := got[key]
		if !ok {
			t.Fatalf("field %q disappeared: %v", key, got)
		}
		if string(gotRaw) != string(want) {
			t.Fatalf("field %q changed: got %s want %s", key, gotRaw, want)
		}
	}
	if _, ok := got["rows"]; !ok {
		t.Fatal("rows missing")
	}
}

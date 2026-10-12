package exampleslide

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCandidateKeyProbeDetectsCollisionWithoutAssumingPhysicalRowIdentity(t *testing.T) {
	catalog, example, _ := fdProbeFixture()
	h := CandidateKeyHypothesis{
		ID: "order-line-business-id", Table: "order_line",
		Fields: []string{"order_id", "line_no"},
	}
	got, err := ProbeBaseCandidateKeys(catalog, example, []CandidateKeyHypothesis{h})
	if err != nil {
		t.Fatal(err)
	}
	want := []FDRecordPair{{LeftID: "record-a", RightID: "record-b"}}
	if len(got) != 1 || got[0].Result != "contradiction" ||
		got[0].Reason != "duplicate-candidate-key-in-example" ||
		!reflect.DeepEqual(got[0].Collisions, want) ||
		got[0].IndeterminatePairs != 0 || got[0].DistinctPairs != 0 {
		t.Fatalf("business key duplicate missed: %+v", got)
	}
	// The physical row IDs differ. Reordering both the records and
	// the declared compound key may not change the collision evidence.
	example.Base[0], example.Base[1] = example.Base[1], example.Base[0]
	h.Fields[0], h.Fields[1] = h.Fields[1], h.Fields[0]
	reordered, err := ProbeBaseCandidateKeys(catalog, example, []CandidateKeyHypothesis{h})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, reordered) {
		t.Fatalf("source order altered candidate-key evidence: %+v vs %+v", got, reordered)
	}
}

func TestCandidateKeyProbeNeverAcceptsKeyFromSmallOrCompleteClaimedExample(t *testing.T) {
	catalog, example, _ := fdProbeFixture()
	h := CandidateKeyHypothesis{ID: "rowid-is-proposed-not-accepted", Table: "order_line", Fields: []string{"row_id"}}
	for _, records := range [][]Record{example.Base, example.Base[:1], nil} {
		in := example
		in.Base = records
		in.CompleteTables = []string{"order_line"}
		got, err := ProbeBaseCandidateKeys(catalog, in, []CandidateKeyHypothesis{h})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Result != "unresolved" ||
			got[0].Reason != "no-confirmed-key-collision-in-example" ||
			len(got[0].Collisions) != 0 {
			t.Fatalf("uniqueness in one sample was promoted to a key: %+v", got)
		}
	}
}

func TestCandidateKeyProbeSeparatesUnknownKeysFromProvenDuplicates(t *testing.T) {
	catalog, example, _ := fdProbeFixture()
	example.Base = append(example.Base, Record{
		ID: "record-c", Table: "order_line",
		Values: map[string]json.RawMessage{
			"row_id": json.RawMessage(`"pk-c"`),
			"order_id": json.RawMessage(`"order-1"`),
		},
	})
	h := CandidateKeyHypothesis{ID: "candidate", Table: "order_line", Fields: []string{"order_id", "line_no"}}
	got, err := ProbeBaseCandidateKeys(catalog, example, []CandidateKeyHypothesis{h})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Result != "contradiction" ||
		!reflect.DeepEqual(got[0].Collisions, []FDRecordPair{{LeftID: "record-a", RightID: "record-b"}}) ||
		!reflect.DeepEqual(got[0].UnusableRecordIDs, []string{"record-c"}) ||
		got[0].IndeterminatePairs != 2 {
		t.Fatalf("lost known collision or unknown pair: %+v", got)
	}
}

func TestCandidateKeyProbeUsesKnownDifferenceBeforeUnknownComponent(t *testing.T) {
	catalog, example, _ := fdProbeFixture()
	delete(example.Base[0].Values, "line_no")
	example.Base[0].Values["order_id"] = json.RawMessage(`"another-order"`)
	h := CandidateKeyHypothesis{ID: "candidate", Table: "order_line", Fields: []string{"order_id", "line_no"}}
	got, err := ProbeBaseCandidateKeys(catalog, example, []CandidateKeyHypothesis{h})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "unresolved" || got[0].IndeterminatePairs != 0 ||
		got[0].DistinctPairs != 1 || len(got[0].Collisions) != 0 ||
		!reflect.DeepEqual(got[0].UnusableRecordIDs, []string{"record-b"}) {
		t.Fatalf("known mismatch incorrectly treated as a collision: %+v", got)
	}
}

func TestCandidateKeyProbeDoesNotCompareOpaqueBusinessIdentifiers(t *testing.T) {
	catalog, example, _ := fdProbeFixture()
	catalog.Tables["order_line"] = Table{Fields: []Field{
		{ID: "row_id", Type: "string", Required: true},
		{ID: "order_id", Type: "opaque", Required: true},
		{ID: "line_no", Type: "integer"},
		{ID: "item", Type: "string"},
	}}
	h := CandidateKeyHypothesis{ID: "candidate", Table: "order_line", Fields: []string{"order_id", "line_no"}}
	got, err := ProbeBaseCandidateKeys(catalog, example, []CandidateKeyHypothesis{h})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "unresolved" || len(got[0].Collisions) != 0 ||
		got[0].IndeterminatePairs != 1 ||
		!reflect.DeepEqual(got[0].UnusableRecordIDs, []string{"record-a", "record-b"}) {
		t.Fatalf("opaque key compared from JSON bytes: %+v", got)
	}
}

func TestCandidateKeyProbeRejectsInvalidInputAndDuplicateHypotheses(t *testing.T) {
	type mutation struct {
		name  string
		apply func(*Catalog, *Example, *CandidateKeyHypothesis)
	}
	tests := []mutation{
		{"stale-revision", func(_ *Catalog, e *Example, _ *CandidateKeyHypothesis) { e.ModelRevision = "old" }},
		{"missing-key-fields", func(_ *Catalog, _ *Example, h *CandidateKeyHypothesis) { h.Fields = nil }},
		{"unknown-key-field", func(_ *Catalog, _ *Example, h *CandidateKeyHypothesis) { h.Fields = []string{"not-a-field"} }},
		{"duplicate-key-field", func(_ *Catalog, _ *Example, h *CandidateKeyHypothesis) { h.Fields = []string{"row_id", "row_id"} }},
		{"unknown-table", func(_ *Catalog, _ *Example, h *CandidateKeyHypothesis) { h.Table = "missing" }},
		{"empty-hypothesis-id", func(_ *Catalog, _ *Example, h *CandidateKeyHypothesis) { h.ID = "" }},
		{"duplicate-record-id", func(_ *Catalog, e *Example, _ *CandidateKeyHypothesis) { e.Base[0].ID = e.Base[1].ID }},
		{"invalid-record-field", func(_ *Catalog, e *Example, _ *CandidateKeyHypothesis) {
			e.Base[0].Values["undeclared"] = json.RawMessage(`"value"`)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, e, _ := fdProbeFixture()
			h := CandidateKeyHypothesis{ID: "k", Table: "order_line", Fields: []string{"order_id", "line_no"}}
			tc.apply(&c, &e, &h)
			if _, err := ProbeBaseCandidateKeys(c, e, []CandidateKeyHypothesis{h}); err == nil {
				t.Fatal("invalid key hypothesis or example was accepted")
			}
		})
	}
	c, e, _ := fdProbeFixture()
	h := CandidateKeyHypothesis{ID: "k", Table: "order_line", Fields: []string{"order_id"}}
	if _, err := ProbeBaseCandidateKeys(c, e, []CandidateKeyHypothesis{h, h}); err == nil {
		t.Fatal("repeated hypothesis ID was accepted")
	}
}

func TestCandidateKeyProbeHypothesisOrderAndInputImmutability(t *testing.T) {
	c, e, _ := fdProbeFixture()
	before, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	h1 := CandidateKeyHypothesis{ID: "z", Table: "order_line", Fields: []string{"order_id", "line_no"}}
	h2 := CandidateKeyHypothesis{ID: "a", Table: "order_line", Fields: []string{"row_id"}}
	left, err := ProbeBaseCandidateKeys(c, e, []CandidateKeyHypothesis{h1, h2})
	if err != nil {
		t.Fatal(err)
	}
	right, err := ProbeBaseCandidateKeys(c, e, []CandidateKeyHypothesis{h2, h1})
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 || !reflect.DeepEqual(left, right) ||
		left[0].HypothesisID != "a" || !reflect.DeepEqual(before, after) {
		t.Fatalf("unstable projection or source modification: %+v / %+v", left, right)
	}
}

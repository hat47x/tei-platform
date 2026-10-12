package exampleslide

import (
	"encoding/json"
	"reflect"
	"testing"
)

func fdProbeFixture() (Catalog, Example, FDHypothesis) {
	catalog := Catalog{
		Revision: "fd-r1",
		Tables: map[string]Table{
			"order_line": {Fields: []Field{
				{ID: "row_id", Type: "string", Required: true},
				{ID: "order_id", Type: "string", Required: true},
				{ID: "line_no", Type: "integer"},
				{ID: "item", Type: "string"},
			}},
		},
	}
	example := Example{ID: "synthetic-order-lines", ModelRevision: "fd-r1", Base: []Record{
		{ID: "record-b", Table: "order_line", Values: map[string]json.RawMessage{
			"row_id": `"pk-b"`, "order_id": `"order-1"`, "line_no": `9007199254740993.0`, "item": `"item-B"`,
		}},
		{ID: "record-a", Table: "order_line", Values: map[string]json.RawMessage{
			"row_id": `"pk-a"`, "order_id": `"order-1"`, "line_no": `9007199254740993`, "item": `"item-A"`,
		}},
	}}
	hypothesis := FDHypothesis{
		ID: "line-identity", Table: "order_line",
		Determinants: []string{"order_id", "line_no"}, Dependents: []string{"item"},
	}
	return catalog, example, hypothesis
}

func TestFDProbeFindsCompositeCounterexampleDespiteDifferentPhysicalIDs(t *testing.T) {
	catalog, example, hypothesis := fdProbeFixture()
	got, err := ProbeBaseFunctionalDependencies(catalog, example, []FDHypothesis{hypothesis})
	if err != nil {
		t.Fatal(err)
	}
	want := []FDRecordPair{{LeftID: "record-a", RightID: "record-b"}}
	if len(got) != 1 || got[0].Result != "contradiction" ||
		got[0].Reason != "equal-determinants-different-dependents" ||
		!reflect.DeepEqual(got[0].Counterexamples, want) {
		t.Fatalf("lost composite FD counterexample: %+v", got)
	}
	// Neither sorting the source rows nor rearranging determinant order may
	// change the resulting witness (a separate physical row ID is irrelevant).
	example.Base[0], example.Base[1] = example.Base[1], example.Base[0]
	hypothesis.Determinants[0], hypothesis.Determinants[1] =
		hypothesis.Determinants[1], hypothesis.Determinants[0]
	reordered, err := ProbeBaseFunctionalDependencies(catalog, example, []FDHypothesis{hypothesis})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, reordered) {
		t.Fatalf("order changes evidence: before=%+v after=%+v", got, reordered)
	}
}

func TestFDProbeDoesNotProveRuleFromConsistentOrPartialExample(t *testing.T) {
	catalog, example, hypothesis := fdProbeFixture()
	example.Base[1].Values["item"] = json.RawMessage(`"item-B"`)
	for _, base := range [][]Record{example.Base, example.Base[:1], nil} {
		snapshot := example
		snapshot.Base = base
		got, err := ProbeBaseFunctionalDependencies(catalog, snapshot, []FDHypothesis{hypothesis})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Result != "unresolved" ||
			got[0].Reason != "no-counterexample-in-example" || len(got[0].Counterexamples) != 0 {
			t.Fatalf("sample was mistaken for an accepted FD: %+v", got)
		}
	}
}

func TestFDProbeKnownDifferenceOutweighsMissingDeterminant(t *testing.T) {
	catalog, example, hypothesis := fdProbeFixture()
	delete(example.Base[0].Values, "line_no")
	example.Base[0].Values["order_id"] = json.RawMessage(`"another-order"`)
	got, err := ProbeBaseFunctionalDependencies(catalog, example, []FDHypothesis{hypothesis})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "unresolved" || got[0].IndeterminatePairs != 0 ||
		len(got[0].Counterexamples) != 0 {
		t.Fatalf("known determinant mismatch should rule out pair: %+v", got)
	}
}

func TestFDProbeMissingOrOpaqueFieldsCannotProveConflict(t *testing.T) {
	catalog, example, hypothesis := fdProbeFixture()
	delete(example.Base[0].Values, "line_no")
	got, err := ProbeBaseFunctionalDependencies(catalog, example, []FDHypothesis{hypothesis})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "unresolved" || got[0].IndeterminatePairs != 1 {
		t.Fatalf("missing determinant was assumed equal: %+v", got)
	}
	catalog, example, hypothesis = fdProbeFixture()
	catalog.Tables["order_line"] = Table{Fields: []Field{
		{ID: "row_id", Type: "string", Required: true},
		{ID: "order_id", Type: "opaque", Required: true},
		{ID: "line_no", Type: "integer"},
		{ID: "item", Type: "string"},
	}}
	got, err = ProbeBaseFunctionalDependencies(catalog, example, []FDHypothesis{hypothesis})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Result != "unresolved" || got[0].IndeterminatePairs != 1 {
		t.Fatalf("opaque determinant was compared without domain semantics: %+v", got)
	}
}

func TestFDProbeRejectsInvalidDeclarationsAndUnvalidatedRecords(t *testing.T) {
	type change struct {
		name string
		apply func(*Catalog, *Example, *FDHypothesis)
	}
	cases := []change{
		{"stale", func(_ *Catalog, e *Example, _ *FDHypothesis) { e.ModelRevision = "stale" }},
		{"unknown table", func(_ *Catalog, _ *Example, h *FDHypothesis) { h.Table = "missing" }},
		{"missing determinant", func(_ *Catalog, _ *Example, h *FDHypothesis) { h.Determinants = nil }},
		{"missing dependent", func(_ *Catalog, _ *Example, h *FDHypothesis) { h.Dependents = nil }},
		{"unknown field", func(_ *Catalog, _ *Example, h *FDHypothesis) { h.Dependents = []string{"unknown"} }},
		{"repeated field", func(_ *Catalog, _ *Example, h *FDHypothesis) { h.Dependents = []string{"item", "item"} }},
		{"duplicate records", func(_ *Catalog, e *Example, _ *FDHypothesis) { e.Base[0].ID = e.Base[1].ID }},
		{"unknown value field", func(_ *Catalog, e *Example, _ *FDHypothesis) {
			e.Base[0].Values["ghost"] = json.RawMessage(`"x"`)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, e, h := fdProbeFixture()
			tc.apply(&c, &e, &h)
			if _, err := ProbeBaseFunctionalDependencies(c, e, []FDHypothesis{h}); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	c, e, h := fdProbeFixture()
	if _, err := ProbeBaseFunctionalDependencies(c, e, []FDHypothesis{h, h}); err == nil {
		t.Fatal("duplicate hypothesis ID accepted")
	}
}

func TestFDProbeKeepsIndependentHypothesesInStableOrder(t *testing.T) {
	c, e, h := fdProbeFixture()
	h2 := FDHypothesis{ID: "a-synthetic-question", Table: "order_line",
		Determinants: []string{"row_id"}, Dependents: []string{"item"}}
	a, err := ProbeBaseFunctionalDependencies(c, e, []FDHypothesis{h, h2})
	if err != nil {
		t.Fatal(err)
	}
	b, err := ProbeBaseFunctionalDependencies(c, e, []FDHypothesis{h2, h})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || !reflect.DeepEqual(a, b) || a[0].HypothesisID != h2.ID ||
		a[0].Result != "unresolved" {
		t.Fatalf("hypothesis order or independent resolution is wrong: %+v vs %+v", a, b)
	}
}

package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestBaseOnlyEdgeSeedsOldDependentWithoutReintroducingDeletedPath(t *testing.T) {
	v := input{
		Schema:              inputSchema,
		Scope:               "native-pdp",
		BaseRevision:        "R0",
		CandidateRevision:   "R1",
		DeclaredScopeClosed: true,
		Nodes: []node{
			{ID: "copy.source", Layer: "content"},
			{ID: "offer.rule", Layer: "business"},
			{ID: "native.pdp", Layer: "presentation"},
		},
		// The candidate no longer has copy.source -> offer.rule.
		Edges:         []edge{{From: "offer.rule", To: "native.pdp"}},
		BaseOnlyEdges: []edge{{From: "copy.source", To: "offer.rule"}},
		Required:      []string{"native.pdp"},
	}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	// The old dependent and its candidate dependents must be re-evaluated,
	// even with no independently changed node declared in R1.
	if !reflect.DeepEqual(r.Impacted, []string{"native.pdp", "offer.rule"}) ||
		!reflect.DeepEqual(r.Lanes, []string{"C1", "C3"}) ||
		r.Disposition != "known-interaction-business-or-release-impact" ||
		r.Coverage != "declared-closed-only" {
		t.Fatalf("deleted R0 edge silently hid its affected consumer: %+v", r)
	}
	if !strings.Contains(strings.Join(r.AuthorityLimits, " "), "unverified-claims") {
		t.Fatalf("caller-declared R0 edge treated as verified provenance: %+v", r)
	}

	// A malformed R0-only edge cannot simultaneously exist in the R1
	// candidate; that would make the claimed difference self-contradictory.
	v.BaseOnlyEdges = []edge{{From: "offer.rule", To: "native.pdp"}}
	if _, err := inspect(v); err == nil {
		t.Fatal("edge claimed deleted despite existing in the candidate")
	}
}

func TestBaseOnlyEdgeUnknownAndAmbiguousInputsRemainUnqualified(t *testing.T) {
	v := input{
		Schema:              inputSchema,
		Scope:               "native-pdp",
		BaseRevision:        "R0",
		CandidateRevision:   "R1",
		DeclaredScopeClosed: true,
		Nodes:               []node{{ID: "native.pdp", Layer: "presentation"}},
		Required:            []string{"native.pdp"},
		BaseOnlyEdges:       []edge{{From: "old.source", To: "native.pdp"}},
	}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "partial-or-unverified" ||
		!reflect.DeepEqual(r.Unresolved, []string{"old.source"}) ||
		!reflect.DeepEqual(r.Impacted, []string{"native.pdp"}) {
		t.Fatalf("deleted source without an R1 identity was called complete: %+v", r)
	}
	v.BaseOnlyEdges = []edge{{From: "old.source", To: "missing.consumer"}}
	r, err = inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Unresolved, []string{"missing.consumer", "old.source"}) ||
		r.Coverage != "partial-or-unverified" {
		t.Fatalf("missing deleted-edge dependent was hidden: %+v", r)
	}
	v.BaseOnlyEdges = []edge{
		{From: "old.source", To: "native.pdp"},
		{From: "old.source", To: "native.pdp"},
	}
	if _, err := inspect(v); err == nil {
		t.Fatal("duplicate R0-only edge accepted")
	}
	v.BaseOnlyEdges = []edge{{From: " ", To: "native.pdp"}}
	if _, err := inspect(v); err == nil {
		t.Fatal("blank R0-only edge endpoint accepted")
	}
	v.BaseOnlyEdges = nil
	if _, err := inspect(v); err == nil {
		t.Fatal("no changed node and no old edge was accepted")
	}
}

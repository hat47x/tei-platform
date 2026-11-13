package main

import (
	"reflect"
	"testing"
)

// A deleted edge outside this consumer's Required input set must not
// requalify an unrelated business lane or poison its missing evidence.
func TestDeletedUnrelatedEdgeDoesNotInvalidateRequiredConsumer(t *testing.T) {
	v := input{
		Schema:              inputSchema,
		Scope:               "pdp.main",
		BaseRevision:        "R0",
		CandidateRevision:   "R1",
		DeclaredScopeClosed: true,
		Nodes: []node{
			{ID: "pdp.main", Layer: "presentation"},
			{ID: "unrelated.rule", Layer: "business"},
			{ID: "unrelated.page", Layer: "presentation"},
		},
		Edges: []edge{{From: "unrelated.rule", To: "unrelated.page"}},
		// The old Source is missing in R1, but its removed edge is not
		// upstream of the consumer for this question in either declared world.
		BaseOnlyEdges: []edge{{From: "unrelated.old-source", To: "unrelated.rule"}},
		Required:      []string{"pdp.main"},
	}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Impacted) != 0 || len(r.Lanes) != 0 || len(r.Unresolved) != 0 ||
		r.Coverage != "declared-closed-only" ||
		r.Disposition != "no-known-critical-impact-within-declared-scope" {
		t.Fatalf("unrelated R0-only edge incorrectly changed this consumer: %+v", r)
	}
	// An independent changed content node may still be reported, but the
	// unrelated deleted business edge must remain outside the change.
	v.Nodes = append(v.Nodes, node{ID: "pdp.headline", Layer: "content"})
	v.Changed = []string{"pdp.headline"}
	r, err = inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Impacted, []string{"pdp.headline"}) ||
		!reflect.DeepEqual(r.Lanes, []string{"C0"}) ||
		len(r.Unresolved) != 0 ||
		r.Disposition != "no-known-critical-impact-within-declared-scope" {
		t.Fatalf("unrelated deleted edge over-invalidated changed content: %+v", r)
	}
}

// When two old edges disappear, the old predecessor chain is visible only
// through the caller's explicit R0-only list. The old Consumer must remain
// affected even if candidate edges no longer connect any of those nodes.
func TestDeletedTwoHopChainRetainsRelevantUnknownAndConsumer(t *testing.T) {
	v := input{
		Schema:              inputSchema,
		Scope:               "native-pdp",
		BaseRevision:        "R0",
		CandidateRevision:   "R1",
		DeclaredScopeClosed: true,
		Nodes: []node{
			{ID: "native.pdp", Layer: "presentation"},
			{ID: "old.rule", Layer: "business"},
			{ID: "unrelated.page", Layer: "presentation"},
		},
		BaseOnlyEdges: []edge{
			{From: "removed.source", To: "old.rule"},
			{From: "old.rule", To: "native.pdp"},
			{From: "other.source", To: "unrelated.page"},
		},
		Required: []string{"native.pdp"},
	}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Impacted, []string{"native.pdp", "old.rule"}) ||
		!reflect.DeepEqual(r.Lanes, []string{"C1", "C3"}) ||
		!reflect.DeepEqual(r.Unresolved, []string{"removed.source"}) ||
		r.Coverage != "partial-or-unverified" ||
		r.Disposition != "known-interaction-business-or-release-impact" {
		t.Fatalf("R0-only chain was lost or unrelated chain was included: %+v", r)
	}
}

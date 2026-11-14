package query

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"tei/internal/query/analyzer"
)

// Branch coverage for the #2582 change impact function. Expected values are
// written by hand from the fixture in change_impact_test.go (operations:
// l1 agg_line | l2 d2 join_rows agg_join | h3 d3 outer_join anti | ref_totals proj),
// not taken from the function's output.

func TestRC04OuterJoinSidesAndOtherOperators(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*analyzer.Result)
		claim   string
		changed []string
		want    rcExpected
		prov    []string
	}{
		// RIGHT JOIN: the left input (header) becomes the null-supplying side.
		{"right join makes the left input null-supplying", func(r *analyzer.Result) { r.Operations[8].JoinKind = "right" },
			"anti", []string{"H"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil, []string{"filter-unparsed:H", "outer-join-null-side:H"}},
			[]string{"unmatched-rows-not-absence-proof:H:unknown"}},
		{"right join leaves the right input preserved", func(r *analyzer.Result) { r.Operations[8].JoinKind = "right" },
			"anti", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil, []string{"filter-unparsed:D2"}},
			[]string{"unmatched-rows-not-absence-proof:H:unknown"}},
		// FULL JOIN: both inputs can be null-supplied.
		{"full join, right input changed", func(r *analyzer.Result) { r.Operations[8].JoinKind = "full" },
			"anti", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil, []string{"filter-unparsed:D2", "outer-join-null-side:D2"}},
			[]string{"unmatched-rows-not-absence-proof:D2:unknown", "unmatched-rows-not-absence-proof:H:unknown"}},
		{"full join, left input changed", func(r *analyzer.Result) { r.Operations[8].JoinKind = "full" },
			"anti", []string{"H"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil, []string{"filter-unparsed:H", "outer-join-null-side:H"}},
			[]string{"unmatched-rows-not-absence-proof:D2:unknown", "unmatched-rows-not-absence-proof:H:unknown"}},
		// A cross join is a monotone product, like an inner join.
		{"cross join is not a hazard", func(r *analyzer.Result) { r.Operations[4].JoinKind = "cross" },
			"join_rows", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1", "D2"}, nil, nil}, nil},
		// DISTINCT / LIMIT may drop or reorder rows when input is added.
		{"distinct over D1", func(r *analyzer.Result) {
			r.Operations = append(r.Operations, analyzer.Operation{ID: "dist", Kind: "distinct", ScopeID: "cte:line_totals", Inputs: []string{"l1"}})
		}, "dist", []string{"D1"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, []string{"distinct:D1"}}, nil},
		{"limit over D1", func(r *analyzer.Result) {
			r.Operations = append(r.Operations, analyzer.Operation{ID: "top", Kind: "limit", ScopeID: "cte:line_totals", Inputs: []string{"l1"}, Expressions: []string{"3"}})
		}, "top", []string{"D1"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, []string{"limit:D1"}}, nil},
		// Sorting alone only reorders; it is not reported as non-monotone.
		{"sort is not a hazard", func(r *analyzer.Result) {
			r.Operations = append(r.Operations, analyzer.Operation{ID: "srt", Kind: "sort", ScopeID: "cte:line_totals", Inputs: []string{"l1"}, Expressions: []string{"amount DESC"}})
		}, "srt", []string{"D1"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, nil}, nil},
		// An operator kind the function does not know is never assumed safe.
		{"unknown operator kind", func(r *analyzer.Result) {
			r.Operations = append(r.Operations, analyzer.Operation{ID: "win", Kind: "window", ScopeID: "cte:line_totals", Inputs: []string{"l1"}})
		}, "win", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"unknown-operator:win"}, nil}, nil},
		{"join with one input", func(r *analyzer.Result) { r.Operations[4].Inputs = []string{"l2"} },
			"join_rows", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"join-arity:join_rows"}, nil}, nil},
		{"aggregate without input", func(r *analyzer.Result) { r.Operations[1].Inputs = nil },
			"agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"missing-input:agg_line"}, nil}, nil},
	}
	for _, c := range cases {
		got := AssessChangeImpact(rcIR(c.mutate), ChangeImpactRequest{Query: "rc.subject", Claim: c.claim, Bindings: rcBindings(), Changed: c.changed})
		checkImpact(t, c.name, got, c.want)
		if !reflect.DeepEqual(got.Provisional, c.prov) {
			t.Errorf("%s: provisional=%v want %v", c.name, got.Provisional, c.prov)
		}
	}
}

func TestRC02StructuralFailuresAreUnknown(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*analyzer.Result)
		want   rcExpected
	}{
		{"self cycle", func(r *analyzer.Result) { r.Operations[1].Inputs = []string{"agg_line"} },
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"cycle:agg_line"}, nil}},
		{"duplicate scope id", func(r *analyzer.Result) {
			r.Scopes = append(r.Scopes, analyzer.Scope{ID: "cte:join_sum", ParentID: "scope:root", Kind: "cte", Name: "again"})
		}, rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"scope-hierarchy-ambiguous"}, nil}},
		{"duplicate operation id", func(r *analyzer.Result) {
			r.Operations = append(r.Operations, analyzer.Operation{ID: "l1", Kind: "source", ScopeID: "cte:line_totals", Source: "line", Alias: "z"})
		}, rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"duplicate-node:l1"}, nil}},
	}
	for _, c := range cases {
		got := AssessChangeImpact(rcIR(c.mutate), ChangeImpactRequest{Query: "rc.subject", Claim: "agg_line", Bindings: rcBindings(), Changed: []string{"D2"}})
		checkImpact(t, c.name, got, c.want)
	}

	// A chain longer than the visit budget stops instead of walking on, and
	// reports why. The fixture is a plain filter chain over one bound source.
	const depth = changeImpactVisitBudget + 50
	big := rcIR(func(r *analyzer.Result) {
		prev := "l1"
		for i := 0; i < depth; i++ {
			id := fmt.Sprintf("f%d", i)
			r.Operations = append(r.Operations, analyzer.Operation{ID: id, Kind: "filter", ScopeID: "cte:line_totals", Inputs: []string{prev}, Expressions: []string{"true"}})
			prev = id
		}
	})
	got := AssessChangeImpact(big, ChangeImpactRequest{Query: "rc.subject", Claim: fmt.Sprintf("f%d", depth-1), Bindings: rcBindings(), Changed: []string{"D2"}})
	checkImpact(t, "graph too large", got, rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"graph-too-large"}, nil})
}

func TestRC01FromSubqueryBinding(t *testing.T) {
	sub := func(binding string) func(*analyzer.Result) {
		return func(r *analyzer.Result) {
			r.Scopes = append(r.Scopes, analyzer.Scope{ID: "sub:from", ParentID: "scope:root", Kind: "subquery"})
			r.Operations = append(r.Operations,
				analyzer.Operation{ID: "s_l", Kind: "source", ScopeID: "sub:from", Source: "line", Alias: "s"},
				analyzer.Operation{ID: "s_agg", Kind: "aggregate", ScopeID: "sub:from", Inputs: []string{"s_l"}, Expressions: []string{"SUM(s.amount)"}},
				analyzer.Operation{ID: "ref_sub", Kind: "source", ScopeID: "scope:root", Source: "t", SourceKind: "subquery", Binding: binding},
			)
			r.Operations[11].Inputs = []string{"ref_sub"}
		}
	}
	bindings := func() map[string]string { b := rcBindings(); b["s_l"] = "D1"; return b }
	run := func(mutate func(*analyzer.Result), changed ...string) ChangeImpact {
		return AssessChangeImpact(rcIR(mutate), ChangeImpactRequest{Query: "rc.subject", Claim: "proj", Bindings: bindings(), Changed: changed})
	}

	checkImpact(t, "from-subquery, D1 changed", run(sub("sub:from"), "D1"),
		rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, []string{"aggregate:D1", "scope-binding:D1"}})
	// The CTEs reading D2 are not reached from this root.
	checkImpact(t, "from-subquery, D2 changed", run(sub("sub:from"), "D2"),
		rcExpected{ImpactRetain, RefreshNone, []string{"D1"}, nil, nil})
	// A binding the IR cannot resolve is never walked through. The subquery
	// scope is then bound by no source occurrence, so under the confirmed rule
	// it counts as a possible dependency and its D1 read is reported too.
	checkImpact(t, "from-subquery with unresolvable binding", run(sub("sub:missing"), "D1"),
		rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil,
			[]string{"binding-unresolved:ref_sub", "unlinked-nested-dependency:D1"}, nil})
}

// The Python QX-16 fixture declares, per claim node, which input collections
// it reads. That table was written by hand and is independent of the Go IR,
// so the Go answer for the matching IR node must agree with it.
func TestChangeImpactDependsOnAgreesWithQX16Nodes(t *testing.T) {
	src, err := os.ReadFile("../../plan/fixtures/query-workbench-f0/explain_qx16.py")
	if err != nil {
		t.Fatalf("QX-16 fixture is required for this cross-check: %v", err)
	}
	re := regexp.MustCompile(`(?s)"(\w+)":\s*\{[^}]*?"inputCollections":\s*\[([^\]]*)\]`)
	nodes := map[string][]string{}
	for _, m := range re.FindAllStringSubmatch(string(src[strings.Index(string(src), "NODES = {"):]), -1) {
		var cols []string
		for _, c := range strings.Split(m[2], ",") {
			if c = strings.Trim(strings.TrimSpace(c), `"`); c != "" {
				cols = append(cols, c)
			}
		}
		sort.Strings(cols)
		nodes[m[1]] = cols
	}
	// Names are the fixture's; contexts are this test's own mapping.
	collection := map[string]string{"line": "D1", "delivery": "D2", "header": "H"}
	irNode := map[string]string{"line_sum": "agg_line", "join_pairs": "join_rows", "join_sum": "agg_join", "left_unmatched": "anti"}
	ir := rcIR(nil)
	for name, claim := range irNode {
		cols, ok := nodes[name]
		if !ok {
			t.Fatalf("QX-16 node %q not found in the fixture", name)
		}
		var want []string
		for _, c := range cols {
			want = append(want, collection[c])
		}
		sort.Strings(want)
		got := AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: claim, Bindings: rcBindings()})
		if !reflect.DeepEqual(got.DependsOn, want) {
			t.Errorf("%s (%s): dependsOn=%v, QX-16 inputCollections=%v", name, claim, got.DependsOn, want)
		}
	}
}

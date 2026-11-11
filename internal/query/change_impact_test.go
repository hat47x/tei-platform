package query

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/query/analyzer"
)

// Synthetic #2582 fixture: header H with independent child contexts D1 (line)
// and D2 (delivery). The analyzer response is hand-written; it exercises the
// real DeriveFromAnalysis builder but is not evidence about the SQL worker.
//
//	cte:line_totals  SUM over line                      -> D1 only
//	cte:join_sum     line JOIN delivery, then SUM       -> D1 x D2
//	cte:unmatched    header LEFT JOIN delivery, IS NULL -> H, D2 (null side)
//	root             projection over line_totals
func rcResult() analyzer.Result {
	return analyzer.Result{
		ProtocolVersion: 1, Status: "ok", Query: "rc.subject", Lowering: "full",
		Scopes: []analyzer.Scope{
			{ID: "scope:root", Kind: "root"},
			{ID: "cte:line_totals", ParentID: "scope:root", Kind: "cte", Name: "line_totals"},
			{ID: "cte:join_sum", ParentID: "scope:root", Kind: "cte", Name: "join_sum"},
			{ID: "cte:unmatched", ParentID: "scope:root", Kind: "cte", Name: "unmatched"},
		},
		Operations: []analyzer.Operation{
			{ID: "l1", Kind: "source", ScopeID: "cte:line_totals", Source: "line", Alias: "l"},
			{ID: "agg_line", Kind: "aggregate", ScopeID: "cte:line_totals", Inputs: []string{"l1"}, Expressions: []string{"SUM(l.amount)"}},

			{ID: "l2", Kind: "source", ScopeID: "cte:join_sum", Source: "line", Alias: "l"},
			{ID: "d2", Kind: "source", ScopeID: "cte:join_sum", Source: "delivery", Alias: "d"},
			{ID: "join_rows", Kind: "join", ScopeID: "cte:join_sum", Inputs: []string{"l2", "d2"}, JoinKind: "inner", Expressions: []string{"l.order_id=d.order_id"}},
			{ID: "agg_join", Kind: "aggregate", ScopeID: "cte:join_sum", Inputs: []string{"join_rows"}, Expressions: []string{"SUM(l.amount)"}},

			{ID: "h3", Kind: "source", ScopeID: "cte:unmatched", Source: "header", Alias: "h"},
			{ID: "d3", Kind: "source", ScopeID: "cte:unmatched", Source: "delivery", Alias: "d"},
			{ID: "outer_join", Kind: "join", ScopeID: "cte:unmatched", Inputs: []string{"h3", "d3"}, JoinKind: "left", Expressions: []string{"h.id=d.order_id"}},
			{ID: "anti", Kind: "filter", ScopeID: "cte:unmatched", Inputs: []string{"outer_join"}, Expressions: []string{"d.id IS NULL"}},

			{ID: "ref_totals", Kind: "source", ScopeID: "scope:root", Source: "line_totals", SourceKind: "cte", Binding: "cte:line_totals"},
			{ID: "proj", Kind: "project", ScopeID: "scope:root", Inputs: []string{"ref_totals"}, Expressions: []string{"total"}},
		},
	}
}

func rcBindings() map[string]string {
	return map[string]string{"l1": "D1", "l2": "D1", "d2": "D2", "d3": "D2", "h3": "H"}
}

func rcIR(mutate func(*analyzer.Result)) DerivedIR {
	result := rcResult()
	if mutate != nil {
		mutate(&result)
	}
	return DeriveFromAnalysis(EmbeddedSQL{ID: "rc.subject"}, "db", result, nil)
}

// rcExpected is written by hand from the fixture description above, not
// produced by running AssessChangeImpact.
type rcExpected struct {
	verdict      ImpactVerdict
	refresh      string
	dependsOn    []string
	unresolved   []string
	nonMonotonic []string
}

func checkImpact(t *testing.T, name string, got ChangeImpact, want rcExpected) {
	t.Helper()
	if got.Verdict != want.verdict || got.Refresh != want.refresh ||
		!reflect.DeepEqual(got.DependsOn, want.dependsOn) ||
		!reflect.DeepEqual(got.Unresolved, want.unresolved) ||
		!reflect.DeepEqual(got.NonMonotonic, want.nonMonotonic) {
		t.Errorf("%s: got %+v, want %+v", name, got, want)
	}
	if got.DisplayReusable != (want.verdict == ImpactRetain) {
		t.Errorf("%s: display reuse must follow retain only: %+v", name, got)
	}
	// A read-dependency verdict never grants operation eligibility.
	if got.OperationEligibility != OperationEligibilityNotEvaluated {
		t.Errorf("%s: operation eligibility leaked: %q", name, got.OperationEligibility)
	}
}

func TestRC01IndependentContextChange(t *testing.T) {
	ir := rcIR(nil)
	cases := []struct {
		name    string
		claim   string
		changed []string
		want    rcExpected
	}{
		// Over-invalidation negative: D2 changed, D1-only total is kept.
		{"D2 keeps D1-only sum", "agg_line", []string{"D2"},
			rcExpected{ImpactRetain, RefreshNone, []string{"D1"}, nil, nil}},
		// False-reuse negatives: anything reading D2 is not current.
		{"D2 rechecks join aggregate", "agg_join", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1", "D2"}, nil, []string{"aggregate:D2"}}},
		// Even a plain inner join gets a full re-read, never a delta.
		{"D2 rechecks inner join rows", "join_rows", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1", "D2"}, nil, nil}},
		{"D2 rechecks left join", "outer_join", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil, []string{"outer-join-null-side:D2"}}},
		// Symmetric controls.
		{"D1 rechecks D1-only sum", "agg_line", []string{"D1"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, []string{"aggregate:D1"}}},
		{"D1 keeps header/delivery claim", "anti", []string{"D1"},
			rcExpected{ImpactRetain, RefreshNone, []string{"D2", "H"}, nil, nil}},
		{"no change keeps claim", "agg_join", nil,
			rcExpected{ImpactRetain, RefreshNone, []string{"D1", "D2"}, nil, nil}},
		// CTE reference: scope granularity, hence monotonicity unproven.
		{"D1 rechecks root via CTE binding", "proj", []string{"D1"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1"}, nil, []string{"aggregate:D1", "scope-binding:D1"}}},
		// A CTE acts only through a source occurrence bound to it. The
		// sibling CTEs reading D2 are not bound from the root's closure.
		{"D2 keeps root: unreferenced sibling CTEs are not dependencies", "proj", []string{"D2"},
			rcExpected{ImpactRetain, RefreshNone, []string{"D1"}, nil, nil}},
	}
	for _, c := range cases {
		got := AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: c.claim, Bindings: rcBindings(), Changed: c.changed})
		checkImpact(t, c.name, got, c.want)
	}
}

func TestRC02UnknownDependencyIsNotNoImpact(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*analyzer.Result)
		bindings func(map[string]string)
		claim    string
		changed  []string
		want     rcExpected
	}{
		{"opaque operator inside the closure", func(r *analyzer.Result) {
			r.Operations[5].Inputs = []string{"opq"}
			r.Operations = append(r.Operations, analyzer.Operation{ID: "opq", Kind: "opaque", ScopeID: "cte:join_sum", Inputs: []string{"join_rows"}})
		}, nil, "agg_join", []string{"H"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"opaque:opq"}, nil}},
		// Grounded local retention stays possible next to the opaque part.
		{"opaque in a sibling scope does not block D1-only sum", func(r *analyzer.Result) {
			r.Operations[5].Inputs = []string{"opq"}
			r.Operations = append(r.Operations, analyzer.Operation{ID: "opq", Kind: "opaque", ScopeID: "cte:join_sum", Inputs: []string{"join_rows"}})
		}, nil, "agg_line", []string{"D2"},
			rcExpected{ImpactRetain, RefreshNone, []string{"D1"}, nil, nil}},
		{"partial lowering is not localized", func(r *analyzer.Result) { r.Lowering = "partial" },
			nil, "agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"lowering-not-full"}, nil}},
		// Same source name as a bound occurrence: never inferred.
		{"source without producer binding", nil, func(b map[string]string) { delete(b, "l1") },
			"agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"source-unbound:l1"}, nil}},
		{"dangling operator input", func(r *analyzer.Result) { r.Operations[1].Inputs = []string{"missing"} },
			nil, "agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"dangling-input:missing"}, nil}},
		{"unlinked nested subquery reads the changed context", func(r *analyzer.Result) {
			r.Scopes = append(r.Scopes, analyzer.Scope{ID: "sub:exists", ParentID: "cte:line_totals", Kind: "subquery"})
			r.Operations = append(r.Operations, analyzer.Operation{ID: "s_d", Kind: "source", ScopeID: "sub:exists", Source: "delivery", Alias: "x"})
		}, func(b map[string]string) { b["s_d"] = "D2" }, "agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"unlinked-nested-dependency:D2"}, nil}},
		{"unlinked nested subquery, other context changed", func(r *analyzer.Result) {
			r.Scopes = append(r.Scopes, analyzer.Scope{ID: "sub:exists", ParentID: "cte:line_totals", Kind: "subquery"})
			r.Operations = append(r.Operations, analyzer.Operation{ID: "s_d", Kind: "source", ScopeID: "sub:exists", Source: "delivery", Alias: "x"})
		}, func(b map[string]string) { b["s_d"] = "D2" }, "agg_line", []string{"H"},
			rcExpected{ImpactRetain, RefreshNone, []string{"D1"}, nil, nil}},
		// The predicate subquery reads D2 only through a CTE reference.
		{"unlinked nested subquery reads the changed context through a CTE", func(r *analyzer.Result) {
			r.Scopes = append(r.Scopes, analyzer.Scope{ID: "sub:exists", ParentID: "cte:line_totals", Kind: "subquery"})
			r.Operations = append(r.Operations, analyzer.Operation{ID: "ref_js", Kind: "source", ScopeID: "sub:exists", Source: "join_sum", SourceKind: "cte", Binding: "cte:join_sum"})
		}, nil, "agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"unlinked-nested-dependency:D2"}, nil}},
		// A stale binding (made for another analysis) is not silently ignored.
		{"binding names no source occurrence of this IR", nil, func(b map[string]string) { b["ghost"] = "D2" },
			"agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"binding-invalid:ghost"}, nil}},
		{"binding placed on a CTE reference instead of a table source", nil, func(b map[string]string) { b["ref_totals"] = "D1" },
			"agg_line", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"binding-invalid:ref_totals"}, nil}},
		{"claim not in the IR", nil, nil, "nope", []string{"D2"},
			rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, nil, []string{"claim-not-found"}, nil}},
		{"unverified join kind is not treated as monotone", func(r *analyzer.Result) { r.Operations[4].JoinKind = "anti" },
			nil, "join_rows", []string{"D2"},
			rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D1", "D2"}, nil, []string{"join-kind-unverified:D2"}}},
	}
	for _, c := range cases {
		bindings := rcBindings()
		if c.bindings != nil {
			c.bindings(bindings)
		}
		got := AssessChangeImpact(rcIR(c.mutate), ChangeImpactRequest{Query: "rc.subject", Claim: c.claim, Bindings: bindings, Changed: c.changed})
		checkImpact(t, c.name, got, c.want)
	}
}

func TestRC02PossibleDependenciesAndQueryIdentity(t *testing.T) {
	ir := rcIR(func(r *analyzer.Result) {
		r.Scopes = append(r.Scopes, analyzer.Scope{ID: "sub:exists", ParentID: "cte:line_totals", Kind: "subquery"})
		r.Operations = append(r.Operations, analyzer.Operation{ID: "s_d", Kind: "source", ScopeID: "sub:exists", Source: "delivery", Alias: "x"})
	})
	bindings := rcBindings()
	bindings["s_d"] = "D2"
	got := AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: "agg_line", Bindings: bindings})
	if got.Verdict != ImpactRetain || !reflect.DeepEqual(got.PossiblyDependsOn, []string{"D2"}) || !reflect.DeepEqual(got.DependsOn, []string{"D1"}) {
		t.Errorf("possible dependency must be reported apart from the declared closure: %+v", got)
	}
	// Bindings declared for one query are not accepted for another.
	got = AssessChangeImpact(rcIR(nil), ChangeImpactRequest{Query: "rc.other", Claim: "agg_line", Bindings: rcBindings(), Changed: []string{"D2"}})
	checkImpact(t, "query mismatch", got, rcExpected{ImpactUnknown, RefreshReadOnlyRefetch, []string{"D1"}, []string{"query-mismatch"}, nil})
}

func TestRC04LeftJoinUnmatchedRowIsProvisional(t *testing.T) {
	ir := rcIR(nil)
	complete := InputSlice{State: SliceComplete, ObservedRows: 1}
	states := []struct {
		name        string
		d2          *InputSlice
		provisional []string
		echoed      InputSlice
	}{
		{"not fetched", &InputSlice{State: SliceNotFetched}, []string{"unmatched-rows-not-absence-proof:D2:not-fetched"}, InputSlice{State: SliceNotFetched}},
		{"partial page", &InputSlice{State: SlicePartial, ObservedRows: 1}, []string{"unmatched-rows-not-absence-proof:D2:partial"}, InputSlice{State: SlicePartial, ObservedRows: 1}},
		{"failed read", &InputSlice{State: SliceFailed}, []string{"unmatched-rows-not-absence-proof:D2:failed"}, InputSlice{State: SliceFailed}},
		{"state not supplied", nil, []string{"unmatched-rows-not-absence-proof:D2:unknown"}, InputSlice{State: SliceStateUnknown}},
		// A verified empty result is an absence within the retrieved scope.
		{"complete with zero rows", &InputSlice{State: SliceComplete}, nil, InputSlice{State: SliceComplete}},
		{"complete with rows", &InputSlice{State: SliceComplete, ObservedRows: 3}, nil, InputSlice{State: SliceComplete, ObservedRows: 3}},
	}
	for _, s := range states {
		slices := map[string]InputSlice{"H": complete}
		if s.d2 != nil {
			slices["D2"] = *s.d2
		}
		got := AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: "anti", Bindings: rcBindings(), Slices: slices})
		if !reflect.DeepEqual(got.Provisional, s.provisional) || got.Inputs["D2"] != s.echoed {
			t.Errorf("%s: got provisional=%v inputs=%v", s.name, got.Provisional, got.Inputs)
		}
		if got.Verdict != ImpactRetain {
			t.Errorf("%s: no change was reported, got %s", s.name, got.Verdict)
		}
	}

	// The preserved (left) side being partial is not a null-side hazard.
	got := AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: "anti", Bindings: rcBindings(),
		Slices: map[string]InputSlice{"H": {State: SlicePartial, ObservedRows: 1}, "D2": complete}})
	if got.Provisional != nil {
		t.Errorf("left side partial must not be reported as null-side: %v", got.Provisional)
	}

	// A matching delivery row arrives later: the shown (H, NULL) row may be
	// replaced, so the claim is re-read rather than appended to.
	got = AssessChangeImpact(ir, ChangeImpactRequest{Query: "rc.subject", Claim: "anti", Bindings: rcBindings(), Changed: []string{"D2"},
		Slices: map[string]InputSlice{"H": complete, "D2": {State: SlicePartial, ObservedRows: 1}}})
	checkImpact(t, "late matching row", got, rcExpected{ImpactRecheck, RefreshReadOnlyRefetch, []string{"D2", "H"}, nil,
		[]string{"filter-unparsed:D2", "outer-join-null-side:D2"}})
	if !reflect.DeepEqual(got.Provisional, []string{"unmatched-rows-not-absence-proof:D2:partial"}) {
		t.Errorf("late matching row: provisional=%v", got.Provisional)
	}
}

func TestChangeImpactIsHeadlessJSON(t *testing.T) {
	got := AssessChangeImpact(rcIR(nil), ChangeImpactRequest{Query: "rc.subject", Claim: "agg_line", Bindings: rcBindings(), Changed: []string{"D2"}})
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"claim":"agg_line","verdict":"retain","refresh":"none","dependsOn":["D1"],"inputs":{"D1":{"state":"unknown","observedRows":0}},"displayReusable":true,"operationEligibility":"not-evaluated"}`
	if string(b) != want {
		t.Fatalf("got %s", b)
	}
}

package derivationrelationsets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/semantic/derivationref"
)

// TestHandFixedExpectedMatchesSharedFixture ties this reference to the
// hand-fixed values that plan/fixtures/metric-aggregation/verify_sqlite.py
// also asserts against. The file is not produced by either side.
func TestHandFixedExpectedMatchesSharedFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "plan", "fixtures", "metric-aggregation", "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases map[string]struct {
			Expected string   `json:"expected"`
			Unsafe   []string `json:"unsafe"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	day2 := []Row{{Identity: "A", Amount: amount("110")}, {Identity: "B", Amount: amount("190")}}
	eligible := map[string]Claim{
		"MC-01": mc01LineTotal(Row{Identity: "H/L1", Amount: amount("100")}, Row{Identity: "H/L2", Amount: amount("100")}),
		"MC-02": mc02Mean(),
		"MC-03": mc03Ratio(),
		"MC-05": mc05Balance(Axis{Name: "account", Kind: AxisDimension}, "day2", day2...),
	}
	for id, claim := range eligible {
		want, ok := fixture.Cases[id]
		if !ok {
			t.Fatalf("%s is missing from expected.json", id)
		}
		got := Reaggregate(claim)
		if got.Verdict != Eligible || got.Exact != want.Expected {
			t.Errorf("%s: want %s, got %+v", id, want.Expected, got)
		}
		for _, unsafe := range want.Unsafe {
			if got.Exact == unsafe {
				t.Errorf("%s: returned the unsafe value %s", id, unsafe)
			}
		}
	}
	// MC-04's correct value needs person identities; only the refusal is owned.
	if _, ok := fixture.Cases["MC-04"]; !ok {
		t.Fatal("MC-04 is missing from expected.json")
	}
}

func mc06Sum(policy AggregatePolicy, completeness Completeness, rows ...Row) Claim {
	set := groups("order.O", "order.line_amount", rows...)
	set.Completeness = completeness
	return Claim{
		MeasureRef: "synthetic.amount",
		Law:        LawSum,
		BaseGrain:  "sales.order_line",
		Scope:      "order O",
		Axis:       Axis{Name: "line", Kind: AxisDimension},
		Partition:  PartitionDisjoint,
		Time:       Temporal{Kind: PeriodFlow},
		Policy:     policy,
		Components: map[Component]Part{ComponentSum: part("JPY", set)},
	}
}

// MC-06: no rows, a NULL-only row, a page and a never-read set are four
// different states. None of them is normalized to 0 without a declaration.
func TestMC06EmptyNullPartialAndUnfetchedStayDistinct(t *testing.T) {
	null := Row{Identity: "L1", Amount: nullAmount()}

	wantNoValue(t, Reaggregate(mc06Sum(AggregatePolicy{}, Complete)), Unknown, CodePolicyUndeclared, GapMissingSemantics)
	wantNoValue(t, Reaggregate(mc06Sum(AggregatePolicy{}, Complete, null)), Unknown, CodePolicyUndeclared, GapMissingSemantics)

	wantEligible(t, Reaggregate(mc06Sum(AggregatePolicy{EmptySet: EmptySetZero}, Complete)), "0", "0")
	emptyAsNull := Reaggregate(mc06Sum(AggregatePolicy{EmptySet: EmptySetNull}, Complete))
	nullPropagated := Reaggregate(mc06Sum(AggregatePolicy{Nulls: NullsPropagate}, Complete, null))
	for _, got := range []Report{emptyAsNull, nullPropagated} {
		if got.Verdict != Eligible || got.Result.State != derivationref.Null || got.Exact != "" {
			t.Fatalf("declared NULL outcome expected, got %+v", got)
		}
	}
	// Skipping the only NULL leaves an empty set, which needs its own policy.
	wantNoValue(t, Reaggregate(mc06Sum(AggregatePolicy{Nulls: NullsSkip}, Complete, null)), Unknown, CodePolicyUndeclared, GapMissingSemantics)

	present := Row{Identity: "L1", Amount: amount("600")}
	wantNoValue(t, Reaggregate(mc06Sum(AggregatePolicy{EmptySet: EmptySetZero}, Partial, present)), Ineligible, CodeIncompleteSet, GapMissingInput)
	wantNoValue(t, Reaggregate(mc06Sum(AggregatePolicy{EmptySet: EmptySetZero}, Unfetched)), Unknown, CodeInputUnavailable, GapMissingInput)
}

// Rounding stage: 1000 facts of 0.004 at scale 2, half-up. round(sum)=4.00 and
// sum(round)=0.00 are both possible business rules. This reference composes
// exactly what it is given and never rounds, so the stage stays visible in the
// input; which stage is correct is a D8 #1705 / owner decision, not made here.
func TestRoundingStageIsNeverChosenHere(t *testing.T) {
	exact := make([]Row, 0, 1000)
	rounded := make([]Row, 0, 1000)
	for i := 0; i < 1000; i++ {
		id := "F" + string(rune('0'+i/100)) + string(rune('0'+i/10%10)) + string(rune('0'+i%10))
		exact = append(exact, Row{Identity: id, Amount: amount("0.004")})
		rounded = append(rounded, Row{Identity: id, Amount: amount("0.00")})
	}
	wantEligible(t, Reaggregate(mc06Sum(AggregatePolicy{}, Complete, exact...)), "4", "4")
	wantEligible(t, Reaggregate(mc06Sum(AggregatePolicy{}, Complete, rounded...)), "0", "0")
}

func TestMeanCountMustBeAWholeNumberOfFacts(t *testing.T) {
	for _, bad := range []string{"2.5", "-2"} {
		c := mc02Mean()
		p := c.Components[ComponentCount]
		p.Set.Rows[0].Amount = amount(bad)
		c.Components[ComponentCount] = p
		wantNoValue(t, Reaggregate(c), Unknown, CodeBindingInvalid, GapMissingSemantics)
	}
}

package derivationrelationsets

import (
	"errors"
	"testing"

	"tei/internal/semantic/derivationref"
)

// The expected values below are fixed from the base facts by hand and are
// independent of this package and of any SQL result (#2593 MC-01..MC-05):
//
//	MC-01  lines L1=100, L2=100, 3 deliveries -> line total 200
//	       (naive JOIN sum 600 and SUM(DISTINCT value) 100 are both wrong)
//	MC-02  A: 2 facts, sum 20 (mean 10); B: 8 facts, sum 160 (mean 20)
//	       -> overall mean 180/10 = 18 (mean of means 15 is wrong)
//	MC-03  A: 1/2, B: 8/10 -> overall 9/12 = 3/4 = 0.75 (mean of ratios 0.65 is wrong)
//	MC-04  A={U1,U2}, B={U2,U3} -> 3 unique (2+2=4 is wrong)
//	MC-05  day1: A100+B200, day2: A110+B190 -> balance at day2 = 300 (600 is wrong)

func groups(anchor, relation string, rows ...Row) SetRef {
	return SetRef{
		Anchor:       anchor,
		RelationID:   relation,
		Key:          []string{"group.id"},
		Shape:        Aggregate,
		Completeness: Complete,
		Rows:         rows,
	}
}

func part(unit string, set SetRef) Part { return Part{Set: set, Unit: unit} }

func mc02Mean() Claim {
	return Claim{
		MeasureRef: "synthetic.score",
		Law:        LawMean,
		BaseGrain:  "synthetic.observation",
		Scope:      "all departments",
		Axis:       Axis{Name: "department", Kind: AxisDimension},
		Partition:  PartitionDisjoint,
		Time:       Temporal{Kind: PeriodFlow},
		Policy:     AggregatePolicy{Nulls: NullsPropagate},
		Components: map[Component]Part{
			ComponentSum:   part("point", groups("org", "dept.score_sum", Row{Identity: "A", Amount: amount("20")}, Row{Identity: "B", Amount: amount("160")})),
			ComponentCount: part("observation", groups("org", "dept.score_count", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "B", Amount: amount("8")})),
		},
	}
}

func mc03Ratio() Claim {
	return Claim{
		MeasureRef: "synthetic.pass_rate",
		Law:        LawRatio,
		BaseGrain:  "synthetic.attempt",
		Scope:      "all departments",
		Axis:       Axis{Name: "department", Kind: AxisDimension},
		Partition:  PartitionDisjoint,
		Time:       Temporal{Kind: PeriodFlow},
		Components: map[Component]Part{
			ComponentNumerator:   part("attempt", groups("org", "dept.passed", Row{Identity: "A", Amount: amount("1")}, Row{Identity: "B", Amount: amount("8")})),
			ComponentDenominator: part("attempt", groups("org", "dept.attempted", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "B", Amount: amount("10")})),
		},
	}
}

func mc01LineTotal(rows ...Row) Claim {
	set := collection("sales.order_lines", rows...)
	set.Anchor = "order.H"
	return Claim{
		MeasureRef: "synthetic.line_amount",
		Law:        LawSum,
		BaseGrain:  "sales.order_line",
		Scope:      "order H",
		Axis:       Axis{Name: "line", Kind: AxisDimension},
		Partition:  PartitionDisjoint,
		Time:       Temporal{Kind: PeriodFlow},
		Components: map[Component]Part{ComponentSum: part("JPY", set)},
	}
}

func mc05Balance(axis Axis, asOf string, rows ...Row) Claim {
	return Claim{
		MeasureRef: "synthetic.balance",
		Law:        LawSum,
		BaseGrain:  "account x date",
		Scope:      "accounts A and B",
		Axis:       axis,
		Partition:  PartitionDisjoint,
		Time:       Temporal{Kind: PointStock, AsOf: asOf},
		Components: map[Component]Part{ComponentSum: part("JPY", groups("ledger", "account.balance", rows...))},
	}
}

func wantEligible(t *testing.T, got Report, exact, decimal string) {
	t.Helper()
	if got.Verdict != Eligible || got.Code != CodeOK || got.Exact != exact ||
		got.Result.State != derivationref.Present || got.Result.Decimal != decimal || len(got.Findings) != 0 {
		t.Fatalf("want eligible %s (%s), got %+v", exact, decimal, got)
	}
}

// wantNoValue asserts the verdict and code, and that no value leaked.
func wantNoValue(t *testing.T, got Report, verdict Verdict, code, gap string) {
	t.Helper()
	if got.Verdict != verdict || got.Code != code || got.Gap != gap {
		t.Fatalf("want %s %s (%s), got %+v", verdict, code, gap, got)
	}
	if got.Exact != "" || got.Result != (derivationref.Input{}) {
		t.Fatalf("a %s report must not carry a value: %+v", verdict, got)
	}
}

func TestMC02MeanIsRecomposedFromSumAndCount(t *testing.T) {
	wantEligible(t, Reaggregate(mc02Mean()), "18", "18")
}

func TestMC02MeanOfMeansIsNotAdoptedWhenStateIsLost(t *testing.T) {
	local := part("point", groups("org", "dept.score_mean", Row{Identity: "A", Amount: amount("10")}, Row{Identity: "B", Amount: amount("20")}))

	onlyMeans := mc02Mean()
	onlyMeans.Components = map[Component]Part{ComponentValue: local}
	got := Reaggregate(onlyMeans)
	wantNoValue(t, got, Ineligible, CodeIntermediateStateLost, GapMissingInput)
	if len(got.Requires) != 2 || got.Requires[0] != ComponentSum || got.Requires[1] != ComponentCount {
		t.Fatalf("report must name the required state: %+v", got.Requires)
	}

	// The same rows passed off as the "sum" component of a sum law still
	// cannot become 30: a count is missing for a mean, and without a declared
	// law nothing is composed at all.
	sumWithoutCount := mc02Mean()
	delete(sumWithoutCount.Components, ComponentCount)
	wantNoValue(t, Reaggregate(sumWithoutCount), Ineligible, CodeIntermediateStateLost, GapMissingInput)

	noLaw := mc02Mean()
	noLaw.Law = ""
	wantNoValue(t, Reaggregate(noLaw), Unknown, CodeLawUndeclared, GapMissingSemantics)

	// The primitive no longer sums already reduced rows by itself.
	if _, err := Sum(local.Set, AggregatePolicy{}); !errors.Is(err, ErrShape) {
		t.Fatalf("Sum must refuse an already reduced set, got %v", err)
	}
}

func TestMC03RatioIsRecomposedFromNumeratorAndDenominator(t *testing.T) {
	wantEligible(t, Reaggregate(mc03Ratio()), "3/4", "0.75")
}

func TestMC03MeanOfRatiosIsNotAdoptedWhenStateIsLost(t *testing.T) {
	local := mc03Ratio()
	local.Components = map[Component]Part{
		ComponentValue: part("1", groups("org", "dept.pass_rate", Row{Identity: "A", Amount: amount("0.5")}, Row{Identity: "B", Amount: amount("0.8")})),
	}
	wantNoValue(t, Reaggregate(local), Ineligible, CodeIntermediateStateLost, GapMissingInput)

	// Numerator and denominator that do not cover the same groups have lost
	// their pairing.
	unpaired := mc03Ratio()
	unpaired.Components[ComponentDenominator] = part("attempt", groups("org", "dept.attempted", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "C", Amount: amount("10")}))
	wantNoValue(t, Reaggregate(unpaired), Ineligible, CodeComponentGroupMismatch, GapConflict)

	otherPopulation := mc03Ratio()
	otherPopulation.Components[ComponentDenominator] = part("attempt", groups("other-org", "dept.attempted", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "B", Amount: amount("10")}))
	wantNoValue(t, Reaggregate(otherPopulation), Ineligible, CodeScopeMismatch, GapConflict)
}

func TestRatioThatDoesNotTerminateKeepsTheExactRational(t *testing.T) {
	c := mc03Ratio()
	c.Components[ComponentDenominator] = part("attempt", groups("org", "dept.attempted", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "B", Amount: amount("25")}))
	got := Reaggregate(c)
	if got.Verdict != Eligible || got.Exact != "1/3" || got.Result != (derivationref.Input{}) {
		t.Fatalf("9/27 must stay 1/3 without an invented rounding: %+v", got)
	}
}

func TestMC01JoinOccurrenceIsNotASecondFact(t *testing.T) {
	// Two different lines with the same amount are two facts: 200, not 100.
	wantEligible(t, Reaggregate(mc01LineTotal(
		Row{Identity: "H/L1", Amount: amount("100")},
		Row{Identity: "H/L2", Amount: amount("100")},
	)), "200", "200")

	// The same two lines after a JOIN with three deliveries: six occurrences
	// of two facts. Neither 600 nor a value-deduplicated 100 is returned.
	var joined []Row
	for i := 0; i < 3; i++ {
		joined = append(joined, Row{Identity: "H/L1", Amount: amount("100")}, Row{Identity: "H/L2", Amount: amount("100")})
	}
	wantNoValue(t, Reaggregate(mc01LineTotal(joined...)), Ineligible, CodeFactOccurrenceRepeated, GapConflict)

	// The existing D9 guard still reports the fan-out itself.
	lines := collection("sales.order_lines", Row{Identity: "H/L1", Amount: amount("100")}, Row{Identity: "H/L2", Amount: amount("100")})
	diagnostic, err := JoinFanOut(lines, o1Deliveries(3))
	if !errors.Is(err, ErrFanOut) || diagnostic.NaiveRows != 6 || diagnostic.DistinctRows != 2 {
		t.Fatalf("fan-out regression: %+v, %v", diagnostic, err)
	}
}

func TestMC01PreAggregatedTotalRejoinedIsNotSummedAgain(t *testing.T) {
	// line_total(H)=200 computed first, then joined with three deliveries:
	// the group H occurs three times. Re-summing must not give 600.
	c := mc01LineTotal()
	c.Axis = Axis{Name: "order", Kind: AxisDimension}
	c.Components[ComponentSum] = part("JPY", groups("order.H", "order.line_total",
		Row{Identity: "H", Amount: amount("200")}, Row{Identity: "H", Amount: amount("200")}, Row{Identity: "H", Amount: amount("200")}))
	wantNoValue(t, Reaggregate(c), Ineligible, CodeFactOccurrenceRepeated, GapConflict)
}

func TestMC04GroupCountsAreNotAddedWhenGroupsOverlap(t *testing.T) {
	headcount := Claim{
		MeasureRef: "synthetic.headcount",
		Law:        LawSum,
		BaseGrain:  "person",
		Scope:      "all departments",
		Axis:       Axis{Name: "department", Kind: AxisDimension},
		Partition:  PartitionOverlapping, // U2 belongs to both A and B
		Time:       Temporal{Kind: PointStock, AsOf: "2026-11-01"},
		Components: map[Component]Part{
			ComponentSum: part("person", groups("org", "dept.unique_persons", Row{Identity: "A", Amount: amount("2")}, Row{Identity: "B", Amount: amount("2")})),
		},
	}
	// 2+2=4 is not the unique headcount 3. Composing 3 needs the person
	// identities, which this reference does not carry.
	wantNoValue(t, Reaggregate(headcount), Ineligible, CodeGroupsOverlap, GapConflict)

	headcount.Partition = PartitionUndeclared
	wantNoValue(t, Reaggregate(headcount), Unknown, CodePartitionUndeclared, GapMissingSemantics)
}

func TestMC05StockIsCombinedAtOnePointNotAcrossTime(t *testing.T) {
	accounts := Axis{Name: "account", Kind: AxisDimension}
	day2 := []Row{{Identity: "A", Amount: amount("110")}, {Identity: "B", Amount: amount("190")}}
	wantEligible(t, Reaggregate(mc05Balance(accounts, "day2", day2...)), "300", "300")

	// Without a shared as-of point the same rows are not combined.
	wantNoValue(t, Reaggregate(mc05Balance(accounts, "", day2...)), Unknown, CodeAsOfUndeclared, GapMissingSemantics)

	// Daily balances 300 and 300 are not a current balance of 600.
	days := Axis{Name: "date", Kind: AxisTime}
	wantNoValue(t, Reaggregate(mc05Balance(days, "", Row{Identity: "day1", Amount: amount("300")}, Row{Identity: "day2", Amount: amount("300")})),
		Ineligible, CodeStockSummedAcrossTime, GapConflict)
}

func TestMissingDeclarationsStayUnknown(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Claim)
		code   string
	}{
		{"scope", func(c *Claim) { c.Scope = "" }, CodeScopeUndeclared},
		{"base grain", func(c *Claim) { c.BaseGrain = "" }, CodeBaseGrainUndeclared},
		{"axis", func(c *Claim) { c.Axis = Axis{} }, CodeAxisUndeclared},
		{"axis kind", func(c *Claim) { c.Axis.Kind = "" }, CodeAxisUndeclared},
		{"partition", func(c *Claim) { c.Partition = PartitionUndeclared }, CodePartitionUndeclared},
		{"time", func(c *Claim) { c.Time = Temporal{} }, CodeTimeUndeclared},
		{"count basis", func(c *Claim) { c.Policy = AggregatePolicy{} }, CodeCountBasisUndeclared},
		{"unit", func(c *Claim) {
			p := c.Components[ComponentSum]
			p.Unit = ""
			c.Components[ComponentSum] = p
		}, CodeUnitUndeclared},
		{"group identity", func(c *Claim) {
			p := c.Components[ComponentCount]
			p.Set.Rows[0].Identity = ""
			c.Components[ComponentCount] = p
		}, CodeBindingInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mc02Mean()
			tt.mutate(&c)
			wantNoValue(t, Reaggregate(c), Unknown, tt.code, GapMissingSemantics)
		})
	}
}

func TestCompletenessAndNullAreNotGuessed(t *testing.T) {
	withCount := func(mutate func(*SetRef)) Claim {
		c := mc02Mean()
		p := c.Components[ComponentCount]
		mutate(&p.Set)
		c.Components[ComponentCount] = p
		return c
	}
	// A page of the groups is known not to be the whole population.
	wantNoValue(t, Reaggregate(withCount(func(s *SetRef) { s.Completeness = Partial })), Ineligible, CodeIncompleteSet, GapMissingInput)
	// A never-read component is unknown, not zero.
	wantNoValue(t, Reaggregate(withCount(func(s *SetRef) { s.Completeness = Unfetched })), Unknown, CodeInputUnavailable, GapMissingInput)
	wantNoValue(t, Reaggregate(withCount(func(s *SetRef) { s.Rows[1].Amount = derivationref.Input{State: derivationref.Missing} })), Unknown, CodeInputUnavailable, GapMissingInput)

	// A NULL group follows the declared policy; it never becomes 0.
	propagated := Reaggregate(withCount(func(s *SetRef) { s.Rows[1].Amount = nullAmount() }))
	if propagated.Verdict != Eligible || propagated.Result.State != derivationref.Null || propagated.Exact != "" {
		t.Fatalf("declared propagate must yield NULL, got %+v", propagated)
	}
	skipped := withCount(func(s *SetRef) { s.Rows[1].Amount = nullAmount() })
	skipped.Policy.Nulls = NullsSkip
	wantNoValue(t, Reaggregate(skipped), Unknown, CodeNullPairingUndefined, GapMissingSemantics)

	// No groups, or a zero total count, is not a mean of 0.
	empty := mc02Mean()
	empty.Components[ComponentSum] = part("point", groups("org", "dept.score_sum"))
	empty.Components[ComponentCount] = part("observation", groups("org", "dept.score_count"))
	wantNoValue(t, Reaggregate(empty), Unknown, CodePolicyUndeclared, GapMissingSemantics)
	zero := withCount(func(s *SetRef) { s.Rows[0].Amount, s.Rows[1].Amount = amount("0"), amount("0") })
	wantNoValue(t, Reaggregate(zero), Unknown, CodeZeroDenominator, GapMissingSemantics)
}

func TestLawThatIsNotOwnedIsNotApproximated(t *testing.T) {
	c := mc02Mean()
	c.Law = "median"
	wantNoValue(t, Reaggregate(c), Unknown, CodeLawNotOwned, GapOutOfScope)
}

func TestDefiniteViolationOutranksMissingDeclaration(t *testing.T) {
	c := mc02Mean()
	c.Scope = ""
	c.Partition = PartitionOverlapping
	got := Reaggregate(c)
	wantNoValue(t, got, Ineligible, CodeGroupsOverlap, GapConflict)
	if len(got.Findings) != 2 {
		t.Fatalf("every finding must be reported, got %+v", got.Findings)
	}
}

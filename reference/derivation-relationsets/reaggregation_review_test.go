package derivationrelationsets

import (
	"testing"

	"tei/internal/semantic/derivationref"
)

// Tests added after the adversarial review of #2593: each pins a rule that a
// deliberately broken implementation would otherwise have passed.

func TestNonPlainDecimalIsNeverRoundedIntoAValue(t *testing.T) {
	for _, bad := range []string{"1e-5", "1/3", "0x10", "+1", ".5", "1."} {
		c := mc03Ratio()
		p := c.Components[ComponentNumerator]
		p.Set.Rows[0].Amount = amount(bad)
		c.Components[ComponentNumerator] = p
		wantNoValue(t, Reaggregate(c), Unknown, CodeBindingInvalid, GapMissingSemantics)
	}
	// The same refusal reaches the plain D9 sum.
	if _, err := Sum(collection("x", Row{Identity: "a", Amount: amount("1e-5")}), AggregatePolicy{}); err == nil {
		t.Fatal("Sum must refuse an exponent literal")
	}
}

func TestMeanGroupWithSumOverZeroFactsIsContradictory(t *testing.T) {
	c := mc02Mean()
	p := c.Components[ComponentCount]
	p.Set.Rows[0].Amount = amount("0") // A: sum 20 over 0 facts
	c.Components[ComponentCount] = p
	wantNoValue(t, Reaggregate(c), Ineligible, CodeGroupStateContradictory, GapConflict)

	// A group with zero facts and zero sum is simply empty and is allowed.
	ok := mc02Mean()
	s, n := ok.Components[ComponentSum], ok.Components[ComponentCount]
	s.Set.Rows[0].Amount, n.Set.Rows[0].Amount = amount("0"), amount("0")
	ok.Components[ComponentSum], ok.Components[ComponentCount] = s, n
	wantEligible(t, Reaggregate(ok), "20", "20")
}

func TestOneRelationCannotStandForTwoComponents(t *testing.T) {
	c := mc02Mean()
	same := c.Components[ComponentSum]
	c.Components[ComponentCount] = Part{Set: same.Set, Unit: "observation"}
	wantNoValue(t, Reaggregate(c), Ineligible, CodeComponentNotDistinct, GapConflict)

	otherKey := mc02Mean()
	p := otherKey.Components[ComponentCount]
	p.Set.Key = []string{"other.id"}
	otherKey.Components[ComponentCount] = p
	wantNoValue(t, Reaggregate(otherKey), Ineligible, CodeComponentGroupMismatch, GapConflict)
}

func TestStockOverTimeWithNonSumLawIsNotOwned(t *testing.T) {
	c := mc05Balance(Axis{Name: "date", Kind: AxisTime}, "", Row{Identity: "d1", Amount: amount("300")}, Row{Identity: "d2", Amount: amount("300")})
	c.Law = LawMean
	c.Policy.Nulls = NullsPropagate
	c.Components[ComponentCount] = part("day", groups("ledger", "balance.days", Row{Identity: "d1", Amount: amount("1")}, Row{Identity: "d2", Amount: amount("1")}))
	wantNoValue(t, Reaggregate(c), Unknown, CodeStockOverTimeNotOwned, GapOutOfScope)
}

func TestNullPropagationDoesNotDependOnComponentOrder(t *testing.T) {
	for _, name := range []Component{ComponentSum, ComponentCount} {
		c := mc02Mean()
		p := c.Components[name]
		p.Set.Rows[1].Amount = nullAmount()
		c.Components[name] = p
		got := Reaggregate(c)
		if got.Verdict != Eligible || got.Result.State != derivationref.Null || got.Exact != "" {
			t.Fatalf("NULL in %s: %+v", name, got)
		}
	}
}

func TestNullSkipInEitherComponentOfAPairIsUndefined(t *testing.T) {
	for _, name := range []Component{ComponentSum, ComponentCount} {
		c := mc02Mean()
		c.Policy.Nulls = NullsSkip
		p := c.Components[name]
		p.Set.Rows[1].Amount = nullAmount()
		c.Components[name] = p
		wantNoValue(t, Reaggregate(c), Unknown, CodeNullPairingUndefined, GapMissingSemantics)
	}
}

func TestUnitIsRequiredOnEveryComponent(t *testing.T) {
	for _, name := range []Component{ComponentSum, ComponentCount} {
		c := mc02Mean()
		p := c.Components[name]
		p.Unit = "  "
		c.Components[name] = p
		wantNoValue(t, Reaggregate(c), Unknown, CodeUnitUndeclared, GapMissingSemantics)
	}
}

func TestGroupPairingIgnoresOrderButNotMembership(t *testing.T) {
	c := mc02Mean()
	p := c.Components[ComponentCount]
	p.Set.Rows[0], p.Set.Rows[1] = p.Set.Rows[1], p.Set.Rows[0] // B, A
	c.Components[ComponentCount] = p
	wantEligible(t, Reaggregate(c), "18", "18")

	c = mc02Mean()
	p = c.Components[ComponentCount]
	p.Set.Rows = append(p.Set.Rows, Row{Identity: "C", Amount: amount("0")})
	c.Components[ComponentCount] = p
	wantNoValue(t, Reaggregate(c), Ineligible, CodeComponentGroupMismatch, GapConflict)
}

func TestWhitespaceOnlyDeclarationsAreUndeclared(t *testing.T) {
	mutate := map[string]func(*Claim){
		"law":        func(c *Claim) { c.Law = "  " },
		"base grain": func(c *Claim) { c.BaseGrain = "\t" },
		"scope":      func(c *Claim) { c.Scope = " " },
		"axis name":  func(c *Claim) { c.Axis.Name = " " },
	}
	for name, f := range mutate {
		c := mc02Mean()
		f(&c)
		if got := Reaggregate(c); got.Verdict != Unknown || got.Exact != "" || got.Result != (derivationref.Input{}) {
			t.Fatalf("%s: %+v", name, got)
		}
	}
	asOf := mc05Balance(Axis{Name: "account", Kind: AxisDimension}, "  ", Row{Identity: "A", Amount: amount("1")})
	wantNoValue(t, Reaggregate(asOf), Unknown, CodeAsOfUndeclared, GapMissingSemantics)
}

func TestZeroDenominatorAndRepeatingDecimals(t *testing.T) {
	c := mc03Ratio()
	p := c.Components[ComponentDenominator]
	p.Set.Rows[0].Amount, p.Set.Rows[1].Amount = amount("0"), amount("0")
	c.Components[ComponentDenominator] = p
	wantNoValue(t, Reaggregate(c), Unknown, CodeZeroDenominator, GapMissingSemantics)

	for _, tt := range []struct{ den, exact, decimal string }{
		{"8", "9/8", "1.125"}, {"20", "9/20", "0.45"}, {"3", "3", "3"},
	} {
		r := mc03Ratio()
		d := r.Components[ComponentDenominator]
		d.Set.Rows[0].Amount, d.Set.Rows[1].Amount = amount(tt.den), amount("0")
		r.Components[ComponentDenominator] = d
		n := r.Components[ComponentNumerator]
		n.Set.Rows[0].Amount, n.Set.Rows[1].Amount = amount("9"), amount("0")
		r.Components[ComponentNumerator] = n
		wantEligible(t, Reaggregate(r), tt.exact, tt.decimal)
	}
	// 9/7 has no finite decimal: only the exact rational is returned.
	r := mc03Ratio()
	d, n := r.Components[ComponentDenominator], r.Components[ComponentNumerator]
	d.Set.Rows[0].Amount, d.Set.Rows[1].Amount = amount("7"), amount("0")
	n.Set.Rows[0].Amount, n.Set.Rows[1].Amount = amount("9"), amount("0")
	r.Components[ComponentDenominator], r.Components[ComponentNumerator] = d, n
	if got := Reaggregate(r); got.Verdict != Eligible || got.Exact != "9/7" || got.Result != (derivationref.Input{}) {
		t.Fatalf("9/7: %+v", got)
	}
}

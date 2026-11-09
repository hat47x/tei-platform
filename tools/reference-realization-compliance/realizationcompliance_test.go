package realizationcompliance

import "testing"

// The fixtures follow Data Derivation D6 (#1694): rule
// `line-amount(quantity decimal, unit_price money[JPY])`, same currency, no
// rounding of the exact product.

func contract() Input {
	return Input{
		RuleID:         "line-amount",
		RuleOutputType: "amount",
		RuleNullPolicy: NullPropagate,
		RuleRounding:   RoundHalfEven,
		RulePure:       true,
	}
}

func oracleCases() []Case {
	return []Case{
		{QuantityState: StatePresent, Quantity: "2", UnitPrice: "300", ResultState: StatePresent, Amount: "600"},
		{QuantityState: StatePresent, Quantity: "0", UnitPrice: "300", ResultState: StatePresent, Amount: "0"},
		{QuantityState: StatePresent, Quantity: "0.2", UnitPrice: "0.5", ResultState: StatePresent, Amount: "0.1"},
		{QuantityState: StateNull, UnitPrice: "300", ResultState: StateNull},
		{QuantityState: StateMissing, UnitPrice: "300", ResultState: StateMissing},
	}
}

func goRealization() Realization {
	return Realization{
		Plugin: "case-line-amount-go", Version: "3", Language: "go",
		OutputType: "amount", NullPolicy: NullPropagate, Rounding: RoundHalfEven,
		Pure: true, Registered: true, Locked: true, LockedVersion: "3",
		Cases: oracleCases(),
	}
}

func TestCompliantRealizationPassesTheOracle(t *testing.T) {
	in := contract()
	in.Realizations = []Realization{goRealization()}
	report := Check(in)
	if !report.Accepted || report.Code != CodeCompliant || report.RuleID != "line-amount" {
		t.Fatalf("report=%+v want COMPLIANT", report)
	}
	if !report.EquivalenceUnproven {
		t.Fatalf("report=%+v want a single realization to leave equivalence unproven", report)
	}
}

func TestMissingStaysDistinctFromNull(t *testing.T) {
	cases := oracleCases()
	cases[3].ResultState = StateMissing // NULL input reported as missing
	cases[3].Amount = ""
	in := contract()
	in.Realizations = []Realization{{Plugin: "p", Version: "1", Language: "go", OutputType: "amount",
		NullPolicy: NullPropagate, Rounding: RoundHalfEven, Pure: true, Registered: true, Locked: true,
		Cases: cases}}
	if report := Check(in); report.Code != CodeFixtureMismatch {
		t.Fatalf("report=%+v want FIXTURE_MISMATCH", report)
	}
}

func TestTwoEquivalentImplementationsProveEquivalence(t *testing.T) {
	sqlRealization := Realization{
		Plugin: "case-line-amount-sql", Version: "2", Language: "sql",
		OutputType: "amount", NullPolicy: NullPropagate, Rounding: RoundHalfEven,
		Pure: true, Registered: true, Locked: true, LockedVersion: "2",
		Cases: oracleCases(),
	}
	in := contract()
	in.Realizations = []Realization{goRealization(), sqlRealization}
	report := Check(in)
	if !report.Accepted || report.Code != CodeCompliant || report.EquivalenceUnproven {
		t.Fatalf("report=%+v want COMPLIANT with proven equivalence", report)
	}
	if len(report.Implementations) != 2 || report.Implementations[0] != "go" || report.Implementations[1] != "sql" {
		t.Fatalf("implementations=%v want [go sql]", report.Implementations)
	}
}

func TestDivergingImplementationsCountAsUnproven(t *testing.T) {
	// The SQL realization covers fewer inputs than the Go realization, so the
	// shared oracle passes both but equivalence is not proven on the same set.
	sqlCases := oracleCases()[:4]
	sqlRealization := Realization{
		Plugin: "case-line-amount-sql", Version: "2", Language: "sql",
		OutputType: "amount", NullPolicy: NullPropagate, Rounding: RoundHalfEven,
		Pure: true, Registered: true, Locked: true, LockedVersion: "2",
		Cases: sqlCases,
	}
	in := contract()
	in.Realizations = []Realization{goRealization(), sqlRealization}
	report := Check(in)
	if report.Accepted || report.Code != CodeEquivalenceUnproven {
		t.Fatalf("report=%+v want EQUIVALENCE_UNPROVEN", report)
	}
}

func TestRoundingModeChangeIsNotAnEquivalentReplacement(t *testing.T) {
	goRealization := goRealization()
	goRealization.Rounding = RoundHalfUp
	in := contract()
	in.Realizations = []Realization{goRealization}
	if report := Check(in); report.Code != CodeMeaningMismatch {
		t.Fatalf("report=%+v want MEANING_MISMATCH", report)
	}
}

func TestContractMismatchesAreRejected(t *testing.T) {
	mutate := func(fn func(r *Realization)) Input {
		in := contract()
		r := goRealization()
		fn(&r)
		in.Realizations = []Realization{r}
		return in
	}
	{
		in := mutate(func(r *Realization) { r.OutputType = "text" })
		if report := Check(in); report.Code != CodeTypeMismatch {
			t.Fatalf("report=%+v want TYPE_MISMATCH", report)
		}
	}
	{
		in := mutate(func(r *Realization) { r.NullPolicy = NullZero })
		if report := Check(in); report.Code != CodeNullPolicyMismatch {
			t.Fatalf("report=%+v want NULL_POLICY_MISMATCH", report)
		}
	}
	{
		in := mutate(func(r *Realization) { r.Pure = false })
		if report := Check(in); report.Code != CodeImpureRealization {
			t.Fatalf("report=%+v want IMPURE_REALIZATION", report)
		}
	}
}

func TestIdentityMismatchesAreRejected(t *testing.T) {
	mutate := func(fn func(r *Realization)) Input {
		in := contract()
		r := goRealization()
		fn(&r)
		in.Realizations = []Realization{r}
		return in
	}
	{
		in := mutate(func(r *Realization) { r.Registered = false })
		if report := Check(in); report.Code != CodeRealizationNotRegistered {
			t.Fatalf("report=%+v want REALIZATION_NOT_REGISTERED", report)
		}
	}
	{
		in := mutate(func(r *Realization) { r.Locked = false })
		if report := Check(in); report.Code != CodeRealizationLockMissing {
			t.Fatalf("report=%+v want REALIZATION_LOCK_MISSING", report)
		}
	}
	{
		in := mutate(func(r *Realization) { r.LockedVersion = "4" })
		if report := Check(in); report.Code != CodeRealizationVersionMismatch {
			t.Fatalf("report=%+v want REALIZATION_VERSION_MISMATCH", report)
		}
	}
}

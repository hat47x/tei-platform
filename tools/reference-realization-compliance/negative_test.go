package realizationcompliance

import "testing"

func sqlRealization() Realization {
	return Realization{
		Plugin: "case-line-amount-sql", Version: "2", Language: "sql",
		OutputType: "amount", NullPolicy: NullPropagate, Rounding: RoundHalfEven,
		Pure: true, Registered: true, Locked: true, LockedVersion: "2",
		Cases: oracleCases(),
	}
}

func wasmRealization() Realization {
	return Realization{
		Plugin: "case-line-amount-wasm", Version: "1", Language: "wasm",
		OutputType: "amount", NullPolicy: NullPropagate, Rounding: RoundHalfEven,
		Pure: true, Registered: true, Locked: true, LockedVersion: "1",
		Cases: oracleCases(),
	}
}

func TestUnpermittedRealizationIsRejected(t *testing.T) {
	in := contract()
	r := goRealization()
	r.PermissionDenied = true
	in.Realizations = []Realization{r}
	report := Check(in)
	if report.Accepted || report.Code != CodeRealizationNotPermitted {
		t.Fatalf("report=%+v want REALIZATION_NOT_PERMITTED", report)
	}
}

// The rule identity is fixed. Replacing the implementation identity (plugin,
// version, language) does not change the rule that the fixture is checked
// against.
func TestRuleIdentityStaysFixedWhenImplementationIdentityIsSwapped(t *testing.T) {
	goIn := contract()
	goIn.Realizations = []Realization{goRealization()}
	goReport := Check(goIn)
	if !goReport.Accepted || goReport.RuleID != "line-amount" {
		t.Fatalf("go report=%+v", goReport)
	}

	sqlIn := contract()
	sqlIn.Realizations = []Realization{sqlRealization()}
	sqlReport := Check(sqlIn)
	if !sqlReport.Accepted || sqlReport.RuleID != goReport.RuleID {
		t.Fatalf("sql report=%+v want the same rule identity", sqlReport)
	}
	if sqlReport.Implementations[0] != "sql" {
		t.Fatalf("implementation=%v", sqlReport.Implementations)
	}
}

func TestCaseSpecificWasmRealizationPassesContractFixture(t *testing.T) {
	in := contract()
	in.Realizations = []Realization{goRealization(), wasmRealization()}
	report := Check(in)
	if !report.Accepted || report.Code != CodeCompliant || report.EquivalenceUnproven {
		t.Fatalf("report=%+v want COMPLIANT with proven equivalence", report)
	}
	if len(report.Implementations) != 2 || report.Implementations[1] != "wasm" {
		t.Fatalf("implementations=%v want [go wasm]", report.Implementations)
	}
}

func TestThirdTargetWithNarrowerCoverageIsUnproven(t *testing.T) {
	wasm := wasmRealization()
	wasm.Cases = oracleCases()[:4] // same oracle points, narrower coverage
	in := contract()
	in.Realizations = []Realization{goRealization(), sqlRealization(), wasm}
	report := Check(in)
	if report.Accepted || report.Code != CodeEquivalenceUnproven {
		t.Fatalf("report=%+v want EQUIVALENCE_UNPROVEN", report)
	}
}

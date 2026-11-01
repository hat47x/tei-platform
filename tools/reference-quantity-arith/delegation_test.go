package quantityarith

import (
	"reflect"
	"testing"
)

// The D8 fixture declares JPY with 5 digits and scale 2, rounded HALF_EVEN.
func delegationModes() []string {
	return []string{RoundHalfUp, RoundHalfEven}
}

func delegationInput() DelegationInput {
	return DelegationInput{
		Target: TargetCapability{
			Name: "reference-sql", MaxPrecision: 18, MaxScale: 4,
			RoundingModes: delegationModes(), ExactDecimal: true,
		},
		ResultType: "JPY",
		Types: map[string]TypeAnnotation{
			"JPY": {Currency: "JPY", Precision: intPtr(5), Scale: intPtr(2)},
		},
		Rounding: RoundHalfEven,
	}
}

func hasReason(report DelegationReport, prefix string) bool {
	for _, reason := range report.Reasons {
		if len(reason) >= len(prefix) && reason[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func TestDelegationReadyWhenTargetKeepsTheDeclaredMeaning(t *testing.T) {
	report := CheckDelegation(delegationInput())
	if report.Status != DelegationReady {
		t.Fatalf("report=%+v want %s", report, DelegationReady)
	}
	if len(report.Reasons) != 0 {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestDelegationPrecisionOverflowIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Types["JPY"] = TypeAnnotation{Currency: "JPY", Precision: intPtr(30), Scale: intPtr(2)}
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "precision-exceeds-target:") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationScaleOverflowIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Types["JPY"] = TypeAnnotation{Currency: "JPY", Precision: intPtr(5), Scale: intPtr(9)}
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "scale-exceeds-target:") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationUnsupportedRoundingIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Target.RoundingModes = []string{RoundHalfUp}
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "rounding-mode-unsupported:") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationToFloatTargetIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Target.ExactDecimal = false
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "target-not-exact-decimal") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationWithoutDeclaredPrecisionIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Types["JPY"] = TypeAnnotation{Currency: "JPY"}
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven {
		t.Fatalf("report=%+v", report)
	}
	if !hasReason(report, "precision-not-declared") || !hasReason(report, "scale-not-declared") {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestDelegationUnknownResultTypeIsUnproven(t *testing.T) {
	in := delegationInput()
	in.ResultType = "missing"
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "result-type-unknown:") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationUndeclaredTargetIsUnproven(t *testing.T) {
	in := delegationInput()
	in.Target.Name = ""
	report := CheckDelegation(in)
	if report.Status != DelegationUnproven || !hasReason(report, "target-not-declared") {
		t.Fatalf("report=%+v", report)
	}
}

func TestDelegationIsDeterministic(t *testing.T) {
	in := delegationInput()
	in.Types["JPY"] = TypeAnnotation{Currency: "JPY", Precision: intPtr(30), Scale: intPtr(9)}
	first := CheckDelegation(in)
	second := CheckDelegation(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("delegation is not deterministic: %+v != %+v", first, second)
	}
	// All unmet conditions are reported, not only the first.
	if len(first.Reasons) < 2 {
		t.Fatalf("reasons=%v want every unmet condition", first.Reasons)
	}
}

func TestNegativeTieRoundingIsDeclaredAndDistinct(t *testing.T) {
	base := func(mode string) Input {
		return Input{
			Currencies: []string{"JPY"},
			Types:      map[string]TypeAnnotation{"JPY": {Currency: "JPY"}},
			Operation: Operation{
				Op:       OpRound,
				Left:     Operand{Type: "JPY", State: "present", Decimal: "-1.005"},
				Rounding: mode,
				Scale:    intPtr(2),
			},
		}
	}
	halfUp := Evaluate(base(RoundHalfUp))
	halfEven := Evaluate(base(RoundHalfEven))
	if !halfUp.Accepted || halfUp.ResultDecimal != "-1.01" {
		t.Fatalf("HALF_UP=%+v want -1.01 (away from zero)", halfUp)
	}
	if !halfEven.Accepted || halfEven.ResultDecimal != "-1.00" {
		t.Fatalf("HALF_EVEN=%+v want -1.00 (tie to even)", halfEven)
	}
	if halfUp.ResultDecimal == halfEven.ResultDecimal {
		t.Fatal("rounding modes were treated as interchangeable")
	}
}

func TestNegativeExactMultiplicationStaysExact(t *testing.T) {
	in := Input{
		Currencies: []string{"JPY"},
		Types: map[string]TypeAnnotation{
			"count":  {Unit: "個"},
			"rate":   {Currency: "JPY", Per: "個"},
			"amount": {Currency: "JPY"},
		},
		Operation: Operation{
			Op:    OpMultiply,
			Left:  Operand{Type: "count", State: "present", Decimal: "-2"},
			Right: &Operand{Type: "rate", State: "present", Decimal: "300"},
		},
	}
	report := Evaluate(in)
	if !report.Accepted || report.ResultDecimal != "-600" || report.Currency != "JPY" {
		t.Fatalf("report=%+v want -600 JPY", report)
	}
}

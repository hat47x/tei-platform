package derivedvalidation

import "testing"

func base() Input {
	return Input{
		ReferenceByID: true,
		Authorization: "allowed",
		Derivation:    Derivation{State: DerivationPresent, Amount: "600"},
		Limit:         Operand{State: "present", Value: "700"},
		Phase:         PhaseSave,
	}
}

// The fixtures follow Data Derivation D15 (#1729): order O1 with quantity 2 and
// unit price 300 derives an amount of 600.

func TestLimitBelowAmountIsNotSatisfied(t *testing.T) {
	in := base()
	in.Limit.Value = "500"
	report := Evaluate(in)
	if report.Code != CodeValidationNotSatisfied || report.Validation != "not-satisfied" {
		t.Fatalf("report=%+v want NOT_SATISFIED", report)
	}
}

func TestLimitAboveAmountIsSatisfiedAtSave(t *testing.T) {
	report := Evaluate(base())
	if report.Code != CodeValidationSatisfied || report.Validation != "satisfied" || !report.SaveGuaranteed {
		t.Fatalf("report=%+v want satisfied and save-guaranteed", report)
	}
}

func TestPreviewIsNotASaveGuarantee(t *testing.T) {
	in := base()
	in.Phase = PhasePreview
	in.Derivation.Amount = "300"
	report := Evaluate(in)
	if report.Code != CodeValidationSatisfied || report.SaveGuaranteed {
		t.Fatalf("report=%+v want satisfied without save guarantee", report)
	}
}

func TestNullPolicyUndefinedIsNotNotSatisfied(t *testing.T) {
	in := base()
	in.Derivation = Derivation{State: DerivationNull}
	report := Evaluate(in)
	if report.Code != CodeDerivationNullUnspecified || report.Validation == "not-satisfied" {
		t.Fatalf("report=%+v want DERIVATION_NULL_UNSPECIFIED", report)
	}
}

func TestInputUnavailableIsDistinct(t *testing.T) {
	in := base()
	in.Derivation = Derivation{State: DerivationInputUnavailable}
	report := Evaluate(in)
	if report.Code != CodeDerivationInputUnavailable || report.Validation == "not-satisfied" {
		t.Fatalf("report=%+v want DERIVATION_INPUT_UNAVAILABLE", report)
	}
}

func TestExecutionFailureIsDistinct(t *testing.T) {
	in := base()
	in.Derivation = Derivation{State: DerivationExecutionFailure}
	report := Evaluate(in)
	if report.Code != CodeDerivationExecutionFailure {
		t.Fatalf("report=%+v want DERIVATION_EXECUTION_FAILURE", report)
	}
}

func TestLimitUnavailableIsDistinct(t *testing.T) {
	in := base()
	in.Limit = Operand{State: "input-unavailable"}
	report := Evaluate(in)
	if report.Code != CodeLimitUnavailable {
		t.Fatalf("report=%+v want LIMIT_UNAVAILABLE", report)
	}
}

func TestAuthorizationDeniedPrecedesEverything(t *testing.T) {
	in := base()
	in.Authorization = "denied"
	in.Cycle = true
	report := Evaluate(in)
	if report.Code != CodeAuthorizationDenied {
		t.Fatalf("report=%+v want AUTHORIZATION_DENIED", report)
	}
}

func TestCycleIsRejected(t *testing.T) {
	in := base()
	in.Cycle = true
	report := Evaluate(in)
	if report.Code != CodeDependencyCycle {
		t.Fatalf("report=%+v want DEPENDENCY_CYCLE", report)
	}
}

func TestSameNameReferenceIsRejected(t *testing.T) {
	in := base()
	in.ReferenceByID = false
	report := Evaluate(in)
	if report.Code != CodeAmbiguousReference {
		t.Fatalf("report=%+v want AMBIGUOUS_REFERENCE", report)
	}
}

func TestInvalidAmountIsRejected(t *testing.T) {
	in := base()
	in.Derivation.Amount = "not-a-number"
	report := Evaluate(in)
	if report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

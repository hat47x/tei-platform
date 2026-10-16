package derivationvalidation

import (
	"errors"
	"testing"

	"tei/internal/semantic/derivationref"
)

func present(decimal string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: decimal}
}

func missingInputError() error {
	_, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Missing},
		present("300"),
	)
	return err
}

func operatorError() error {
	_, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: "abc"},
		present("300"),
	)
	return err
}

func TestD15SatisfiedAndNotSatisfied(t *testing.T) {
	over, err := Check(present("600"), nil, false, present("500"))
	if err != nil || over.Derivation != DerivationValue || over.Validation != ValidationNotSatisfied {
		t.Fatalf("600 > 500: %+v, %v", over, err)
	}
	ok, err := Check(present("600"), nil, false, present("700"))
	if err != nil || ok.Validation != ValidationSatisfied || ok.Value != "600" || ok.Limit != "700" {
		t.Fatalf("600 <= 700: %+v, %v", ok, err)
	}
}

func TestD15UnavailableIsNotNotSatisfied(t *testing.T) {
	result, err := Check(derivationref.Input{}, missingInputError(), false, present("500"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Derivation != DerivationUnavailable || result.Validation != ValidationNotEvaluated {
		t.Fatalf("missing input reduced to a validation verdict: %+v", result)
	}
}

func TestD15ExecutionFailureDistinct(t *testing.T) {
	result, err := Check(derivationref.Input{}, operatorError(), false, present("500"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Derivation != DerivationFailure || result.Validation != ValidationNotEvaluated {
		t.Fatalf("evaluation failure reduced to a verdict: %+v", result)
	}
}

func TestD15DeclaredNullIsNotNotSatisfied(t *testing.T) {
	result, err := Check(derivationref.Input{State: derivationref.Null}, nil, false, present("500"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Derivation != DerivationNull || result.Validation != ValidationNotEvaluated {
		t.Fatalf("declared NULL became a verdict: %+v", result)
	}
}

func TestD15DeniedIsDistinct(t *testing.T) {
	result, err := Check(present("600"), nil, true, present("500"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Derivation != DerivationDenied || result.Validation != ValidationNotEvaluated {
		t.Fatalf("authorization denial treated as a verdict: %+v", result)
	}
}

func TestD15LimitUnavailableFailsClosed(t *testing.T) {
	result, err := Check(present("600"), nil, false, derivationref.Input{State: derivationref.Missing})
	if !errors.Is(err, ErrLimitUnavailable) {
		t.Fatalf("unavailable limit accepted: %v", err)
	}
	if result.Derivation != DerivationValue || result.Validation != ValidationNotEvaluated {
		t.Fatalf("limit problem turned into a verdict: %+v", result)
	}
}

func TestD15PreviewRecheckedAtCommit(t *testing.T) {
	previewResult, err := Check(present("300"), nil, false, present("700"))
	if err != nil || previewResult.Validation != ValidationSatisfied {
		t.Fatalf("preview: %+v, %v", previewResult, err)
	}
	preview := Decision{Result: previewResult, InputVersion: "v1", Phase: PreviewPhase}

	// The limit changed from 700 to 500 before commit: the old preview must
	// not authorize the save.
	if _, err := RecheckAtCommit(preview, "v2", previewResult); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("stale preview was reused: %v", err)
	}

	// The same input version is re-evaluated and the fresh result is used.
	fresh, err := Check(present("600"), nil, false, present("500"))
	if err != nil {
		t.Fatal(err)
	}
	commit, err := RecheckAtCommit(preview, "v1", fresh)
	if err != nil || commit.Phase != CommitPhase || commit.Result.Validation != ValidationNotSatisfied {
		t.Fatalf("commit decision reused the preview: %+v, %v", commit, err)
	}
	if _, err := RecheckAtCommit(commit, "v1", fresh); !errors.Is(err, ErrPhase) {
		t.Fatalf("a commit decision was accepted as a preview: %v", err)
	}
}

func TestD15OperandIdentityIgnoresName(t *testing.T) {
	primary := DerivedRef{RuleID: "sales.line_amount", OutputID: "sales.order_line.amount", Name: "amount"}
	decoy := DerivedRef{RuleID: "sales.tax_amount", OutputID: "sales.order_line.tax", Name: "amount"}
	if err := ValidateRef(primary); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRef(decoy); err != nil {
		t.Fatal(err)
	}
	if primary.RuleID == decoy.RuleID || primary.OutputID == decoy.OutputID {
		t.Fatal("operands with the same label were not kept distinct")
	}
	if err := ValidateRef(DerivedRef{Name: "amount"}); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("name-only operand was accepted: %v", err)
	}
}

func TestD15ValidationCycleIsRejectedBeforeExecution(t *testing.T) {
	defs := []derivationref.Definition{
		{ID: "sales.line_amount", Pure: true, Operator: "multiply"},
		{ID: "sales.credit_available", Pure: true, Operator: "multiply"},
	}
	// The validation consumes line_amount to produce credit_available, and the
	// rule line_amount depends on credit_available: a cycle the existing D1
	// graph must reject before any evaluation.
	deps := map[string][]string{
		"sales.line_amount":      {"sales.credit_available"},
		"sales.credit_available": {"sales.line_amount"},
	}
	if err := derivationref.ValidateGraph(defs, deps); !errors.Is(err, derivationref.ErrCycle) {
		t.Fatalf("dependency cycle accepted: %v", err)
	}
}

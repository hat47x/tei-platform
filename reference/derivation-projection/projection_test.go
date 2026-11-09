package derivationprojection

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic/derivationref"
)

func present(decimal string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: decimal}
}

func multiply(quantity, unitPrice derivationref.Input) (derivationref.Input, error) {
	return derivationref.Multiply(quantity, unitPrice)
}

func missingPriceError() error {
	_, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Missing},
		present("300"),
	)
	return err
}

func operatorError() error {
	_, err := derivationref.Multiply(present("abc"), present("300"))
	return err
}

func derivedField() Field {
	return Field{ID: "sales.order_line.amount", Kind: Derived, RuleID: "sales.line_amount_v1"}
}

func TestD10PreviewDoesNotChangeConfirmedFact(t *testing.T) {
	stored := "600"
	field := derivedField().ID

	atTwo := PreviewValue(field, present("2"), present("300"), multiply)
	if atTwo.State != Preview || atTwo.Amount != "600" {
		t.Fatalf("preview at quantity 2: %+v", atTwo)
	}
	atThree := PreviewValue(field, present("3"), present("300"), multiply)
	if atThree.State != Preview || atThree.Amount != "900" {
		t.Fatalf("unsaved quantity change: %+v", atThree)
	}

	confirmed := ConfirmedValue(field, &stored)
	if confirmed.State != Confirmed || confirmed.Amount != "600" {
		t.Fatalf("stored fact changed by preview: %+v", confirmed)
	}
	if stored != "600" {
		t.Fatalf("stored value mutated: %s", stored)
	}
}

func TestD10ConfirmedSurvivesMasterPriceChange(t *testing.T) {
	field := derivedField().ID
	stored := "600"

	masterChanged := Evaluate(field, present("2"), present("350"), multiply)
	if masterChanged.State != Evaluated || masterChanged.Amount != "700" {
		t.Fatalf("new unconfirmed computation: %+v", masterChanged)
	}
	confirmed := ConfirmedValue(field, &stored)
	if confirmed.State != Confirmed || confirmed.Amount != "600" {
		t.Fatalf("confirmed fact followed the master change: %+v", confirmed)
	}
}

func TestD10DerivedDirectWriteRejectedAtBoundary(t *testing.T) {
	if err := WriteTarget(derivedField(), "999"); !errors.Is(err, ErrDerivedNotWritable) {
		t.Fatalf("direct write to a derived cell was accepted: %v", err)
	}
	physical := Field{ID: "sales.order_line.quantity", Kind: PhysicalStored}
	if err := WriteTarget(physical, "3"); err != nil {
		t.Fatalf("physical write rejected: %v", err)
	}
	confirmed := Field{ID: "sales.order_line.confirmed_amount", Kind: ConfirmedStored}
	if err := WriteTarget(confirmed, "600"); err != nil {
		t.Fatalf("confirmed write rejected: %v", err)
	}
}

func TestD10UnavailableDoesNotShowStaleValue(t *testing.T) {
	field := derivedField().ID
	// The previous successful display was 600; a failed input must not keep it.
	if !errors.Is(missingPriceError(), derivationref.ErrInput) {
		t.Fatalf("fixture precondition: %v", missingPriceError())
	}
	value := PreviewValue(field, derivationref.Input{State: derivationref.Missing}, present("300"), multiply)
	if value.State != Unavailable || value.Amount != "" {
		t.Fatalf("unavailable input showed a stale value: %+v", value)
	}
}

func TestD10DeniedAndFailureStates(t *testing.T) {
	field := derivedField().ID
	if denied := Denied(field); denied.State != NotAuthorized || denied.Amount != "" {
		t.Fatalf("denied state: %+v", denied)
	}
	failed := PreviewValue(field, present("abc"), present("300"), multiply)
	if failed.State != EvaluationFailed || failed.Amount != "" {
		t.Fatalf("evaluation failure: %+v", failed)
	}
	if !errors.Is(operatorError(), derivationref.ErrOperator) {
		t.Fatalf("fixture precondition: %v", operatorError())
	}
}

func TestD10RendererPortability(t *testing.T) {
	field := derivedField().ID
	value := Evaluate(field, present("2"), present("350"), multiply)

	web := WebRenderer{}.Render(value)
	cli := CLIRenderer{}.Render(value)
	if meaning(web) != meaning(cli) {
		t.Fatalf("renderers disagree on the meaning: %s vs %s", web, cli)
	}
	if !strings.Contains(meaning(web), string(Evaluated)) {
		t.Fatalf("renderer lost the meaning state: %s", web)
	}

	unavailable := PreviewValue(field, derivationref.Input{State: derivationref.Missing}, present("300"), multiply)
	if meaning(WebRenderer{}.Render(unavailable)) != meaning(CLIRenderer{}.Render(unavailable)) {
		t.Fatalf("renderers disagree on an unavailable state")
	}
	if !strings.HasSuffix(WebRenderer{}.Render(unavailable), "|") {
		t.Fatal("an unavailable cell rendered an amount")
	}
}

// meaning returns the target-independent part of a rendered cell.
func meaning(rendered string) string {
	parts := strings.SplitN(rendered, "|", 2)
	if len(parts) != 2 {
		return rendered
	}
	return parts[1]
}

func TestD10ReprojectionKeepsFieldIdentity(t *testing.T) {
	before := derivedField()
	after, err := Reproject(before, "sales.line_amount_v2")
	if err != nil {
		t.Fatal(err)
	}
	if after.ID != before.ID {
		t.Fatalf("redeployment changed the business identity: %s -> %s", before.ID, after.ID)
	}
	if after.RuleID != "sales.line_amount_v2" {
		t.Fatalf("rule revision not applied: %+v", after)
	}
	physical := Field{ID: "sales.order_line.quantity", Kind: PhysicalStored}
	if _, err := Reproject(physical, "v2"); !errors.Is(err, ErrRuleNotDerived) {
		t.Fatalf("reprojection accepted a stored field: %v", err)
	}
}

func TestD10RuleEditIsNotAValueWrite(t *testing.T) {
	field := derivedField()
	if err := ValidateRuleEdit(field, RuleEdit{FieldID: field.ID, ProposedRuleID: "sales.line_amount_v2"}); err != nil {
		t.Fatalf("valid rule edit rejected: %v", err)
	}
	if err := ValidateRuleEdit(field, RuleEdit{FieldID: field.ID}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("empty rule proposal accepted: %v", err)
	}
	physical := Field{ID: "sales.order_line.quantity", Kind: PhysicalStored}
	if err := ValidateRuleEdit(physical, RuleEdit{FieldID: physical.ID, ProposedRuleID: "v2"}); !errors.Is(err, ErrRuleNotDerived) {
		t.Fatalf("rule edit accepted on a stored field: %v", err)
	}
	// A rule edit is not a value write; the derived cell stays read-only.
	if err := WriteTarget(field, "999"); !errors.Is(err, ErrDerivedNotWritable) {
		t.Fatalf("rule edit exposed a value write: %v", err)
	}
}

func TestD10FieldValidation(t *testing.T) {
	if err := (Field{ID: "x", Kind: Derived}).Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("derived field without a rule identity accepted: %v", err)
	}
	if err := (Field{ID: "x", Kind: PhysicalStored, RuleID: "r"}).Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("stored field with a rule identity accepted: %v", err)
	}
	if err := (Field{ID: "", Kind: PhysicalStored}).Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("field without an id accepted: %v", err)
	}
}

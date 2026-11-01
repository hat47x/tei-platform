package derivedfieldprojection

import (
	"reflect"
	"testing"
)

// The D10 fixture: order line quantity 2 × unit price 300 = derived amount 600.
// After confirmation, 600 is the business-confirmed saved fact. The current
// price later becomes 350, so a fresh runtime evaluation would be 700.

func derivedField() Field {
	return Field{ID: "order-line.amount", Kind: KindDerived}
}

func preview() Input {
	return Input{
		Field:         derivedField(),
		Phase:         PhasePreview,
		Operation:     OpDisplay,
		DerivedState:  DerivedPresent,
		DerivedValue:  "900",
		SavedValue:    "600",
		InputsUnsaved: true,
		Renderers:     []string{"web", "cli"},
	}
}

func TestDeclarationMarksDerivedFieldReadOnly(t *testing.T) {
	in := preview()
	in.Phase = PhaseDeclaration
	in.DerivedState = ""
	in.DerivedValue = ""
	report := Evaluate(in)
	if report.Code != CodeProjectionDeclared || report.Editable {
		t.Fatalf("report=%+v want PROJECTION_DECLARED not editable", report)
	}
	if report.SavedValueUnchanged == false {
		t.Fatal("a declaration must not claim it changed the saved fact")
	}
}

func TestDirectWriteToDerivedFieldIsRejectedAtSaveBoundary(t *testing.T) {
	for _, phase := range []string{PhasePreview, PhaseRuntime} {
		in := preview()
		in.Phase = phase
		in.Operation = OpWrite
		in.AttemptedValue = "999"
		report := Evaluate(in)
		if report.Code != CodeDerivedReadOnly {
			t.Errorf("phase %s: code=%s want %s", phase, report.Code, CodeDerivedReadOnly)
		}
		if report.SavedValueUnchanged == false {
			t.Errorf("phase %s: a rejected write reported a saved-value change", phase)
		}
		if report.Value == "999" {
			t.Errorf("phase %s: the rejected value was accepted", phase)
		}
	}
}

func TestPreviewReevaluatesUnsavedInputsWithoutChangingSavedValue(t *testing.T) {
	report := Evaluate(preview())
	if report.Code != CodePreviewPresent || report.Value != "900" {
		t.Fatalf("report=%+v want PREVIEW_PRESENT 900", report)
	}
	if !report.SavedValueUnchanged {
		t.Fatal("preview changed the saved value")
	}
	if report.SavedValue != "600" {
		t.Fatalf("savedValue=%q want 600", report.SavedValue)
	}
}

func TestCancelAfterPreviewLeavesSavedValueIntact(t *testing.T) {
	// Preview says 900, then the user cancels: the confirmed read still returns
	// 600 and never the preview value.
	previewReport := Evaluate(preview())
	if previewReport.Code != CodePreviewPresent {
		t.Fatalf("preview=%+v", previewReport)
	}
	confirmed := Input{
		Field: Field{ID: "order-line.amount", Kind: KindConfirmed},
		Phase: PhaseRuntime, Operation: OpDisplay,
		SavedValue: "600", DerivedValue: "900",
	}
	report := Evaluate(confirmed)
	if report.Code != CodeConfirmedSaved || report.Value != "600" {
		t.Fatalf("report=%+v want CONFIRMED_SAVED 600", report)
	}
	if !report.SavedValueUnchanged {
		t.Fatal("confirmed read reported a change")
	}
}

func TestUnavailableInputDoesNotReturnStaleValue(t *testing.T) {
	in := preview()
	in.DerivedState = DerivedInputUnavailable
	in.DerivedValue = "600" // a stale value the UI might still hold
	report := Evaluate(in)
	if report.Code != CodePreviewInputUnavailable {
		t.Fatalf("report=%+v", report)
	}
	if report.Value != "" {
		t.Fatalf("stale value %q was returned as a current result", report.Value)
	}
}

func TestAuthorizationDeniedReturnsNoValue(t *testing.T) {
	in := preview()
	in.DerivedState = DerivedAuthorization
	in.DerivedValue = "600"
	report := Evaluate(in)
	if report.Code != CodePreviewAuthorizationDenied || report.Value != "" {
		t.Fatalf("report=%+v", report)
	}
}

func TestComputationFailureReturnsNoValue(t *testing.T) {
	in := preview()
	in.DerivedState = DerivedComputationError
	in.DerivedValue = "600"
	report := Evaluate(in)
	if report.Code != CodePreviewComputationFailed || report.Value != "" {
		t.Fatalf("report=%+v", report)
	}
}

func TestRuntimePresentIsDistinctFromPreview(t *testing.T) {
	in := preview()
	in.Phase = PhaseRuntime
	in.DerivedValue = "700"
	report := Evaluate(in)
	if report.Code != CodeRuntimePresent || report.Value != "700" {
		t.Fatalf("report=%+v want RUNTIME_PRESENT 700", report)
	}
	if report.Code == CodePreviewPresent {
		t.Fatal("runtime result was labelled preview")
	}
}

func TestConfirmedFieldWriteRequiresBusinessAction(t *testing.T) {
	in := Input{
		Field: Field{ID: "order-line.amount", Kind: KindConfirmed},
		Phase: PhaseRuntime, Operation: OpWrite, AttemptedValue: "999", SavedValue: "600",
	}
	report := Evaluate(in)
	if report.Code != CodeConfirmedWriteRequiresAction {
		t.Fatalf("report=%+v want CONFIRMED_WRITE_REQUIRES_ACTION", report)
	}
}

func TestRuleEditIsNotAValueEdit(t *testing.T) {
	in := preview()
	in.Operation = OpEditRule
	report := Evaluate(in)
	if report.Code != CodeRuleEditRequiresAcceptance {
		t.Fatalf("report=%+v want RULE_EDIT_REQUIRES_ACCEPTANCE", report)
	}
}

func TestPhysicalFieldStaysWritable(t *testing.T) {
	in := preview()
	in.Field = Field{ID: "order-line.quantity", Kind: KindPhysical}
	in.Operation = OpWrite
	in.AttemptedValue = "3"
	report := Evaluate(in)
	if report.Code != CodePhysicalWritable || !report.Editable || report.Value != "3" {
		t.Fatalf("report=%+v want PHYSICAL_FIELD_WRITABLE 3", report)
	}
}

func TestSameSemanticStateForEveryRenderer(t *testing.T) {
	report := Evaluate(preview())
	if len(report.Renderers) != 2 {
		t.Fatalf("renderers=%v", report.Renderers)
	}
	for renderer, state := range report.Renderers {
		if state != report.ProjectionState {
			t.Errorf("renderer %s state=%q want %q", renderer, state, report.ProjectionState)
		}
	}
}

func TestUnknownRequestIsRejected(t *testing.T) {
	in := preview()
	in.Phase = "final"
	if report := Evaluate(in); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

func TestEvaluationIsDeterministic(t *testing.T) {
	in := preview()
	first := Evaluate(in)
	second := Evaluate(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic: %+v != %+v", first, second)
	}
}

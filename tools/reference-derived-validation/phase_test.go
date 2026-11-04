package derivedvalidation

import (
	"reflect"
	"testing"
)

// The D15 fixture: order O1 has quantity 2 × unit price 300 = amount 600.
// The credit limit is a typed, accepted input from the same snapshot.
func baseValidation() Input {
	return Input{
		ReferenceByID: true,
		Authorization: "allowed",
		Derivation:    Derivation{State: DerivationPresent, Amount: "600"},
		Limit:         Operand{State: "present", Value: "700"},
		Phase:         PhasePreview,
	}
}

// A preview success must not stand in for a save-time decision. The fixture
// changes the limit from 700 to 500 right before the save, so the same derived
// amount is not satisfied at save time.
func TestSaveReevaluatesChangedLimitInsteadOfReusingPreview(t *testing.T) {
	preview := Evaluate(baseValidation())
	if preview.Code != CodeValidationSatisfied {
		t.Fatalf("preview=%+v want satisfied", preview)
	}

	save := baseValidation()
	save.Phase = PhaseSave
	save.Limit = Operand{State: "present", Value: "500"}
	saveReport := Evaluate(save)
	if saveReport.Code != CodeValidationNotSatisfied {
		t.Fatalf("save=%+v want not satisfied", saveReport)
	}
	if saveReport.Code == preview.Code {
		t.Fatal("a save-time limit change was not reflected; the preview result leaked into save")
	}
	if preview.SaveGuaranteed || saveReport.SaveGuaranteed {
		t.Fatal("no synthetic phase may claim a save guarantee")
	}
}

func TestSaveWithSameLimitIsSatisfiedButStillNotASaveGuarantee(t *testing.T) {
	in := baseValidation()
	in.Phase = PhaseSave
	report := Evaluate(in)
	if report.Code != CodeValidationSatisfied {
		t.Fatalf("report=%+v want satisfied", report)
	}
	if report.SaveGuaranteed {
		t.Fatal("validation satisfied is not a commit or save guarantee")
	}
}

func TestUnknownPhaseIsNotTreatedAsPreview(t *testing.T) {
	for _, phase := range []string{"Save", "commit", "preivew"} {
		in := baseValidation()
		in.Phase = phase
		report := Evaluate(in)
		if report.Code != CodePhaseUnknown {
			t.Errorf("phase %q: code=%s want %s", phase, report.Code, CodePhaseUnknown)
		}
	}
}

func TestEmptyPhaseKeepsPreviewMeaning(t *testing.T) {
	in := baseValidation()
	in.Phase = ""
	report := Evaluate(in)
	if report.Code != CodeValidationSatisfied {
		t.Fatalf("report=%+v want satisfied", report)
	}
}

func TestPreviewSuccessDoesNotOverrideDeniedSave(t *testing.T) {
	preview := Evaluate(baseValidation())
	if preview.Code != CodeValidationSatisfied {
		t.Fatalf("preview=%+v", preview)
	}
	save := baseValidation()
	save.Phase = PhaseSave
	save.Authorization = "denied"
	if report := Evaluate(save); report.Code != CodeAuthorizationDenied {
		t.Fatalf("save=%+v want AUTHORIZATION_DENIED", report)
	}
}

func TestEvaluationIsDeterministic(t *testing.T) {
	in := baseValidation()
	in.Phase = PhaseSave
	first := Evaluate(in)
	second := Evaluate(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("evaluation is not deterministic: %+v != %+v", first, second)
	}
}

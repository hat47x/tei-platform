package derivationprojection

import "testing"

// The fixtures follow Data Derivation D10 (#1707): an order line has quantity 2
// and unit price 300, so the derived amount is 600.

func TestUnsavedInputChangeUpdatesPreviewOnly(t *testing.T) {
	report := Apply(Input{State: StatePreview, Value: "600", Event: EventEditInput, NewValue: "900"})
	if !report.Accepted || report.Code != CodePreviewUpdated || report.State != StatePreview || report.Value != "900" {
		t.Fatalf("report=%+v want PREVIEW_UPDATED 900", report)
	}
}

func TestDirectWriteToDerivedCellIsForbidden(t *testing.T) {
	for _, state := range []string{StatePreview, StateRuntimeDerived, StateConfirmedSaved} {
		report := Apply(Input{State: state, Value: "600", Event: EventWriteValue})
		if report.Accepted || report.Code != CodeEditForbidden {
			t.Fatalf("state=%s report=%+v want EDIT_FORBIDDEN", state, report)
		}
	}
}

func TestUnavailableInputDoesNotReuseStaleValue(t *testing.T) {
	report := Apply(Input{State: StatePreview, Value: "600", Event: EventReevaluate, Recompute: RecomputeInputUnavailable, NewValue: "700"})
	if report.Accepted || report.Code != CodeRecomputeUnavailable || !report.ValueNull {
		t.Fatalf("report=%+v want RECOMPUTE_UNAVAILABLE with unknown value", report)
	}
}

func TestAuthorizationDeniedDoesNotShowConfirmedValue(t *testing.T) {
	report := Apply(Input{State: StateRuntimeDerived, Value: "600", Event: EventReevaluate, Recompute: RecomputeAuthorizationDenied})
	if report.Accepted || report.Code != CodeAuthorizationDenied || !report.ValueNull {
		t.Fatalf("report=%+v want AUTHORIZATION_DENIED with unknown value", report)
	}
}

func TestConfirmedSavedValueSurvivesMasterChange(t *testing.T) {
	report := Apply(Input{State: StateConfirmedSaved, Value: "600", Event: EventMasterChange, NewValue: "700"})
	if !report.Accepted || report.Code != CodeSavedFactImmutable || report.Value != "600" {
		t.Fatalf("report=%+v want SAVED_FACT_IMMUTABLE 600", report)
	}
}

func TestConfirmedSavedValueSurvivesReevaluate(t *testing.T) {
	report := Apply(Input{State: StateConfirmedSaved, Value: "600", Event: EventReevaluate, NewValue: "700"})
	if !report.Accepted || report.Code != CodeSavedFactImmutable || report.Value != "600" {
		t.Fatalf("report=%+v want SAVED_FACT_IMMUTABLE 600", report)
	}
}

func TestConfirmFreezesTheValue(t *testing.T) {
	report := Apply(Input{State: StateRuntimeDerived, Value: "600", Event: EventConfirm, NewValue: "600"})
	if !report.Accepted || report.Code != CodeConfirmedSaved || report.State != StateConfirmedSaved || report.Value != "600" {
		t.Fatalf("report=%+v want CONFIRMED_SAVED 600", report)
	}
}

func TestConfirmWithUnavailableInputIsRejected(t *testing.T) {
	report := Apply(Input{State: StatePreview, Value: "600", Event: EventConfirm, Recompute: RecomputeComputeError, NewValue: "0"})
	if report.Accepted || report.Code != CodeRecomputeUnavailable {
		t.Fatalf("report=%+v want RECOMPUTE_UNAVAILABLE", report)
	}
}

func TestProposalBeforeAcceptanceDoesNotApply(t *testing.T) {
	report := Apply(Input{State: StateRuntimeDerived, Value: "600", Event: EventPropose, NewValue: "700"})
	if !report.Accepted || report.Code != CodeProposalNotApplied || report.Value != "600" {
		t.Fatalf("report=%+v want PROPOSAL_NOT_APPLIED 600", report)
	}
	report = Apply(Input{State: StateDeclared, Event: EventAccept, NewValue: "700"})
	if !report.Accepted || report.Code != CodeProposalApplied || report.State != StateRuntimeDerived || report.Value != "700" {
		t.Fatalf("report=%+v want PROPOSAL_APPLIED 700", report)
	}
}

func TestRendererKeepsTheSameMeaningState(t *testing.T) {
	for _, renderer := range []string{"web", "cli", "desktop", "case-specific-ui"} {
		report := Apply(Input{State: StateRuntimeDerived, Value: "600", Event: EventRender, Renderer: renderer, RendererValue: "600"})
		if !report.Accepted || report.Code != CodeRendererAgnostic || report.Value != "600" {
			t.Fatalf("renderer=%s report=%+v want RENDERER_AGNOSTIC 600", renderer, report)
		}
	}
}

func TestRendererContractViolationIsRejected(t *testing.T) {
	report := Apply(Input{State: StateRuntimeDerived, Value: "600", Event: EventRender, Renderer: "web", RendererValue: "999"})
	if report.Accepted || report.Code != CodeRendererContractViolation {
		t.Fatalf("report=%+v want RENDERER_CONTRACT_VIOLATION", report)
	}
}

func TestUnknownStateAndEventAreRejected(t *testing.T) {
	if report := Apply(Input{State: "nonsense", Event: EventReevaluate}); report.Accepted || report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
	if report := Apply(Input{State: StatePreview, Event: "nonsense"}); report.Accepted || report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

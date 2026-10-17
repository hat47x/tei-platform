package derivationevidence

import "testing"

// The fixtures follow Data Derivation D7 (#1696): quantity 2 × unit price 300
// gives amount 600 via rule line-amount on target order-line.amount.

func admin() string { return RequesterAdmin }

func TestPresentValueExplainsItsEvidence(t *testing.T) {
	report := Explain(Input{
		RuleID:             "line-amount",
		AcceptedRevision:   "R",
		TargetID:           "order-line.amount",
		Realization:        "X",
		RealizationVersion: "1",
		InputID:            "order-line",
		InputVersion:       "7",
		InputVersionKnown:  true,
		ContextClass:       "normal",
		ResultState:        StatePresent,
		InputValue:         "2",
		UnitPrice:          "300",
		Requester:          admin(),
	})
	if !report.Accepted || report.Code != CodePresent || report.RuleID != "line-amount" || report.TargetID != "order-line.amount" {
		t.Fatalf("report=%+v want RESULT_PRESENT with rule/target", report)
	}
	if !report.Traceable {
		t.Fatalf("report=%+v want traceable", report)
	}
}

func TestMissingNullAndUndeclaredNullStayDistinct(t *testing.T) {
	make := func(state string) Report {
		return Explain(Input{RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
			InputVersionKnown: true, ResultState: state, Requester: admin()})
	}
	missing := make(StateInputMissing)
	declared := make(StateNullDeclared)
	undeclared := make(StateNullUndeclared)
	if missing.Code != CodeInputMissing || declared.Code != CodeNullDeclared || undeclared.Code != CodeNullUndeclared {
		t.Fatalf("codes missing=%s declared=%s undeclared=%s", missing.Code, declared.Code, undeclared.Code)
	}
	if missing.Code == declared.Code || declared.Code == undeclared.Code || missing.Code == undeclared.Code {
		t.Fatalf("diagnostics collapsed: %s %s %s", missing.Code, declared.Code, undeclared.Code)
	}
}

func TestSourceMoveKeepsIdentity(t *testing.T) {
	report := Explain(Input{RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		InputVersionKnown: true, ResultState: StatePresent, Requester: admin(), SourceMoved: true})
	if report.RuleID != "line-amount" || report.Code != CodePresent {
		t.Fatalf("report=%+v want same identity", report)
	}
	if report.Message == "" {
		t.Fatalf("report=%+v want a note that identity is unchanged", report)
	}
}

func TestGeneralRequesterDoesNotReceiveHiddenInputs(t *testing.T) {
	in := Input{
		RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		InputVersionKnown: true, ResultState: StatePresent,
		InputValue: "2", UnitPrice: "300",
		Requester: RequesterGeneral,
	}
	report := Explain(in)
	if report.InputValue != "" || report.UnitPrice != "" {
		t.Fatalf("report=%+v leaked a hidden input to a general requester", report)
	}
	if len(report.Redacted) == 0 {
		t.Fatalf("report=%+v want redacted fields", report)
	}
}

func TestAdminReceivesInputs(t *testing.T) {
	report := Explain(Input{RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		InputVersionKnown: true, ResultState: StatePresent, InputValue: "2", UnitPrice: "300", Requester: admin()})
	if report.InputValue != "2" || report.UnitPrice != "300" {
		t.Fatalf("report=%+v want disclosed inputs for admin", report)
	}
}

func TestUnknownInputVersionIsNotAFullReproduction(t *testing.T) {
	report := Explain(Input{RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		InputVersionKnown: false, ResultState: StatePresent, Requester: admin()})
	if report.Traceable {
		t.Fatalf("report=%+v must not claim a full reproduction", report)
	}
	if len(report.Unknown) != 1 || report.Unknown[0] != "inputVersion" {
		t.Fatalf("unknown=%v want [inputVersion]", report.Unknown)
	}
}

func TestUnauthorizedDisclosesNoValue(t *testing.T) {
	report := Explain(Input{RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		InputVersionKnown: true, ResultState: StateUnauthorized, InputValue: "2", UnitPrice: "300", Requester: RequesterGeneral})
	if report.Code != CodeUnauthorized {
		t.Fatalf("report=%+v want UNAUTHORIZED", report)
	}
	if report.InputValue != "" || report.UnitPrice != "" {
		t.Fatalf("report=%+v leaked value on unauthorized", report)
	}
}

func TestUnknownStateAndRequesterAreRejected(t *testing.T) {
	if report := Explain(Input{ResultState: "nonsense", Requester: admin()}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
	if report := Explain(Input{ResultState: StatePresent, Requester: "nobody"}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

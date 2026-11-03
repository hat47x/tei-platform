package derivationevidence

import (
	"strings"
	"testing"
)

func evidenceInput(state string, requester string) Input {
	return Input{
		RuleID: "line-amount", AcceptedRevision: "R", TargetID: "order-line.amount",
		Realization: "X", RealizationVersion: "1",
		InputID: "order-line", InputVersion: "7", InputVersionKnown: true,
		ResultState: state, InputValue: "2", UnitPrice: "300",
		Requester: requester,
	}
}

func contains(list []string, want string) bool {
	for _, value := range list {
		if value == want {
			return true
		}
	}
	return false
}

// A value is disclosed to an admin only for a present result. A missing, NULL,
// unauthorized, or failed result has no value to disclose.
func TestAdminDoesNotReceiveValuesForNonPresentStates(t *testing.T) {
	for _, state := range []string{
		StateInputMissing, StateNullDeclared, StateNullUndeclared,
		StateUnauthorized, StateComputeError, StateMultiplicityViolation,
		StateTypeMismatch, StateUnsupported,
	} {
		report := Explain(evidenceInput(state, RequesterAdmin))
		if report.InputValue != "" || report.UnitPrice != "" {
			t.Errorf("%s: report=%+v leaked a value", state, report)
		}
		if !contains(report.Redacted, "inputValue") || !contains(report.Redacted, "unitPrice") {
			t.Errorf("%s: redacted=%v", state, report.Redacted)
		}
	}
}

func TestAdminUnauthorizedStillRedactsRowExistence(t *testing.T) {
	report := Explain(evidenceInput(StateUnauthorized, RequesterAdmin))
	if !contains(report.Redacted, "rowExistence") {
		t.Fatalf("redacted=%v want rowExistence", report.Redacted)
	}
	if report.InputValue != "" || report.UnitPrice != "" {
		t.Fatalf("report=%+v leaked on unauthorized", report)
	}
}

func TestComputeErrorIsDistinctFromNullAndMissing(t *testing.T) {
	compute := Explain(evidenceInput(StateComputeError, RequesterAdmin))
	nullDeclared := Explain(evidenceInput(StateNullDeclared, RequesterAdmin))
	missing := Explain(evidenceInput(StateInputMissing, RequesterAdmin))
	codes := map[string]bool{compute.Code: true, nullDeclared.Code: true, missing.Code: true}
	if len(codes) != 3 {
		t.Fatalf("compute=%s null=%s missing=%s are not distinct", compute.Code, nullDeclared.Code, missing.Code)
	}
	if compute.Code != CodeComputeError {
		t.Fatalf("compute code=%s", compute.Code)
	}
}

func TestEveryDiagnosticStateHasItsOwnCode(t *testing.T) {
	states := []struct {
		state string
		code  string
	}{
		{StatePresent, CodePresent},
		{StateInputMissing, CodeInputMissing},
		{StateNullDeclared, CodeNullDeclared},
		{StateNullUndeclared, CodeNullUndeclared},
		{StateUnauthorized, CodeUnauthorized},
		{StateComputeError, CodeComputeError},
		{StateMultiplicityViolation, CodeMultiplicityViolation},
		{StateTypeMismatch, CodeTypeMismatch},
		{StateUnsupported, CodeUnsupported},
	}
	seen := map[string]bool{}
	for _, tc := range states {
		report := Explain(evidenceInput(tc.state, RequesterAdmin))
		if report.Code != tc.code {
			t.Errorf("%s: code=%s want %s", tc.state, report.Code, tc.code)
		}
		if seen[report.Code] {
			t.Errorf("code %s is reused", report.Code)
		}
		seen[report.Code] = true
	}
}

func TestMessagesDoNotCarrySensitiveValues(t *testing.T) {
	in := evidenceInput(StateComputeError, RequesterAdmin)
	in.InputValue = "secret-quantity"
	in.UnitPrice = "secret-price"
	report := Explain(in)
	if strings.Contains(report.Message, "secret-quantity") || strings.Contains(report.Message, "secret-price") {
		t.Fatalf("message leaked a value: %q", report.Message)
	}
	if strings.Contains(report.Message, "2") || strings.Contains(report.Message, "300") {
		t.Fatalf("message carried the input values: %q", report.Message)
	}
}

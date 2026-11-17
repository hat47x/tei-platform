package command_test

import (
	"strings"
	"testing"

	"tei/internal/command"
	"tei/internal/failure"
)

// TestReportedFailureNamesAPartialSuccess checks that the Command boundary reports
// "only part of the work was applied" under its own code.
//
// X-TEA's bulk update (F-14) could leave external calls applied after the database
// was rolled back, and the knowledge inventory records that it had no way to say
// "partly done". TEI had the meaning (failure.PartialError, produced by
// internal/execution's compensationFailure) but the boundary reported nothing at
// all: a bare PartialError is neither an ExecutionError nor effect-unknown.
func TestReportedFailureNamesAPartialSuccess(t *testing.T) {
	err := &failure.PartialError{
		Code:    "COMPENSATION_PARTIAL",
		Message: "2 of 3 compensations were applied",
	}
	code, message, ok := command.ReportedFailure(err)
	if !ok {
		t.Fatal("a partial success is not reported at all")
	}
	if code != "COMMAND_PARTIAL_SUCCESS" {
		t.Fatalf("code=%q", code)
	}
	// The provider's own reason survives in the message, so a reader can tell which
	// partial case it was.
	if !strings.Contains(message, "2 of 3 compensations were applied") {
		t.Fatalf("message=%q", message)
	}
}

// TestPartialSuccessIsReportedBeforeEffectUnknown checks the ordering decision.
//
// "Unknown" says nothing is known; "partial" says some effects certainly landed. The
// more informative name is reported first, so a caller does not have to ask the
// provider to learn what it could have been told. The retry policy is not relaxed by
// this: a partial result is still looked up before it is sent again, which the second
// half of this test pins.
func TestPartialSuccessIsReportedBeforeEffectUnknown(t *testing.T) {
	// An ExecutionError that is both effect-unknown and a partial success: the
	// partial name wins, because it carries more information.
	err := &command.ExecutionError{
		Code:          "PROVIDER_PARTIAL",
		Message:       "one of two writes landed",
		EffectUnknown: true,
		Cause: &failure.PartialError{
			Code:    "COMPENSATION_PARTIAL",
			Message: "1 of 2 compensations were applied",
		},
	}
	if !command.EffectMayBeUnknown(err) {
		t.Fatal("the fixture is not effect-unknown; the ordering test would prove nothing")
	}
	code, _, ok := command.ReportedFailure(err)
	if !ok || code != "COMMAND_PARTIAL_SUCCESS" {
		t.Fatalf("code=%q ok=%v", code, ok)
	}
	// The retry policy stays "look up first": the ordering must not invite a blind
	// resend of a request that partly applied.
	if err.Retryable() != failure.LookupFirst {
		t.Fatalf("retryable=%v, want lookup-first", err.Retryable())
	}
}

// TestEffectUnknownStillWinsWhenNothingIsPartial checks the other direction: an
// effect-unknown failure that is not a partial success keeps its own code, so the
// new branch does not swallow the existing rule.
func TestEffectUnknownStillWinsWhenNothingIsPartial(t *testing.T) {
	err := &command.ExecutionError{
		Code:          "COMMIT_FAILED",
		Message:       "commit did not confirm",
		EffectUnknown: true,
	}
	code, _, ok := command.ReportedFailure(err)
	if !ok || code != "COMMAND_EFFECT_UNKNOWN" {
		t.Fatalf("code=%q ok=%v", code, ok)
	}
}

// TestAPlainExecutionErrorKeepsItsCode checks that the ordinary path is untouched.
func TestAPlainExecutionErrorKeepsItsCode(t *testing.T) {
	err := &command.ExecutionError{Code: "COMMAND_UNAUTHORIZED", Message: "not allowed"}
	code, _, ok := command.ReportedFailure(err)
	if !ok || code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("code=%q ok=%v", code, ok)
	}
}

// TestAPartialErrorIsStillLookedUpBeforeResend checks the claim the ordering rests
// on: naming a partial success does not relax the retry policy. The error itself
// says "look up first", so a caller that reads the code and resends blindly is going
// against the error it was given, not following it.
func TestAPartialErrorIsStillLookedUpBeforeResend(t *testing.T) {
	err := &failure.PartialError{Code: "COMPENSATION_PARTIAL", Message: "1 of 2 applied"}
	if err.Retryable() != failure.LookupFirst {
		t.Fatalf("retryable=%v, want lookup-first", err.Retryable())
	}
	if !err.HasEffectUnknown() {
		t.Fatal("a partial success must still flag that effects may be applied")
	}
}

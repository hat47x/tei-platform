package studiohost

import (
	"errors"
	"testing"

	"tei/internal/command"
	"tei/internal/failure"
	"tei/internal/operations"
)

// TestCommandFailureCarriesUnifiedKind checks that the Studio bridge keeps the
// unified failure classification of the boundary it delegates to. A missing
// input is a business rejection the caller must not retry, while a failed
// commit is a technical failure whose effect state is unknown, so the caller
// must look the result up before resending with a new identity.
func TestCommandFailureCarriesUnifiedKind(t *testing.T) {
	host, err := NewReadOnly([]string{referenceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	// A missing input is a business rejection: the request is not permitted as
	// sent, and sending it again unchanged answers the same way.
	invalid := host.ExecuteCommand("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", nil)
	if invalid.OK || invalid.Failure == nil || invalid.Failure.Code != "INVALID_COMMAND_INPUT" {
		t.Fatalf("run=%+v failure=%+v", invalid, invalid.Failure)
	}
	if invalid.Failure.Kind != string(failure.BusinessRejection) {
		t.Fatalf("kind=%q, want %q", invalid.Failure.Kind, failure.BusinessRejection)
	}
	if invalid.Failure.Retryable != string(failure.NotRetryable) {
		t.Fatalf("retryable=%q, want %q", invalid.Failure.Retryable, failure.NotRetryable)
	}
	if invalid.Failure.EffectUnknown {
		t.Fatal("a rejected request was reported as having an unknown effect")
	}

	// An unauthorized capability is also a business rejection, not a fault of
	// the host.
	denied := host.ExecuteCommandWithPolicy("workspace-1", "order_entry", "order_entry.submit_order", "sales.order.submit", map[string]string{"order": "ORD-1"}, nil, []string{"sales.other"})
	if denied.OK || denied.Failure == nil || denied.Failure.Code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("run=%+v failure=%+v", denied, denied.Failure)
	}
	if denied.Failure.Kind != string(failure.BusinessRejection) {
		t.Fatalf("kind=%q, want %q", denied.Failure.Kind, failure.BusinessRejection)
	}
	if denied.Failure.Retryable != string(failure.NotRetryable) {
		t.Fatalf("retryable=%q, want %q", denied.Failure.Retryable, failure.NotRetryable)
	}
}

// TestCommandFailureFromCommitIsTechnical checks the technical side of the
// contract: a commit failure keeps the ExecutionError's LookupFirst policy and
// the effect-unknown warning, so the response does not read like a business
// rejection the caller may ignore.
func TestCommandFailureFromCommitIsTechnical(t *testing.T) {
	commitErr := &command.ExecutionError{
		Code:          "COMMAND_COMMIT_FAILED",
		Message:       "commit failed",
		Cause:         errors.New("storage unavailable"),
		EffectUnknown: true,
	}
	got := commandFailure("COMMAND_COMMIT_FAILED", "command", commitErr.Message, commitErr)
	if got.Kind != string(failure.TechnicalFailure) {
		t.Fatalf("kind=%q, want %q", got.Kind, failure.TechnicalFailure)
	}
	if got.Retryable != string(failure.LookupFirst) {
		t.Fatalf("retryable=%q, want %q", got.Retryable, failure.LookupFirst)
	}
	if !got.EffectUnknown {
		t.Fatal("a failed commit was reported as having a known effect")
	}
}

// TestOperationFailureIsTechnicalWithUnknownEffect checks that a host-side
// failure that carries no unified contract still reports the effect state it
// knows, and is classified as technical rather than as a business rejection.
func TestOperationFailureIsTechnicalWithUnknownEffect(t *testing.T) {
	loadErr := &operations.ValidationFailure{Stage: operations.StageLoad, Err: errors.New("cannot read")}
	got := operationFailure(loadErr)
	if got.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("code=%q", got.Code)
	}
	if got.Kind != string(failure.TechnicalFailure) {
		t.Fatalf("kind=%q, want %q", got.Kind, failure.TechnicalFailure)
	}
	if got.Retryable != string(failure.LookupFirst) {
		t.Fatalf("retryable=%q, want %q", got.Retryable, failure.LookupFirst)
	}
	if !got.EffectUnknown {
		t.Fatal("a failed load was reported as having a known effect")
	}
}

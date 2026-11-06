package main

import (
	"errors"
	"fmt"
	"testing"

	"tei/internal/command"
)

// TestCommandFailureDistinguishesUnknownEffect checks that a failure whose data
// effect may already be applied is reported as unknown, not as a clean
// rejection, and that a lifecycle refusal keeps its own code.
func TestCommandFailureDistinguishesUnknownEffect(t *testing.T) {
	unknown := command.NewUnknownExecutionError("PROVIDER_PARTIAL", "write may have been applied", nil)
	code, message, ok := commandFailure(unknown)
	if !ok || code != "COMMAND_EFFECT_UNKNOWN" || message == "" {
		t.Fatalf("code=%q message=%q ok=%v", code, message, ok)
	}

	plain := &command.ExecutionError{Code: "COMMAND_EXECUTION_FAILED", Message: "rejected"}
	if code, _, ok := commandFailure(plain); ok {
		t.Fatalf("a clean rejection was reported as %q", code)
	}

	refused := &command.ExecutionError{Code: "COMMAND_LIFECYCLE_REFUSED", Message: "already observed"}
	if code, _, ok := commandFailure(refused); !ok || code != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("lifecycle refusal code=%q ok=%v", code, ok)
	}

	wrapped := fmt.Errorf("commit failed: %w", command.NewUnknownExecutionError("COMMAND_COMMIT_FAILED", "commit may have applied", nil))
	if code, _, ok := commandFailure(wrapped); !ok || code != "COMMAND_EFFECT_UNKNOWN" {
		t.Fatalf("wrapped unknown effect code=%q ok=%v", code, ok)
	}

	if code, _, ok := commandFailure(errors.New("plain")); ok {
		t.Fatalf("a plain error was reported as %q", code)
	}
}

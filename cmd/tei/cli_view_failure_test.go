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

	// A clean rejection keeps its own code. This used to fall through to a
	// generic COMMAND_FAILED because the CLI carried a hand-written allow-list
	// that omitted it (#3807).
	plain := &command.ExecutionError{Code: "COMMAND_EXECUTION_FAILED", Message: "rejected"}
	if code, _, ok := commandFailure(plain); !ok || code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("a clean rejection: code=%q ok=%v", code, ok)
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

// TestCommandFailureNamesTheScopeCodes checks that a period split by a closing day
// keeps its own refusal code, so a caller can tell "supply a half-aware state"
// from "fix the business date" and from a generic execution failure.
func TestCommandFailureNamesTheScopeCodes(t *testing.T) {
	unavailable := &command.ExecutionError{Code: "COMMAND_PERIOD_SCOPE_UNAVAILABLE", Message: "the state cannot answer for a half month"}
	if code, message, ok := commandFailure(unavailable); !ok || code != "COMMAND_PERIOD_SCOPE_UNAVAILABLE" || message == "" {
		t.Fatalf("code=%q message=%q ok=%v", code, message, ok)
	}
	unknown := &command.ExecutionError{Code: "COMMAND_PERIOD_SCOPE_UNKNOWN", Message: "the business date is not named"}
	if code, _, ok := commandFailure(unknown); !ok || code != "COMMAND_PERIOD_SCOPE_UNKNOWN" {
		t.Fatalf("code=%q ok=%v", code, ok)
	}
}

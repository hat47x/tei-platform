package hook_test

import (
	"errors"
	"fmt"
	"testing"

	"tei/internal/failure"
	"tei/internal/hook"
)

func TestHookErrorImplementsFailure(t *testing.T) {
	hookErr := &hook.Error{
		Code:    "HOOK_CALL_REFUSED",
		Message: "refused by hook",
		Cause:   errors.New("boom"),
	}

	if hookErr.FailureCode() != "HOOK_CALL_REFUSED" {
		t.Errorf("FailureCode() = %q", hookErr.FailureCode())
	}
	if hookErr.FailureMessage() != "refused by hook" {
		t.Errorf("FailureMessage() = %q", hookErr.FailureMessage())
	}
	if hookErr.FailureCause() == nil {
		t.Error("FailureCause() is nil")
	} else if hookErr.FailureCause().Error() != "boom" {
		t.Errorf("FailureCause().Error() = %q", hookErr.FailureCause().Error())
	}
	if hookErr.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v", hookErr.Kind())
	}
	if hookErr.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", hookErr.Retryable())
	}
	if hookErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

func TestHookErrorAsFailure(t *testing.T) {
	hookErr := &hook.Error{
		Code:    "HOOK_IMPLEMENTATION_FAILED",
		Message: "boom",
	}

	f, ok := failure.AsFailure(hookErr)
	if !ok {
		t.Fatal("AsFailure() = false")
	}
	if f.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v", f.Kind())
	}
	if f.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", f.Retryable())
	}
	if f.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

func TestHookErrorWrappedAsFailure(t *testing.T) {
	hookErr := &hook.Error{
		Code:    "HOOK_PATTERN_NOT_FOUND",
		Message: "missing",
	}
	wrapped := fmt.Errorf("wrapped: %w", hookErr)

	f, ok := failure.AsFailure(wrapped)
	if !ok {
		t.Fatal("AsFailure(wrapped) = false")
	}
	if f.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v", f.Kind())
	}
	if f.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", f.Retryable())
	}
	if f.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

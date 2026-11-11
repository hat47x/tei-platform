package command_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"tei/internal/command"
	"tei/internal/failure"
)

func TestExecutionErrorImplementsFailure(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:          "COMMAND_EXECUTION_FAILED",
		Message:       "executor boom",
		Cause:         errors.New("boom"),
		EffectUnknown: true,
	}

	if execErr.FailureCode() != "COMMAND_EXECUTION_FAILED" {
		t.Errorf("FailureCode() = %q", execErr.FailureCode())
	}
	if execErr.FailureMessage() != "executor boom" {
		t.Errorf("FailureMessage() = %q", execErr.FailureMessage())
	}
	if execErr.FailureCause() == nil {
		t.Error("FailureCause() is nil")
	} else if execErr.FailureCause().Error() != "boom" {
		t.Errorf("FailureCause().Error() = %q", execErr.FailureCause().Error())
	}
	if execErr.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if !execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false")
	}
}

func TestExecutionErrorBusinessRejection(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:    "COMMAND_PERIOD_CLOSED",
		Message: "period is closed",
	}

	if execErr.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

func TestExecutionErrorPartialSuccess(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:          "PROVIDER_PARTIAL",
		Message:       "partial write",
		EffectUnknown: true,
	}

	if execErr.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if !execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false")
	}
}

func TestExecutionErrorAsFailure(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:    "COMMAND_UNAUTHORIZED",
		Message: "not allowed",
	}

	f, ok := failure.AsFailure(execErr)
	if !ok {
		t.Fatal("AsFailure() = false")
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

func TestExecutionErrorWrappedAsFailure(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:          "COMMAND_COMMIT_FAILED",
		Message:       "commit failed",
		EffectUnknown: true,
	}
	wrapped := fmt.Errorf("wrapped: %w", execErr)

	f, ok := failure.AsFailure(wrapped)
	if !ok {
		t.Fatal("AsFailure(wrapped) = false")
	}
	if f.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v", f.Kind())
	}
	if f.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v", f.Retryable())
	}
	if !f.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false")
	}
}

func TestExecutionErrorBusinessDenied(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:    "COMMAND_BUSINESS_DENIED",
		Message: "business rule refused",
	}

	if execErr.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

func TestExecutionErrorResourceConflict(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:    "COMMAND_RESOURCE_CONFLICT",
		Message: "resource version changed",
	}

	if execErr.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = true")
	}
}

func TestWrapTimeout(t *testing.T) {
	execErr := &command.ExecutionError{
		Code:          "TIMEOUT",
		Message:       "operation timed out: context deadline exceeded",
		Cause:         context.DeadlineExceeded,
		EffectUnknown: true,
	}
	if execErr.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v", execErr.Kind())
	}
	if execErr.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v", execErr.Retryable())
	}
	if !execErr.HasEffectUnknown() {
		t.Error("HasEffectUnknown() = false")
	}
}

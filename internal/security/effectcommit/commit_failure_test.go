package effectcommit

import (
	"errors"
	"testing"

	"tei/internal/failure"
)

// CommitUncertainError is the one failure whose effect state is always unknown,
// so it must reach a caller that only knows the unified failure contract with
// both the warning and the lookup-first policy. A blind retry with a new
// OperationID would duplicate the effect, which is exactly what the contract
// exists to prevent.
func TestCommitUncertainErrorImplementsFailure(t *testing.T) {
	err := &CommitUncertainError{Cause: errors.New("receipt could not be confirmed")}

	unified, ok := failure.AsFailure(err)
	if !ok {
		t.Fatal("AsFailure did not recognize CommitUncertainError")
	}
	if unified.FailureCode() != "COMMIT_UNCERTAIN" {
		t.Fatalf("FailureCode() = %q", unified.FailureCode())
	}
	if unified.FailureCause() == nil || unified.FailureCause().Error() != "receipt could not be confirmed" {
		t.Fatalf("FailureCause() = %v", unified.FailureCause())
	}
	if unified.Kind() != failure.TechnicalFailure {
		t.Fatalf("Kind() = %v, want %v", unified.Kind(), failure.TechnicalFailure)
	}
	if unified.Retryable() != failure.LookupFirst {
		t.Fatalf("Retryable() = %v, want %v", unified.Retryable(), failure.LookupFirst)
	}
	if !unified.HasEffectUnknown() {
		t.Fatal("HasEffectUnknown() = false for an uncertain commit")
	}

	// The same answers must survive wrapping, because a Host reports the
	// failure through its own error chain.
	wrapped := errors.Join(errors.New("host failed"), err)
	if !failure.IsTechnicalFailure(wrapped) {
		t.Fatal("a wrapped uncertain commit was not reported as a technical failure")
	}
	if !failure.EffectMayBeUnknown(wrapped) {
		t.Fatal("a wrapped uncertain commit lost the effect-unknown warning")
	}
	if !failure.IsRetryable(wrapped) {
		t.Fatal("a wrapped uncertain commit was reported as not retryable at all")
	}
	if got := wrapped.(interface{ Unwrap() []error }).Unwrap(); len(got) != 2 {
		t.Fatalf("Unwrap() = %v", got)
	}
}

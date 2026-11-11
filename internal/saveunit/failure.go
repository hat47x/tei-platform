package saveunit

import (
	"errors"

	"tei/internal/failure"
)

// Reference-store failure codes. They name a store condition the caller can
// act on, not a fault of the host.
const (
	// ReferenceRecordExists means a create found the key already stored.
	ReferenceRecordExists = "RECORD_EXISTS"
	// ReferenceRecordMissing means an update or delete found no stored row.
	ReferenceRecordMissing = "RECORD_MISSING"
)

// ReferenceFailure is the reference (in-memory) store's refusal for one entity
// change. It is a business rejection: the requested change does not hold
// against the stored state, and sending the same change again answers the same
// way. It carries no effect, because the change was not applied.
type ReferenceFailure struct {
	Code    string
	Message string
}

func (e *ReferenceFailure) Error() string { return e.Code + ": " + e.Message }

// FailureCode returns the machine-readable failure code.
func (e *ReferenceFailure) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *ReferenceFailure) FailureMessage() string { return e.Message }

// FailureCause returns nil: the reference store reports the condition itself
// and does not wrap another error.
func (e *ReferenceFailure) FailureCause() error { return nil }

// Kind classifies the business meaning of the failure.
func (e *ReferenceFailure) Kind() failure.Kind { return failure.BusinessRejection }

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure. The caller must change the request, so the
// answer is no.
func (e *ReferenceFailure) Retryable() failure.RetryPolicy { return failure.NotRetryable }

// HasEffectUnknown reports whether the failure may have left partial effects
// applied.
func (e *ReferenceFailure) HasEffectUnknown() bool { return false }

var _ failure.Failure = (*ReferenceFailure)(nil)

// FailureCode returns the machine-readable failure code.
func (e *SaveError) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *SaveError) FailureMessage() string { return e.Message }

// FailureCause returns the underlying error, if any.
func (e *SaveError) FailureCause() error { return e.Cause }

// Kind classifies the business meaning of the failure.
//
// The boundary distinguishes a refusal the save unit itself carries (an empty
// unit, a dependency cycle, a store that already holds the key) from a fault of
// the storage boundary (a transaction that could not begin or commit, a
// transient provider failure). When an apply failure wraps a cause, the cause
// decides: a wrapped ReferenceFailure keeps the store's business rejection
// instead of being reported as a host fault.
func (e *SaveError) Kind() failure.Kind {
	switch e.Code {
	case "SAVE_UNIT_EMPTY", "SAVE_STORE_UNAVAILABLE", "SAVE_CYCLE", "SAVE_UNKNOWN_KIND":
		return failure.BusinessRejection
	case "SAVE_APPLY_FAILED":
		if e.Cause != nil && failure.IsBusinessRejection(e.Cause) {
			return failure.BusinessRejection
		}
		return failure.TechnicalFailure
	default:
		return failure.TechnicalFailure
	}
}

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure. Only a transient provider failure may be sent
// again unchanged. A commit failure may have applied the effect, so the caller
// must look the result up before resending with a new identity.
func (e *SaveError) Retryable() failure.RetryPolicy {
	if e.EffectUnknown {
		return failure.LookupFirst
	}
	if e.Code == "SAVE_PROVIDER_TRANSIENT" {
		return failure.Retryable
	}
	return failure.NotRetryable
}

// HasEffectUnknown reports whether the failure may have left partial effects
// applied.
func (e *SaveError) HasEffectUnknown() bool { return e.EffectUnknown }

var _ failure.Failure = (*SaveError)(nil)

// FailureCode returns the machine-readable failure code.
func (e *HookFailure) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *HookFailure) FailureMessage() string { return e.Message }

// FailureCause returns the underlying error, if any.
func (e *HookFailure) FailureCause() error { return e.Cause }

// Kind classifies the business meaning of the failure. A hook that denies the
// save unit refuses it for a rule the unit itself carries, which is a business
// rejection rather than a fault of the host.
func (e *HookFailure) Kind() failure.Kind { return failure.BusinessRejection }

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure. The denial stays until the unit changes.
func (e *HookFailure) Retryable() failure.RetryPolicy { return failure.NotRetryable }

// HasEffectUnknown reports whether the failure may have left partial effects
// applied. A denied save applies nothing.
func (e *HookFailure) HasEffectUnknown() bool { return false }

var _ failure.Failure = (*HookFailure)(nil)

// SaveFailureCause reports the cause of a save failure, following the unified
// contract. It returns nil for an error that carries no cause.
func SaveFailureCause(err error) error {
	var saveErr *SaveError
	if errors.As(err, &saveErr) {
		return saveErr.Cause
	}
	return nil
}
